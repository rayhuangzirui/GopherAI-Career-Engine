package analyzer

import "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"

type Analyzer interface {
	AnalyzeResume(input model.ResumeAnalysisInput) (model.ResumeAnalysisResult, error)
}
