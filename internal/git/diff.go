package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/style"
)

const MaxDiffBytes = 60_000

func StagedDiff() (diff string, files []style.FileInfo, err error) {
	diff, err = runGit("diff", "--cached", "--unified=3", "--stat")
	if err != nil || diff == "" {
		diff, err = runGit("diff", "--unified=3", "HEAD", "--stat")
		if err != nil || diff == "" {
			return "", nil, fmt.Errorf("no changes to commit")
		}
	}
	// Separate stat from diff
	files = parseStat(diff)
	diff = stripStat(diff)
	diff = truncate(diff)
	return
}

func RepoInfo() *style.RepoInfo {
	root, _ := runGit("rev-parse", "--show-toplevel")
	branch, _ := runGit("rev-parse", "--abbrev-ref", "HEAD")
	remote, _ := runGit("remote", "get-url", "origin")
	return &style.RepoInfo{
		Root:   root,
		Branch: branch,
		Remote: remote,
	}
}

func runGit(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func parseStat(diffWithStat string) []style.FileInfo {
	// The stat is at the end of `git diff --stat` output.
	// Look for lines like: " file.go | 10 +++---"
	var files []style.FileInfo
	lines := strings.Split(diffWithStat, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "|") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		path := strings.TrimSpace(parts[0])
		if path == "" || strings.Contains(path, "...") {
			continue
		}
		files = append(files, style.FileInfo{Path: path, Status: "modified"})
	}
	return files
}

func stripStat(diffWithStat string) string {
	// Remove the trailing stat section
	lines := strings.Split(diffWithStat, "\n")
	var cleanLines []string
	inStat := false
	for _, line := range lines {
		if inStat {
			// Still in stat section — check if it's a file stat line
			if strings.TrimSpace(line) == "" || strings.Contains(line, "|") {
				continue
			}
			inStat = false
		}
		if strings.Contains(line, " changed, ") && strings.Contains(line, " insertion") {
			inStat = true
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	return strings.Join(cleanLines, "\n")
}

func truncate(diff string) string {
	if len(diff) > MaxDiffBytes {
		return diff[:MaxDiffBytes] + "\n... (diff truncated)"
	}
	return diff
}
