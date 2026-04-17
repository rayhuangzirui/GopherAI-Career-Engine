package analyzer

import (
	"strings"
	"unicode"
)

type SanitizedInput struct {
	CleanText 	string
	Truncated 	bool
	Suspicious 	bool
	FlagReasons []string
}

func SanitizeLLMText(raw string, maxChars int) SanitizedInput {
	s := raw

	// remove NULs and normalize newlines/tabs
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", " ")

	// remove most control chars except newline
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		if c == '\n' || !unicode.IsControl(c) {
			b.WriteRune(c)
		}
	}
	s = b.String()

	// collapse excessive blank lines
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	// collapse multiple spaces
	s = strings.Join(strings.Fields(s), " ")

	out := SanitizedInput{
		CleanText: s,
		Truncated: false,
		Suspicious: false,
		FlagReasons: nil,
	}

	lower := strings.ToLower(s)
	suspiciousPatterns := []string{
		"ignore previous instructions",
		"ignore all previous instructions",
		"system prompt",
		"developer message",
		"you are a helpful assistant",
		"assistant",
		"act as",
		"do not follow",
		"return exactly",
		"tool call",
		"function call",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lower, pattern) {
			out.Suspicious = true
			out.FlagReasons = append(out.FlagReasons, "suspicious pattern: "+pattern)
		}
	}

	if maxChars > 0 && len(out.CleanText) > maxChars {
		out.CleanText = out.CleanText[:maxChars]
		out.Truncated = true
	}

	return out
}

func BuildDelimitedResumeJDInput(resume SanitizedInput, jd SanitizedInput) string {
	var meta []string
	if resume.Truncated {
		meta = append(meta, "resume_truncated=true")
	}
	if jd.Truncated {
		meta = append(meta, "jd_truncated=true")
	}
	if resume.Suspicious {
		meta = append(meta, "resume_suspicious=true")
	}
	if jd.Suspicious {
		meta = append(meta, "jd_suspicious=true")
	}

	metaLine := "none"
	if len(meta) > 0 {
		metaLine = strings.Join(meta, ",")
	}

	return "Input metadata: " + metaLine + "\n\n" +
		 "Resume:\n<<<RESUME_TEXT_START>>>\n" + resume.CleanText + "\n<<<RESUME_TEXT_END>>>\n\n" +
		"Job Description:\n<<<JD_TEXT_START>>>\n" + jd.CleanText + "\n<<<JD_TEXT_END>>>\n"
}
