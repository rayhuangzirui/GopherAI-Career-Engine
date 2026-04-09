package analyzer

import (
	"errors"
	"strings"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
)

type MockAnalyzer struct{}

func NewMockAnalyzer() *MockAnalyzer {
	return &MockAnalyzer{}
}

func (a *MockAnalyzer) AnalyzeResume(input model.ResumeAnalysisInput) (model.ResumeAnalysisResult, error) {
	if strings.Contains(input.ResumeText, "FAIL_ANALYSIS") {
		return model.ResumeAnalysisResult{}, errors.New("mock analyzer failed: simulated analysis error")
	}
	return model.ResumeAnalysisResult{
		SkillSummary: []string{
			"Go",
			"MySQL",
			"Redis",
			"RabbitMQ",
			"Docker",
			"REST APIs",
		},
		ExperienceSummary: []string{
			"Backend Developer at XYZ Corp (2020-2023)",
			"Software Engineer at ABC Inc (2018-2020)",
		},
		MissingKeywords: []string{
			"Kubernetes",
			"AWS",
		},
		Suggestions: []string{
			"Add measurable backend impact",
			"Highlight distributed systems experience",
		},
	}, nil
}

func (a *MockAnalyzer) MatchResumeJD(input model.ResumeJDMatchInput) (model.ResumeJDMatchResult, error) {
	if strings.Contains(input.ResumeText, "FAIL_ANALYSIS") ||
		strings.Contains(input.JobDescriptionText, "FAIL_ANALYSIS") {
		return model.ResumeJDMatchResult{}, errors.New("mock matcher failed: simulated analysis error")
	}

	return model.ResumeJDMatchResult{
		MatchScore: 80,
		MatchedKeywords: []string{
			"Go",
			"Docker",
			"REST APIs",
		},
		MissingKeywords: []string{
			"Kubernetes",
			"AWS",
		},
		Suggestions: []string{
			"Add more distributed system impact",
			"Mention cloud deployment experience",
		},
	}, nil
}
