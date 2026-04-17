package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/config"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
)

type LLMAnalyzer struct {
	cfg      *config.Config
	client   *http.Client
	fallback Analyzer
}

func NewLLMAnalyzer(cfg *config.Config, fallback Analyzer) *LLMAnalyzer {
	timeout := time.Duration(cfg.LLMTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	return &LLMAnalyzer{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
		fallback: fallback,
	}
}

func (a *LLMAnalyzer) AnalyzeResume(input model.ResumeAnalysisInput) (model.ResumeAnalysisResult, error) {
	if a.fallback == nil {
		return model.ResumeAnalysisResult{}, fmt.Errorf("llm analyzer is not configured for AnalyzeResume")
	}
	return a.fallback.AnalyzeResume(input)
}

func (a *LLMAnalyzer) MatchResumeJD(input model.ResumeJDMatchInput) (model.ResumeJDMatchResult, error) {
	log.Printf("llm_analyzer: MatchResumeJD started, provider=%s model=%s", a.cfg.LLMProvider, a.cfg.LLMModel)
	if strings.Contains(input.ResumeText, "FAIL_ANALYSIS") ||
		strings.Contains(input.JobDescriptionText, "FAIL_ANALYSIS") {
		return model.ResumeJDMatchResult{}, fmt.Errorf("llm analyzer failed: simulated analysis error")
	}

	if a.cfg.LLMAPIKey == "" {
		return model.ResumeJDMatchResult{}, fmt.Errorf("missing LLM_API_KEY")
	}

	resume := SanitizeLLMText(input.ResumeText, a.cfg.LLMMaxInputChars)
	jd := SanitizeLLMText(input.JobDescriptionText, a.cfg.LLMMaxInputChars)
	delimitedInput := BuildDelimitedResumeJDInput(resume, jd)
	systemPrompt, userPrompt := buildResumeJDMatchPrompts(delimitedInput)

	log.Printf("llm_analyzer: calling chat completions, resume_len=%d jd_len=%d", len(resume.CleanText), len(jd.CleanText))
	raw, err := a.callChatCompletions(systemPrompt, userPrompt)

	if err != nil {
		log.Printf("llm_analyzer: fallback to rules analyzer because chat completion failed: %v", err)
		if a.fallback != nil {
			fallbackResult, fbErr := a.fallback.MatchResumeJD(input)
			if fbErr != nil {
				return model.ResumeJDMatchResult{}, fmt.Errorf("llm failed: %v; fallback failed: %w", err, fbErr)
			}
			fallbackResult.Source = "rules_fallback"
			return fallbackResult, nil
		}

		return model.ResumeJDMatchResult{}, err
	}

	log.Printf("llm_analyzer: raw response received, len=%d", len(raw))

	parsed, err := ParseAndValidateResumeJDMatchResponse(raw)
	if err != nil {
		log.Printf("llm_analyzer: fallback to rules analyzer because response parsing failed: %v", err)
		log.Printf("llm_analyzer: raw response: %s", raw)
		if a.fallback != nil {
			fallbackResult, fbErr := a.fallback.MatchResumeJD(input)
			if fbErr != nil {
				return model.ResumeJDMatchResult{}, fmt.Errorf("llm failed: %v; fallback failed: %w", err, fbErr)
			}
			fallbackResult.Source = "rules_fallback"
			return fallbackResult, nil
		}

		return model.ResumeJDMatchResult{}, err
	}

	log.Printf("llm_analyzer: response parsed and validated successfully")

	result := BuildValidatedResumeJDMatchResult(parsed)
	result.Source = "llm"
	return result, nil
}

type opeAICompatChatRequest struct {
	Model          string                      `json:"model"`
	Messages       []openAICompatMessage       `json:"messages"`
	Temperature    float64                     `json:"temperature"`
	MaxTokens      int                         `json:"max_tokens"`
	ResponseFormat *openAICompatResponseFormat `json:"response_format"`
}

type openAICompatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompatResponseFormat struct {
	Type string `json:"type"`
}

type opeAICompatChatResponse struct {
	Choices []struct {
		Message openAICompatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *LLMAnalyzer) callChatCompletions(systemPrompt, userPrompt string) (string, error) {
	baseURL := strings.TrimRight(a.cfg.LLMBaseURL, "/")
	url := baseURL + "/chat/completions"

	reqBody := opeAICompatChatRequest{
		Model: a.cfg.LLMModel,
		Messages: []openAICompatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: a.cfg.LLMTemperature,
		MaxTokens:   a.cfg.LLMMaxOutputTokens,
		ResponseFormat: &openAICompatResponseFormat{
			Type: "json_object",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.cfg.LLMTimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create llm request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.LLMAPIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call llm api: %w", err)
	}

	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var parsed opeAICompatChatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal llm response envelope: %w", err)
	}

	if parsed.Error != nil {
		return "", fmt.Errorf("llm api returned error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm response has no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("llm response has empty content")
	}

	return content, nil
}
