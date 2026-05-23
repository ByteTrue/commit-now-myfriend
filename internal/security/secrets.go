package security

import (
	"regexp"
	"strings"
)

type SecretSeverity string

const (
	SecretSeverityWarning  SecretSeverity = "warning"
	SecretSeverityBlocking SecretSeverity = "blocking"
)

type SecretFinding struct {
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Line        int            `json:"line"`
	Excerpt     string         `json:"excerpt"`
	Severity    SecretSeverity `json:"severity"`
}

type SecretScanResult struct {
	Findings            []SecretFinding `json:"findings"`
	HasBlockingFindings bool            `json:"hasBlockingFindings"`
}

type secretPattern struct {
	code        string
	description string
	expression  *regexp.Regexp
	severity    SecretSeverity
}

var secretPatterns = []secretPattern{
	{
		code:        "private_key",
		description: "Private key material appears in the diff.",
		expression:  regexp.MustCompile(`(?i)-----BEGIN (?:RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY-----`),
		severity:    SecretSeverityWarning,
	},
	{
		code:        "aws_access_key",
		description: "AWS access key-like token appears in the diff.",
		expression:  regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
		severity:    SecretSeverityWarning,
	},
	{
		code:        "provider_api_key",
		description: "Provider API key-like token appears in the diff.",
		expression:  regexp.MustCompile(`(?i)\b(?:sk|xox[baprs]|ghp|github_pat)_[A-Za-z0-9_\-]{16,}\b`),
		severity:    SecretSeverityWarning,
	},
	{
		code:        "assigned_secret",
		description: "Secret-looking assignment appears in the diff.",
		expression:  regexp.MustCompile(`(?i)(?:api[_-]?key|secret|token|password)\s*[:=]\s*["'][^"'\s]{12,}["']`),
		severity:    SecretSeverityWarning,
	},
}

var excerptRedactor = regexp.MustCompile(`(["'=:\s])([A-Za-z0-9_\-/+]{8})[A-Za-z0-9_\-/+]{4,}(["'\s]|$)`)

func ScanTextForSecrets(text string) SecretScanResult {
	findings := make([]SecretFinding, 0)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	for index, line := range lines {
		for _, pattern := range secretPatterns {
			if pattern.expression.MatchString(line) {
				findings = append(findings, SecretFinding{
					Code:        pattern.code,
					Description: pattern.description,
					Line:        index + 1,
					Excerpt:     redactExcerpt(line),
					Severity:    pattern.severity,
				})
			}
		}
	}

	result := SecretScanResult{Findings: findings}
	for _, finding := range findings {
		if finding.Severity == SecretSeverityBlocking {
			result.HasBlockingFindings = true
			break
		}
	}

	return result
}

func redactExcerpt(line string) string {
	redacted := excerptRedactor.ReplaceAllString(line, `$1$2…$3`)
	runes := []rune(redacted)
	if len(runes) > 160 {
		return string(runes[:160])
	}
	return redacted
}
