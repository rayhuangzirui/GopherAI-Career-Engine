package analyzer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
)

type llmMatchResponse struct {
	MatchedKeywords          []string `json:"matched_keywords"`
	MissingKeywords          []string `json:"missing_keywords"`
	ExperienceEvidence       []string `json:"experience_evidence"`
	Suggestions              []string `json:"suggestions"`
	SemanticAlignmentSummary string   `json:"semantic_alignment_summary"`
}

func ParseAndValidateResumeJDMatchResponse(raw string) (*llmMatchResponse, error) {
	var temp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &temp); err != nil {
		return nil, fmt.Errorf("llm response is not valid json: %w", err)
	}

	allowed := map[string]bool{
		"matched_keywords":           true,
		"missing_keywords":           true,
		"experience_evidence":        true,
		"suggestions":                true,
		"semantic_alignment_summary": true,
	}

	for key := range temp {
		if !allowed[key] {
			return nil, fmt.Errorf("llm response contains unsupported field: %s", key)
		}
	}

	var parsed llmMatchResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse llm response schema: %w", err)
	}

	parsed.MatchedKeywords = normalizeStringList(parsed.MatchedKeywords, 20, 60)
	parsed.MissingKeywords = normalizeStringList(parsed.MissingKeywords, 20, 60)
	parsed.ExperienceEvidence = normalizeStringList(parsed.ExperienceEvidence, 6, 180)
	parsed.Suggestions = normalizeStringList(parsed.Suggestions, 5, 180)
	parsed.SemanticAlignmentSummary = cleanSingleText(parsed.SemanticAlignmentSummary, 300)

	return &parsed, nil
}

func BuildValidatedResumeJDMatchResult(parsed *llmMatchResponse) model.ResumeJDMatchResult {
	score := computeRuleBoundedMatchScore(parsed.MatchedKeywords, parsed.MissingKeywords)

	suggestions := make([]string, 0, len(parsed.Suggestions)+1)
	suggestions = append(suggestions, parsed.Suggestions...)

	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return model.ResumeJDMatchResult{
		MatchScore:               score,
		MatchedKeywords:          parsed.MatchedKeywords,
		MissingKeywords:          parsed.MissingKeywords,
		Suggestions:              suggestions,
		SemanticAlignmentSummary: parsed.SemanticAlignmentSummary,
		Source:                   "llm",
	}
}

func computeRuleBoundedMatchScore(matched, missing []string) int {
	m := len(matched)
	x := len(missing)
	total := m + x

	if total == 0 {
		return 50
	}

	score := (m * 100) / total

	if m <= 2 && score > 65 {
		score = 65
	}

	if m >= 5 {
		score += 5
	}

	if x >= 6 {
		score -= 5
	}

	if score < 0 {
		score = 0
	}

	if score > 100 {
		score = 100
	}

	return score

}

func normalizeStringList(items []string, maxItems int, maxItemLen int) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(items))

	for _, item := range items {
		cleaned := cleanSingleText(item, maxItemLen)
		if cleaned == "" {
			continue
		}

		if containsDisallowedContent(cleaned) {
			continue
		}

		key := strings.ToLower(cleaned)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, cleaned)

		if len(out) >= maxItems {
			break
		}
	}

	sort.Strings(out)
	return out
}

func cleanSingleText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.Join(strings.Fields(s), " ")

	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimSpace(s)
	}

	return s
}

func containsDisallowedContent(s string) bool {
	lower := strings.ToLower(s)

	disallowed := []string{
		"<script",
		"</script",
		"<html",
		"you are hired",
		"guaranteed interview",
		"guaranteed job",
		"visa advice",
		"legal advice",
		"medical advice",
	}

	for _, p := range disallowed {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}
