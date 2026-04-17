package analyzer

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
)

type RulesAnalyzer struct{}

func NewRulesAnalyzer() *RulesAnalyzer {
	return &RulesAnalyzer{}
}

var keywordAliases = map[string][]string{
	"go":                  {"go", "golang"},
	"python":              {"python"},
	"java":                {"java"},
	"c/c++":               {"c", "c++"},
	"javascript":          {"js", "javascript"},
	"typescript":          {"ts", "typescript"},
	"mysql":               {"mysql"},
	"postgresql":          {"postgres", "postgresql", "psql"},
	"redis":               {"redis"},
	"rabbitmq":            {"rabbitmq"},
	"kafka":               {"kafka"},
	"docker":              {"docker"},
	"kubernetes":          {"k8s", "kubernetes"},
	"aws":                 {"aws", "amazon web services"},
	"gcp":                 {"gcp", "google cloud platform"},
	"azure":               {"azure"},
	"rest api":            {"rest", "restful api", "rest api"},
	"grpc":                {"grpc"},
	"microservices":       {"microservice", "microservices"},
	"distributed systems": {"distributed system", "distributed systems"},
	"sql":                 {"sql"},
	"nosql":               {"nosql"},
	"message queue":       {"message queue", "message queues", "mq"},
	"ci/cd":               {"ci/cd", "cicd", "continuous integration/continuous delivery"},
	"git":                 {"git"},
	"testing":             {"testing", "unit test", "unit tests", "integration test", "integration tests"},
}

var backendCoreKeywords = []string{
	"go",
	"python",
	"java",
	"c/c++",
	"javascript",
	"typescript",
	"mysql",
	"docker",
	"distributed systems",
	"aws",
	"sql",
	"nosql",
	"message queue",
	"ci/cd",
	"git",
	"testing",
}

var jdWeightedKeywords = map[string]int{
	"go":                  10,
	"python":              10,
	"java":                10,
	"c/c++":               10,
	"mysql":               8,
	"postgresql":          8,
	"redis":               8,
	"rabbitmq":            8,
	"kafka":               8,
	"docker":              10,
	"kubernetes":          12,
	"aws":                 10,
	"azure":               8,
	"rest api":            8,
	"grpc":                8,
	"microservices":       10,
	"distributed systems": 14,
	"sql":                 6,
	"nosql":               6,
	"message queue":       8,
	"ci/cd":               6,
	"git":                 6,
	"testing":             6,
}

func (a *RulesAnalyzer) AnalyzeResume(input model.ResumeAnalysisInput) (model.ResumeAnalysisResult, error) {
	if strings.Contains(input.ResumeText, "FAIL_ANALYSIS") {
		return model.ResumeAnalysisResult{}, errors.New("rules analyzer failed: simulated analysis error")
	}

	normalized := normalizeText(input.ResumeText)
	detected := detectKeywords(normalized)
	skills := orderedIntersection(detected, backendCoreKeywords)

	if len(skills) == 0 {
		skills = topKeywords(detected, 6)
	}

	experienceSummary := buildExperienceSummary(normalized, detected)
	missingKeywords := detectMissingResumeKeywords(detected)
	suggestions := buildResumeSuggestions(detected, missingKeywords)

	return model.ResumeAnalysisResult{
		SkillSummary:      skills,
		ExperienceSummary: experienceSummary,
		MissingKeywords:   missingKeywords,
		Suggestions:       suggestions,
		Source:            "rules",
	}, nil
}

func (a *RulesAnalyzer) MatchResumeJD(input model.ResumeJDMatchInput) (model.ResumeJDMatchResult, error) {
	if strings.Contains(input.ResumeText, "FAIL_ANALYSIS") ||
		strings.Contains(input.JobDescriptionText, "FAIL_ANALYSIS") {
		return model.ResumeJDMatchResult{}, errors.New("rules analyzer failed: simulated analysis error")
	}

	resumeNorm := normalizeText(input.ResumeText)
	jdNorm := normalizeText(input.JobDescriptionText)
	resumeKeywords := detectKeywords(resumeNorm)
	jdKeywords := detectKeywords(jdNorm)

	requiredKeywords := orderedJDKeywords(jdKeywords)
	matched := make([]string, 0)
	missing := make([]string, 0)

	totalWeight := 0
	matchedWeight := 0

	for _, kw := range requiredKeywords {
		weight := keywordWeight(kw)
		totalWeight += weight

		if resumeKeywords[kw] {
			matched = append(matched, kw)
			matchedWeight += weight
		} else {
			missing = append(missing, kw)
		}
	}

	score := 0
	if totalWeight > 0 {
		score = matchedWeight * 100 / totalWeight
	} else {
		shared := countSharedKeywords(resumeKeywords, jdKeywords)
		score = min(shared*15, 85)
	}

	score = adjustScore(score, resumeKeywords, jdKeywords)

	suggestions := buildMatchSuggestions(missing, resumeKeywords)
	return model.ResumeJDMatchResult{
		MatchScore:      score,
		MatchedKeywords: matched,
		MissingKeywords: missing,
		Suggestions:     suggestions,
		Source:          "rules",
	}, nil
}

func normalizeText(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	re := regexp.MustCompile(`[^a-z0-9+#./ ]+`)
	s = re.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func detectKeywords(text string) map[string]bool {
	found := make(map[string]bool)

	for canonical, aliases := range keywordAliases {
		for _, alias := range aliases {
			if containsToken(text, alias) {
				found[canonical] = true
				break
			}
		}
	}

	return found
}

func containsToken(text, phrase string) bool {
	if phrase == "" {
		return false
	}

	return strings.Contains(" "+text+" ", " "+phrase+" ")
}

func orderedIntersection(found map[string]bool, preferred []string) []string {
	out := make([]string, 0)
	for _, kw := range preferred {
		if found[kw] {
			out = append(out, kw)
		}
	}
	return out
}

func topKeywords(found map[string]bool, limit int) []string {
	out := make([]string, 0, len(found))
	for kw := range found {
		out = append(out, kw)
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildExperienceSummary(text string, detected map[string]bool) []string {
	summary := make([]string, 0)

	if detected["go"] || detected["java"] || detected["python"] {
		summary = append(summary, "Shows backend programming language experience relevant to service development")
	}
	if detected["mysql"] || detected["postgresql"] || detected["sql"] {
		summary = append(summary, "Demonstrates relational database experience for application data handling")
	}
	if detected["redis"] || detected["nosql"] {
		summary = append(summary, "Includes cache or non-relational data layer experience")
	}
	if detected["rabbitmq"] || detected["kafka"] || detected["message queue"] {
		summary = append(summary, "Indicates asynchronous processing or message-driven system exposure")
	}
	if detected["docker"] || detected["kubernetes"] || detected["aws"] {
		summary = append(summary, "Suggests deployment or infrastructure familiarity beyond basic application code")
	}
	if detected["testing"] || detected["ci/cd"] {
		summary = append(summary, "Shows evidence of engineering workflow maturity through testing or delivery practices")
	}

	if len(summary) == 0 {
		summary = append(summary, "Resume shows limited explicit backend infrastructure keywords")
	}
	return summary
}

func detectMissingResumeKeywords(detected map[string]bool) []string {
	candidates := []string{
		"aws",
		"docker",
		"kubernetes",
		"microservices",
		"distributed systems",
		"rest api",
		"grpc",
		"sql",
		"nosql",
		"message queue",
		"ci/cd",
	}

	missing := make([]string, 0)
	for _, kw := range candidates {
		if !detected[kw] {
			missing = append(missing, kw)
		}
	}
	return missing
}

func buildResumeSuggestions(detected map[string]bool, missing []string) []string {
	suggestions := make([]string, 0)

	if !detected["distributed systems"] {
		suggestions = append(suggestions, "Add a concrete example of concurrency, async pipelines, or multi-worker task processing")
	}
	if !detected["aws"] && !detected["gcp"] && !detected["azure"] {
		suggestions = append(suggestions, "Mention cloud deployment or infrastructure experience if available")
	}
	if !detected["testing"] {
		suggestions = append(suggestions, "Highlight unit or integration testing to strengthen engineering credibility")
	}
	if !detected["kubernetes"] && detected["docker"] {
		suggestions = append(suggestions, "If applicable, connect Docker experience to container orchestration or production deployment workflows")
	}
	if len(suggestions) == 0 && len(missing) == 0 {
		suggestions = append(suggestions, "Resume already shows a balanced backend profile; focus next on measurable impact and scale")
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Strengthen the resume with more measurable backend impact and project scale details")
	}

	return suggestions
}
func orderedJDKeywords(found map[string]bool) []string {
	out := make([]string, 0)
	for kw := range found {
		if _, ok := jdWeightedKeywords[kw]; ok {
			out = append(out, kw)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		wi := keywordWeight(out[i])
		wj := keywordWeight(out[j])
		if wi == wj {
			return out[i] < out[j]
		}
		return wi > wj
	})

	return out
}

func keywordWeight(kw string) int {
	if w, ok := jdWeightedKeywords[kw]; ok {
		return w
	}
	return 5
}

func countSharedKeywords(a, b map[string]bool) int {
	count := 0
	for kw := range b {
		if a[kw] {
			count++
		}
	}
	return count
}

func adjustScore(score int, resumeKeywords, jdKeywords map[string]bool) int {
	if resumeKeywords["go"] && jdKeywords["go"] {
		score += 5
	}
	if resumeKeywords["docker"] && jdKeywords["docker"] {
		score += 4
	}
	if resumeKeywords["aws"] && jdKeywords["aws"] {
		score += 5
	}
	if resumeKeywords["distributed systems"] && jdKeywords["distributed systems"] {
		score += 6
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func buildMatchSuggestions(missing []string, resumeKeywords map[string]bool) []string {
	suggestions := make([]string, 0)

	for _, kw := range missing {
		switch kw {
		case "aws":
			suggestions = append(suggestions, "Add AWS-related deployment, storage, or infrastructure work if you have it")
		case "kubernetes":
			suggestions = append(suggestions, "Include container orchestration experience if relevant to your projects")
		case "distributed systems":
			suggestions = append(suggestions, "Show distributed-system thinking through queues, retries, idempotency, or multi-worker design")
		case "rabbitmq", "kafka", "message queue":
			suggestions = append(suggestions, "Emphasize async processing and message queue usage for backend workloads")
		case "testing":
			suggestions = append(suggestions, "Mention test coverage, integration testing, or validation strategy")
		default:
			suggestions = append(suggestions, "Strengthen evidence for "+kw+" with a concrete project example")
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Focus on measurable impact, scale, and ownership to improve an already solid match")
	}

	if len(suggestions) > 4 {
		suggestions = suggestions[:4]
	}

	return suggestions
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
