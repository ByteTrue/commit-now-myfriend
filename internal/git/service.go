package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/security"
)

const DefaultMaxDiffBytes = 200_000
const DefaultMaxReadBytes = 80_000

func DefaultCommandRunner(cwd string, args []string, env map[string]string) (CommandResult, error) {
	command := exec.Command("git", args...)
	command.Dir = cwd
	command.Env = mergeEnv(env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	result := CommandResult{
		Stdout: strings.TrimRight(stdout.String(), "\n"),
		Stderr: strings.TrimRight(stderr.String(), "\n"),
	}

	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitError *exec.ExitError
	if ok := asExitError(err, &exitError); ok {
		result.ExitCode = exitError.ExitCode()
		return result, nil
	}

	result.ExitCode = 1
	if result.Stderr == "" {
		result.Stderr = err.Error()
	}
	return result, nil
}

func InspectRepository(options InspectOptions) (Inspection, error) {
	maxDiffBytes := options.MaxDiffBytes
	if maxDiffBytes <= 0 {
		maxDiffBytes = DefaultMaxDiffBytes
	}
	gitRunner := options.GitRunner
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}

	repository := detectRepository(options.CWD, options.Env, gitRunner)
	if !repository.IsRepository || repository.IsBare {
		emptyDiff, metadata := truncateDiff("", maxDiffBytes)
		files := []FileStatus{}
		issues := buildIssues(repository, files, metadata)
		warnings, blockingIssues := splitIssues(issues)
		return Inspection{
			Repository:         repository,
			Files:              files,
			StagedFiles:        []FileStatus{},
			UnstagedFiles:      []FileStatus{},
			UntrackedFiles:     []FileStatus{},
			StagedDiff:         emptyDiff,
			Diff:               metadata,
			SecretScan:         security.ScanTextForSecrets(emptyDiff),
			Issues:             issues,
			Warnings:           warnings,
			BlockingIssues:     blockingIssues,
			HasStagedChanges:   false,
			HasUnstagedChanges: false,
			HasUntrackedFiles:  false,
		}, nil
	}

	statusResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}, options.Env))
	numstatResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"diff", "--cached", "--numstat", "-z", "--find-renames"}, options.Env))
	diffResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"diff", "--cached", "--patch", "--binary", "--find-renames"}, options.Env))

	commandIssues := make([]Issue, 0, 3)
	if statusResult.ExitCode != 0 {
		commandIssues = append(commandIssues, createIssue("git_status_failed", formatGitFailure("git status --porcelain", statusResult.Stderr, statusResult.Stdout), IssueSeverityBlocking))
	}
	if numstatResult.ExitCode != 0 {
		commandIssues = append(commandIssues, createIssue("git_diff_numstat_failed", formatGitFailure("git diff --cached --numstat", numstatResult.Stderr, numstatResult.Stdout), IssueSeverityBlocking))
	}
	if diffResult.ExitCode != 0 {
		commandIssues = append(commandIssues, createIssue("git_diff_patch_failed", formatGitFailure("git diff --cached --patch", diffResult.Stderr, diffResult.Stdout), IssueSeverityBlocking))
	}

	files := applyBinaryMetadata(parsePorcelainStatus(statusResult.Stdout), numstatResult.Stdout)
	stagedFiles := filterFiles(files, func(file FileStatus) bool { return file.Staged != nil })
	unstagedFiles := filterFiles(files, func(file FileStatus) bool { return file.Unstaged != nil })
	untrackedFiles := filterFiles(files, func(file FileStatus) bool { return file.Untracked })
	truncatedDiff, diffMetadata := truncateDiff(diffResult.Stdout, maxDiffBytes)
	secretScan := security.ScanTextForSecrets(diffResult.Stdout)
	issues := append(buildIssues(repository, files, diffMetadata), commandIssues...)
	if len(secretScan.Findings) > 0 {
		issues = append(issues, createIssue("secret_scan_match", "Potential secrets were found in inspected diff content.", IssueSeverityWarning))
	}
	warnings, blockingIssues := splitIssues(issues)

	return Inspection{
		Repository:         repository,
		Files:              files,
		StagedFiles:        stagedFiles,
		UnstagedFiles:      unstagedFiles,
		UntrackedFiles:     untrackedFiles,
		StagedDiff:         truncatedDiff,
		Diff:               diffMetadata,
		SecretScan:         secretScan,
		Issues:             issues,
		Warnings:           warnings,
		BlockingIssues:     blockingIssues,
		HasStagedChanges:   len(stagedFiles) > 0,
		HasUnstagedChanges: len(unstagedFiles) > 0,
		HasUntrackedFiles:  len(untrackedFiles) > 0,
	}, nil
}

func InspectCommitScope(options CommitScopeOptions) (CommitScope, error) {
	maxDiffBytes := options.MaxDiffBytes
	if maxDiffBytes <= 0 {
		maxDiffBytes = DefaultMaxDiffBytes
	}
	maxReadBytes := options.MaxReadBytes
	if maxReadBytes <= 0 {
		maxReadBytes = DefaultMaxReadBytes
	}
	contextPolicy := ContextPolicy{Mode: ContextPolicyModeBounded, FileReadsAllowed: true}
	if options.DiffOnly {
		contextPolicy = ContextPolicy{Mode: ContextPolicyModeDiffOnly, FileReadsAllowed: false}
	}
	gitRunner := options.GitRunner
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}

	statusArgs := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}
	if len(options.Pathspecs) > 0 {
		statusArgs = append(statusArgs, "--")
		statusArgs = append(statusArgs, options.Pathspecs...)
	}
	statusResult := normalizeRunnerResult(gitRunner(options.CWD, statusArgs, options.Env))
	if statusResult.ExitCode != 0 {
		return CommitScope{}, fmt.Errorf("%s", formatGitFailure("git status --porcelain", statusResult.Stderr, statusResult.Stdout))
	}

	files := parsePorcelainStatus(statusResult.Stdout)
	cachedNumstatResult := normalizeRunnerResult(gitRunner(options.CWD, scopedGitArgs([]string{"diff", "--cached", "--numstat", "-z", "--find-renames"}, options.Pathspecs), options.Env))
	worktreeNumstatResult := normalizeRunnerResult(gitRunner(options.CWD, scopedGitArgs([]string{"diff", "--numstat", "-z", "--find-renames"}, options.Pathspecs), options.Env))
	if cachedNumstatResult.ExitCode == 0 {
		files = applyBinaryMetadata(files, cachedNumstatResult.Stdout)
	}
	if worktreeNumstatResult.ExitCode == 0 {
		files = applyBinaryMetadata(files, worktreeNumstatResult.Stdout)
	}
	files = applyWorkingTreeBinaryMetadata(options.CWD, files)
	diffResult := normalizeRunnerResult(gitRunner(options.CWD, scopedGitArgs([]string{"diff", "--patch", "--binary", "--find-renames"}, options.Pathspecs), options.Env))
	_, diffMetadata := truncateDiff("", maxDiffBytes)
	if diffResult.ExitCode == 0 {
		_, diffMetadata = truncateDiff(diffResult.Stdout, maxDiffBytes)
	}
	headResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"rev-parse", "--verify", "HEAD"}, options.Env))
	var head *string
	if headResult.ExitCode == 0 && strings.TrimSpace(headResult.Stdout) != "" {
		head = stringPtr(strings.TrimSpace(headResult.Stdout))
	}
	stagedFiles := filterFiles(files, func(file FileStatus) bool { return file.Staged != nil })
	selected := make([]FileStatus, 0, len(files))
	for _, file := range files {
		if options.StagedOnly {
			if file.Staged != nil {
				selected = append(selected, file)
			}
			continue
		}
		if file.Untracked && !options.IncludeUntracked {
			continue
		}
		if file.Staged != nil || file.Unstaged != nil || file.Untracked {
			selected = append(selected, file)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Path < selected[j].Path })
	readUsage := BudgetUsage{MaxBytes: maxReadBytes}
	secretBlockers := []ScopedSecretFinding{}
	if contextPolicy.FileReadsAllowed {
		var usedReadBytes int
		secretBlockers, usedReadBytes = scanScopeSecrets(options.CWD, selected, maxReadBytes)
		readUsage.UsedBytes = usedReadBytes
	}
	exposure := buildAIExposureSummary(selected, secretBlockers, diffMetadata, readUsage, options.PreferenceSources)

	return CommitScope{
		Files:              selected,
		Pathspecs:          append([]string{}, options.Pathspecs...),
		StagedOnly:         options.StagedOnly,
		IncludesUntracked:  options.IncludeUntracked,
		HasSelectedChanges: len(selected) > 0,
		IndexSnapshot:      &IndexSnapshot{Head: head, StagedFiles: stagedFiles},
		SecretBlockers:     secretBlockers,
		ContextPolicy:      contextPolicy,
		AIExposure:         exposure,
	}, nil
}

func scanScopeSecrets(cwd string, files []FileStatus, maxReadBytes int) ([]ScopedSecretFinding, int) {
	findings := make([]ScopedSecretFinding, 0)
	usedReadBytes := 0
	for _, file := range files {
		if file.Binary || file.Staged != nil && file.Unstaged == nil && !file.Untracked {
			// Staged-only content is scanned through staged diffs later; this baseline
			// scanner handles working tree files for the new Commit Scope path.
			continue
		}
		content, err := os.ReadFile(filepath.Join(cwd, file.Path))
		if err != nil || bytes.IndexByte(content, 0) >= 0 {
			continue
		}
		if maxReadBytes > 0 && usedReadBytes+len(content) > maxReadBytes {
			continue
		}
		usedReadBytes += len(content)
		scan := security.ScanTextForSecrets(string(content))
		for _, finding := range scan.Findings {
			findings = append(findings, ScopedSecretFinding{
				Path:        file.Path,
				Code:        finding.Code,
				Description: finding.Description,
				Line:        finding.Line,
				Excerpt:     finding.Excerpt,
				Severity:    finding.Severity,
			})
		}
	}
	return findings, usedReadBytes
}

func buildAIExposureSummary(files []FileStatus, secretBlockers []ScopedSecretFinding, diffMetadata DiffMetadata, readUsage BudgetUsage, preferenceSources PreferenceExposure) AIExposureSummary {
	opaqueChanges := make([]OpaqueChangeSummary, 0)
	visibleFiles := make([]ProviderVisibleFile, 0, len(files))
	for _, file := range files {
		visible := ProviderVisibleFile{Path: file.Path, Source: "diff", Opaque: file.Binary}
		if file.Untracked {
			visible.Source = "metadata"
		}
		if file.Binary {
			visible.Source = "metadata"
		}
		visibleFiles = append(visibleFiles, visible)
		if file.Binary {
			opaqueChanges = append(opaqueChanges, OpaqueChangeSummary{Path: file.Path, Reason: "binary"})
		}
	}
	return AIExposureSummary{
		SelectedFileCount:    len(files),
		OpaqueChangeCount:    len(opaqueChanges),
		SecretBlockerCount:   len(secretBlockers),
		ProviderVisibleFiles: visibleFiles,
		PreferenceSources:    preferenceSources,
		DiffBudget: BudgetUsage{
			MaxBytes:      diffMetadata.MaxBytes,
			UsedBytes:     diffMetadata.Bytes,
			OriginalBytes: diffMetadata.OriginalBytes,
			Truncated:     diffMetadata.Truncated,
			OmittedBytes:  diffMetadata.OmittedBytes,
		},
		ReadBudget:    readUsage,
		OpaqueChanges: opaqueChanges,
	}
}

func StageAllChanges(options StageAllChangesOptions) (StageAllChangesResult, error) {
	inspectOptions := InspectOptions{
		CWD:          options.CWD,
		MaxDiffBytes: options.MaxDiffBytes,
		Env:          options.Env,
		GitRunner:    options.GitRunner,
	}

	if !options.Confirmed {
		inspection, err := InspectRepository(inspectOptions)
		return StageAllChangesResult{Staged: false, Reason: "not_confirmed", Inspection: inspection}, err
	}

	if !options.IsTTY {
		inspection, err := InspectRepository(inspectOptions)
		return StageAllChangesResult{Staged: false, Reason: "non_tty", Inspection: inspection}, err
	}

	gitRunner := options.GitRunner
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}
	addResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"add", "-A"}, options.Env))
	if addResult.ExitCode != 0 {
		inspection, err := InspectRepository(inspectOptions)
		inspection = addBlockingIssue(inspection, createIssue("git_add_failed", formatGitFailure("git add -A", addResult.Stderr, addResult.Stdout), IssueSeverityBlocking))
		return StageAllChangesResult{Staged: false, Reason: "git_add_failed", Inspection: inspection}, err
	}

	inspection, err := InspectRepository(inspectOptions)
	return StageAllChangesResult{Staged: true, Reason: "confirmed", Inspection: inspection}, err
}

func GetRecentCommits(cwd string, env map[string]string, limit int, gitRunner CommandRunner) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}
	result := normalizeRunnerResult(gitRunner(cwd, []string{"log", fmt.Sprintf("--max-count=%d", limit), "--pretty=format:%s%n%b", "--no-merges"}, env))
	if result.ExitCode != 0 {
		return []string{}, nil
	}

	commits := make([]string, 0)
	for _, commit := range strings.Split(result.Stdout, "\n\n") {
		trimmed := strings.TrimSpace(commit)
		if trimmed != "" {
			commits = append(commits, trimmed)
		}
	}
	if len(commits) > limit {
		commits = commits[:limit]
	}
	return commits, nil
}

func CommitWithMessage(cwd, message string, env map[string]string, gitRunner CommandRunner) (CommandResult, error) {
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}

	tempDir, err := os.MkdirTemp("", "cnm-commit-")
	if err != nil {
		return CommandResult{ExitCode: 1, Stderr: err.Error()}, nil
	}
	defer os.RemoveAll(tempDir)

	messagePath := filepath.Join(tempDir, "COMMIT_EDITMSG")
	if err := os.WriteFile(messagePath, []byte(strings.TrimSpace(message)+"\n"), 0o600); err != nil {
		return CommandResult{ExitCode: 1, Stderr: err.Error()}, nil
	}

	return normalizeRunnerResult(gitRunner(cwd, []string{"commit", "-F", messagePath}, env)), nil
}

func CommitScopeWithMessage(options CommitScopeCommitOptions) (CommitScopeCommitResult, error) {
	gitRunner := options.GitRunner
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}
	if strings.TrimSpace(options.Message) == "" {
		return CommitScopeCommitResult{Git: CommandResult{ExitCode: 1, Stderr: "commit message is required"}}, nil
	}
	if len(options.Scope.Files) == 0 {
		return CommitScopeCommitResult{Git: CommandResult{ExitCode: 1, Stderr: "no selected changes"}}, nil
	}

	addArgs := []string{"add", "--"}
	for _, file := range options.Scope.Files {
		addArgs = append(addArgs, file.Path)
	}
	addResult := normalizeRunnerResult(gitRunner(options.CWD, addArgs, options.Env))
	if addResult.ExitCode != 0 {
		return CommitScopeCommitResult{Message: options.Message, Git: addResult}, nil
	}

	commitResult, err := commitWithMessageOptions(options.CWD, options.Message, options.Env, gitRunner, options.NoVerify)
	if err != nil {
		return CommitScopeCommitResult{}, err
	}
	if commitResult.ExitCode != 0 {
		return CommitScopeCommitResult{Message: options.Message, Git: commitResult}, nil
	}
	headResult := normalizeRunnerResult(gitRunner(options.CWD, []string{"rev-parse", "HEAD"}, options.Env))
	hash := strings.TrimSpace(headResult.Stdout)
	return CommitScopeCommitResult{Hash: hash, Message: options.Message, Git: commitResult}, nil
}

func CaptureCommitTransactionSnapshot(cwd string, env map[string]string, gitRunner CommandRunner) (CommitTransactionSnapshot, error) {
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}
	headResult := normalizeRunnerResult(gitRunner(cwd, []string{"rev-parse", "HEAD"}, env))
	if headResult.ExitCode != 0 {
		return CommitTransactionSnapshot{}, fmt.Errorf("%s", formatGitFailure("git rev-parse HEAD", headResult.Stderr, headResult.Stdout))
	}
	statusResult := normalizeRunnerResult(gitRunner(cwd, []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}, env))
	if statusResult.ExitCode != 0 {
		return CommitTransactionSnapshot{}, fmt.Errorf("%s", formatGitFailure("git status --porcelain", statusResult.Stderr, statusResult.Stdout))
	}
	return CommitTransactionSnapshot{Head: strings.TrimSpace(headResult.Stdout), Status: statusResult.Stdout}, nil
}

func RollbackCommitTransaction(cwd string, env map[string]string, snapshot CommitTransactionSnapshot, gitRunner CommandRunner) CommitTransactionRollbackResult {
	if gitRunner == nil {
		gitRunner = DefaultCommandRunner
	}
	if strings.TrimSpace(snapshot.Head) == "" {
		return CommitTransactionRollbackResult{RolledBack: false, Status: "unsafe", Message: "missing transaction start head"}
	}
	resetResult := normalizeRunnerResult(gitRunner(cwd, []string{"reset", "--mixed", snapshot.Head}, env))
	if resetResult.ExitCode != 0 {
		return CommitTransactionRollbackResult{RolledBack: false, Status: "failed", Message: firstNonEmpty(resetResult.Stderr, resetResult.Stdout, "git reset failed")}
	}
	statusResult := normalizeRunnerResult(gitRunner(cwd, []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}, env))
	if statusResult.ExitCode != 0 {
		return CommitTransactionRollbackResult{RolledBack: true, Status: "rolled_back", Message: "rollback status verification failed"}
	}
	if statusResult.Stdout != snapshot.Status {
		return CommitTransactionRollbackResult{RolledBack: true, Status: "rolled_back_with_status_change", Message: "working tree state differs from transaction start"}
	}
	return CommitTransactionRollbackResult{RolledBack: true, Status: "rolled_back"}
}

func commitWithMessageOptions(cwd, message string, env map[string]string, gitRunner CommandRunner, noVerify bool) (CommandResult, error) {
	tempDir, err := os.MkdirTemp("", "cnm-commit-")
	if err != nil {
		return CommandResult{ExitCode: 1, Stderr: err.Error()}, nil
	}
	defer os.RemoveAll(tempDir)

	messagePath := filepath.Join(tempDir, "COMMIT_EDITMSG")
	if err := os.WriteFile(messagePath, []byte(strings.TrimSpace(message)+"\n"), 0o600); err != nil {
		return CommandResult{ExitCode: 1, Stderr: err.Error()}, nil
	}

	args := []string{"commit", "-F", messagePath}
	if noVerify {
		args = append(args, "--no-verify")
	}
	return normalizeRunnerResult(gitRunner(cwd, args, env)), nil
}

func detectRepository(cwd string, env map[string]string, gitRunner CommandRunner) RepositoryState {
	bareResult := normalizeRunnerResult(gitRunner(cwd, []string{"rev-parse", "--is-bare-repository"}, env))
	if bareResult.ExitCode != 0 {
		return RepositoryState{}
	}

	isBare := strings.TrimSpace(bareResult.Stdout) == "true"
	rootResult := normalizeRunnerResult(gitRunner(cwd, []string{"rev-parse", "--show-toplevel"}, env))
	gitDirResult := normalizeRunnerResult(gitRunner(cwd, []string{"rev-parse", "--git-dir"}, env))
	headResult := normalizeRunnerResult(gitRunner(cwd, []string{"rev-parse", "--verify", "HEAD"}, env))
	branchResult := normalizeRunnerResult(gitRunner(cwd, []string{"symbolic-ref", "--quiet", "--short", "HEAD"}, env))
	nameResult := normalizeRunnerResult(gitRunner(cwd, []string{"config", "--get", "user.name"}, env))
	emailResult := normalizeRunnerResult(gitRunner(cwd, []string{"config", "--get", "user.email"}, env))

	rootPath := stringPtr(cwd)
	if trimmed := strings.TrimSpace(rootResult.Stdout); rootResult.ExitCode == 0 && trimmed != "" {
		rootPath = stringPtr(trimmed)
	}

	var gitDirPath *string
	if trimmed := strings.TrimSpace(gitDirResult.Stdout); gitDirResult.ExitCode == 0 && trimmed != "" {
		base := cwd
		if rootPath != nil {
			base = *rootPath
		}
		gitDirPath = stringPtr(toAbsolutePath(base, trimmed))
	}

	var branchName *string
	if trimmed := strings.TrimSpace(branchResult.Stdout); branchResult.ExitCode == 0 && trimmed != "" {
		branchName = stringPtr(trimmed)
	}

	var name *string
	if trimmed := strings.TrimSpace(nameResult.Stdout); nameResult.ExitCode == 0 && trimmed != "" {
		name = stringPtr(trimmed)
	}

	var email *string
	if trimmed := strings.TrimSpace(emailResult.Stdout); emailResult.ExitCode == 0 && trimmed != "" {
		email = stringPtr(trimmed)
	}

	repository := RepositoryState{
		IsRepository:         true,
		RootPath:             rootPath,
		GitDirPath:           gitDirPath,
		IsBare:               isBare,
		IsInitialCommit:      headResult.ExitCode != 0,
		IsDetachedHead:       headResult.ExitCode == 0 && branchResult.ExitCode != 0,
		BranchName:           branchName,
		MergeInProgress:      false,
		RebaseInProgress:     false,
		CherryPickInProgress: false,
		HasGitIdentity:       name != nil && email != nil,
	}
	repository.GitIdentity.Name = name
	repository.GitIdentity.Email = email

	if gitDirPath != nil {
		repository.MergeInProgress = fileExists(filepath.Join(*gitDirPath, "MERGE_HEAD"))
		repository.RebaseInProgress = fileExists(filepath.Join(*gitDirPath, "rebase-merge")) || fileExists(filepath.Join(*gitDirPath, "rebase-apply"))
		repository.CherryPickInProgress = fileExists(filepath.Join(*gitDirPath, "CHERRY_PICK_HEAD"))
	}

	return repository
}

func buildIssues(repository RepositoryState, files []FileStatus, diff DiffMetadata) []Issue {
	issues := make([]Issue, 0)
	if !repository.IsRepository {
		issues = append(issues, createIssue("not_git_repository", "Current directory is not inside a git repository.", IssueSeverityBlocking))
		return issues
	}
	if repository.IsBare {
		issues = append(issues, createIssue("bare_repository", "Bare git repositories are not supported by this workflow.", IssueSeverityBlocking))
	}
	if repository.MergeInProgress {
		issues = append(issues, createIssue("merge_in_progress", "A merge is in progress; resolve it before generating a commit.", IssueSeverityBlocking))
	}
	if repository.RebaseInProgress {
		issues = append(issues, createIssue("rebase_in_progress", "A rebase is in progress; resolve it before generating a commit.", IssueSeverityBlocking))
	}
	if repository.CherryPickInProgress {
		issues = append(issues, createIssue("cherry_pick_in_progress", "A cherry-pick is in progress; resolve it before generating a commit.", IssueSeverityBlocking))
	}
	if repository.IsDetachedHead {
		issues = append(issues, createIssue("detached_head", "Repository is on a detached HEAD; committing here may be hard to find later.", IssueSeverityWarning))
	}
	if !repository.HasGitIdentity {
		issues = append(issues, createIssue("git_identity_missing", "Git user.name and user.email must be configured before committing.", IssueSeverityBlocking))
	}
	for _, file := range files {
		if file.Unstaged != nil {
			issues = append(issues, createIssue("unstaged_changes_present", "Unstaged tracked changes are present in the working tree.", IssueSeverityWarning))
			break
		}
	}
	for _, file := range files {
		if file.Untracked {
			issues = append(issues, createIssue("untracked_files_present", "Untracked non-ignored files are present in the working tree.", IssueSeverityWarning))
			break
		}
	}
	if diff.Truncated {
		issues = append(issues, createIssue("diff_truncated", "The inspected diff exceeded the configured size limit and was truncated.", IssueSeverityWarning))
	}
	return issues
}

func parsePorcelainStatus(output string) []FileStatus {
	if output == "" {
		return []FileStatus{}
	}
	entries := splitNullSeparated(output)
	files := map[string]FileStatus{}

	for index := 0; index < len(entries); index++ {
		entry := entries[index]
		if entry == "" {
			continue
		}
		x := byte(' ')
		y := byte(' ')
		if len(entry) > 0 {
			x = entry[0]
		}
		if len(entry) > 1 {
			y = entry[1]
		}
		rawPath := ""
		if len(entry) > 3 {
			rawPath = entry[3:]
		}

		if x == '?' && y == '?' {
			mergeFileStatus(files, FileStatus{Path: rawPath, Untracked: true, Binary: false})
			continue
		}

		if (x == 'R' || x == 'C') && index+1 < len(entries) {
			originalPath := entries[index+1]
			index++
			mergeFileStatus(files, FileStatus{
				Path:         rawPath,
				OriginalPath: stringPtr(originalPath),
				Staged:       changeKind(x),
				Unstaged:     changeKind(y),
				Binary:       false,
			})
			continue
		}

		mergeFileStatus(files, FileStatus{
			Path:      rawPath,
			Staged:    changeKind(x),
			Unstaged:  changeKind(y),
			Binary:    false,
			Untracked: false,
		})
	}

	result := make([]FileStatus, 0, len(files))
	for _, file := range files {
		result = append(result, file)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func applyBinaryMetadata(files []FileStatus, numstatOutput string) []FileStatus {
	if numstatOutput == "" {
		return files
	}
	binaryPaths := map[string]bool{}
	for _, entry := range splitNullSeparated(numstatOutput) {
		parts := strings.Split(entry, "\t")
		if len(parts) >= 3 && parts[0] == "-" && parts[1] == "-" && parts[2] != "" {
			binaryPaths[parts[2]] = true
		}
	}
	result := make([]FileStatus, 0, len(files))
	for _, file := range files {
		if binaryPaths[file.Path] {
			file.Binary = true
		}
		result = append(result, file)
	}
	return result
}

func applyWorkingTreeBinaryMetadata(cwd string, files []FileStatus) []FileStatus {
	result := make([]FileStatus, 0, len(files))
	for _, file := range files {
		if file.Binary || file.Staged != nil && file.Unstaged == nil && !file.Untracked {
			result = append(result, file)
			continue
		}
		if workingTreeFileLooksBinary(filepath.Join(cwd, file.Path)) {
			file.Binary = true
		}
		result = append(result, file)
	}
	return result
}

func workingTreeFileLooksBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 8000)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return false
	}
	return bytes.IndexByte(buffer[:n], 0) >= 0
}

func scopedGitArgs(base []string, pathspecs []string) []string {
	args := append([]string{}, base...)
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	return args
}

func truncateDiff(diff string, maxBytes int) (string, DiffMetadata) {
	buffer := []byte(diff)
	if len(buffer) <= maxBytes {
		return diff, DiffMetadata{Bytes: len(buffer), OriginalBytes: len(buffer), Truncated: false, OmittedBytes: 0, MaxBytes: maxBytes}
	}
	truncated := string(buffer[:maxBytes])
	suffix := fmt.Sprintf("\n[cnm: diff truncated; omitted %d bytes]\n", len(buffer)-maxBytes)
	result := truncated + suffix
	return result, DiffMetadata{Bytes: len([]byte(result)), OriginalBytes: len(buffer), Truncated: true, OmittedBytes: len(buffer) - maxBytes, MaxBytes: maxBytes}
}

func addBlockingIssue(inspection Inspection, issue Issue) Inspection {
	inspection.Issues = append(inspection.Issues, issue)
	inspection.BlockingIssues = append(inspection.BlockingIssues, issue)
	return inspection
}

func createIssue(code, message string, severity IssueSeverity) Issue {
	return Issue{Code: code, Message: message, Severity: severity}
}

func splitIssues(issues []Issue) ([]Issue, []Issue) {
	warnings := make([]Issue, 0)
	blocking := make([]Issue, 0)
	for _, issue := range issues {
		if issue.Severity == IssueSeverityBlocking {
			blocking = append(blocking, issue)
		} else {
			warnings = append(warnings, issue)
		}
	}
	return warnings, blocking
}

func filterFiles(files []FileStatus, include func(FileStatus) bool) []FileStatus {
	result := make([]FileStatus, 0)
	for _, file := range files {
		if include(file) {
			result = append(result, file)
		}
	}
	return result
}

func mergeFileStatus(files map[string]FileStatus, next FileStatus) {
	existing, ok := files[next.Path]
	if !ok {
		files[next.Path] = next
		return
	}
	if next.OriginalPath != nil {
		existing.OriginalPath = next.OriginalPath
	}
	if next.Staged != nil {
		existing.Staged = next.Staged
	}
	if next.Unstaged != nil {
		existing.Unstaged = next.Unstaged
	}
	existing.Untracked = existing.Untracked || next.Untracked
	existing.Binary = existing.Binary || next.Binary
	files[next.Path] = existing
}

func changeKind(status byte) *ChangeKind {
	var kind ChangeKind
	switch status {
	case 'A':
		kind = ChangeAdded
	case 'C':
		kind = ChangeCopied
	case 'D':
		kind = ChangeDeleted
	case 'M':
		kind = ChangeModified
	case 'R':
		kind = ChangeRenamed
	case 'T':
		kind = ChangeTypechange
	case 'U':
		kind = ChangeUnmerged
	case '?':
		kind = ChangeUnknown
	case ' ':
		return nil
	default:
		if status == 0 {
			return nil
		}
		kind = ChangeUnknown
	}
	return &kind
}

func formatGitFailure(command, stderr, stdout string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(stderr) != "" {
		parts = append(parts, strings.TrimSpace(stderr))
	}
	if strings.TrimSpace(stdout) != "" {
		parts = append(parts, strings.TrimSpace(stdout))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s failed.", command)
	}
	return fmt.Sprintf("%s failed.\n%s", command, strings.Join(parts, "\n"))
}

func normalizeRunnerResult(result CommandResult, err error) CommandResult {
	if err != nil {
		if result.ExitCode == 0 {
			result.ExitCode = 1
		}
		if strings.TrimSpace(result.Stderr) == "" {
			result.Stderr = err.Error()
		}
	}
	return result
}

func mergeEnv(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return os.Environ()
	}
	base := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			base[parts[0]] = parts[1]
		}
	}
	for key, value := range overrides {
		base[key] = value
	}
	result := make([]string, 0, len(base))
	for key, value := range base {
		result = append(result, key+"="+value)
	}
	return result
}

func asExitError(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*target = exitError
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func toAbsolutePath(basePath, candidatePath string) string {
	if filepath.IsAbs(candidatePath) {
		return candidatePath
	}
	return filepath.Clean(filepath.Join(basePath, candidatePath))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitNullSeparated(output string) []string {
	parts := strings.Split(output, "\x00")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func stringPtr(value string) *string {
	v := value
	return &v
}
