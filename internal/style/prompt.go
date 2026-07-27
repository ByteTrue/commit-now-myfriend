package style

import (
	"fmt"
	"strings"
)

type Style string

const (
	Auto        Style = "auto"
	Conventional Style = "conventional"
	Angular     Style = "angular"
	Google      Style = "google"
	Atom        Style = "atom"
	Plain       Style = "plain"
	Custom      Style = "custom"
)

var All = []Style{Auto, Conventional, Angular, Google, Atom, Plain, Custom}

const defaultMaxSubjectLen = 72

func (s Style) Valid() bool {
	for _, v := range All {
		if s == v {
			return true
		}
	}
	return false
}

func (s Style) Instruction(maxLen int) string {
	if maxLen <= 0 {
		maxLen = defaultMaxSubjectLen
	}
	switch s {
	case Auto:
		return "Infer the commit message style from repository context when possible. " +
			"If the input does not provide enough history/context to infer a style, use Conventional Commits: type(scope)?: subject with an optional body."
	case Conventional:
		return "Use Conventional Commits: type(scope)?: subject with an optional body separated by one blank line."
	case Angular:
		return "Use Angular commit format: type(scope): subject. " +
			"Use a lowercase type such as build, chore, ci, docs, feat, fix, perf, refactor, revert, style, or test. " +
			"Use an imperative, lowercase subject without a trailing period."
	case Google:
		return "Use Google-style commit message guidance: a short, specific, imperative subject line with no trailing period. " +
			"After a blank line, include a body only when useful to explain what changed and why."
	case Atom:
		return "Use Atom-style commit messages: a concise imperative subject line, optionally followed by a body with supporting details. " +
			"Do not require a Conventional Commit type prefix unless it is clearly natural for the repository."
	case Plain:
		return "Use a plain, concise natural-language commit message. Do not require a type prefix or strict format."
	case Custom:
		return ""
	default:
		return fmt.Sprintf("Use %s style. Keep the subject line at or below %d characters when possible.", s, maxLen)
	}
}

type CommitInput struct {
	Diff        string
	Files       []FileInfo
	Repo        *RepoInfo
	Style       Style
	CustomPrompt string
	MaxSubjectLen int
}

type FileInfo struct {
	Path   string
	Status string
}

type RepoInfo struct {
	Root   string
	Branch string
	Remote string
}

func BuildPrompt(input CommitInput) (system, user string) {
	maxLen := input.MaxSubjectLen
	if maxLen <= 0 {
		maxLen = defaultMaxSubjectLen
	}

	var parts []string

	parts = append(parts,
		"You write git commit messages from staged changes.",
		"Return only the commit message, with no markdown, labels, explanation, or surrounding quotes.",
		"Output the final commit message directly; do not include reasoning or analysis.",
		fmt.Sprintf("Keep the subject line at or below %d characters when possible.", maxLen),
		"Do not invent details that are not supported by the diff.",
	)

	if inst := input.Style.Instruction(maxLen); inst != "" {
		parts = append(parts, inst)
	}

	if cp := strings.TrimSpace(input.CustomPrompt); cp != "" {
		parts = append(parts, "Additional user guidance:\n"+cp)
	}

	system = strings.Join(parts, "\n")
	user = buildUserPrompt(input)
	return
}

func buildUserPrompt(input CommitInput) string {
	var parts []string

	if input.Repo != nil {
		var repoLines []string
		if input.Repo.Root != "" {
			repoLines = append(repoLines, "root: "+input.Repo.Root)
		}
		if input.Repo.Branch != "" {
			repoLines = append(repoLines, "branch: "+input.Repo.Branch)
		}
		if input.Repo.Remote != "" {
			repoLines = append(repoLines, "remote: "+input.Repo.Remote)
		}
		if len(repoLines) > 0 {
			parts = append(parts, "Repository metadata:\n"+strings.Join(repoLines, "\n"))
		}
	}

	if len(input.Files) > 0 {
		var fileLines []string
		for _, f := range input.Files {
			if f.Status != "" {
				fileLines = append(fileLines, fmt.Sprintf("- %s %s", f.Status, f.Path))
			} else {
				fileLines = append(fileLines, "- "+f.Path)
			}
		}
		parts = append(parts, "Changed files:\n"+strings.Join(fileLines, "\n"))
	}

	parts = append(parts, "Diff:\n"+input.Diff)

	return strings.Join(parts, "\n\n")
}