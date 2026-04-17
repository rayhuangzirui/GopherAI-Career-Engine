package model

type ResumeAnalysisInput struct {
	ResumeText string `json:"resume_text"`
}

type JDAnalysisInput struct {
	JobDescriptionText string `json:"job_description_text"`
}

type ResumeJDMatchInput struct {
	ResumeText         string `json:"resume_text"`
	JobDescriptionText string `json:"job_description_text"`
}

type ResumeAnalysisResult struct {
	SkillSummary      []string `json:"skill_summary"`
	ExperienceSummary []string `json:"experience_summary"`
	MissingKeywords   []string `json:"missing_keywords"`
	Suggestions       []string `json:"suggestions"`
	Source          	string   `json:"source,omitempty"`
}


type ResumeJDMatchResult struct {
	MatchScore      int      `json:"match_score"`
	MatchedKeywords []string `json:"matched_keywords"`
	MissingKeywords []string `json:"missing_keywords"`
	Suggestions     []string `json:"suggestions"`
	SemanticAlignmentSummary string `json:"semantic_alignment_summary,omitempty"`
	Source          string   `json:"source,omitempty"`
}
