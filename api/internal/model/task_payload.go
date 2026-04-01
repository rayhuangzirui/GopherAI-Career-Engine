package model

type ResumeAnalysisInput struct {
	ResumeText string `json:"resumeText"`
}

type JDAnalysisInput struct {
	JobDescriptionText string `json:"jobDescriptionText"`
}

type ResumeJDMatchInput struct {
	ResumeText         string `json:"resumeText"`
	JobDescriptionText string `json:"jobDescriptionText"`
}

type ResumeAnalysisResult struct {
	SkillSummary      []string `json:"skillSummary"`
	ExperienceSummary []string `json:"experienceSummary"`
	MissingKeywords   []string `json:"missingKeywords"`
	Suggestions       []string `json:"suggestions"`
}

type JDAnalysisResult struct {
	KeyRequirements []string `json:"keyRequirements"`
	PreferredSkills []string `json:"preferredSkills"`
	Summary         string   `json:"summary"`
}

type ResumeJDMatchResult struct {
	MatchScore      int      `json:"matchScore"`
	MatchedKeywords []string `json:"matchedKeywords"`
	MissingKeywords []string `json:"missingKeywords"`
	Suggestions     []string `json:"suggestions"`
}
