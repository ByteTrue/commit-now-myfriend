package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type tempRepo struct {
	path string
}

func createTempRepo(t *testing.T, initialCommit bool, identity bool) *tempRepo {
	t.Helper()
	repoPath := t.TempDir()
	runGit(t, repoPath, nil, "init")
	if identity {
		runGit(t, repoPath, nil, "config", "user.name", "CNM Test")
		runGit(t, repoPath, nil, "config", "user.email", "cnm@example.test")
	}
	repo := &tempRepo{path: repoPath}
	if initialCommit {
		repo.writeFile(t, "README.md", []byte("initial\n"))
		runGit(t, repoPath, nil, "add", "README.md")
		runGit(t, repoPath, nil, "commit", "-m", "chore: initial")
	}
	return repo
}

func (r *tempRepo) writeFile(t *testing.T, relativePath string, content []byte) {
	t.Helper()
	target := filepath.Join(r.path, relativePath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func runGit(t *testing.T, cwd string, env map[string]string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = cwd
	command.Env = mergeEnv(env)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func createFailingGitRunner(shouldFail func(args []string) bool) CommandRunner {
	return func(cwd string, args []string, env map[string]string) (CommandResult, error) {
		if shouldFail(args) {
			return CommandResult{ExitCode: 128, Stderr: "simulated failure for git " + strings.Join(args, " ")}, nil
		}
		return DefaultCommandRunner(cwd, args, env)
	}
}

func filePaths(files []FileStatus) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.Path)
	}
	return result
}

func issueCodes(issues []Issue) []string {
	result := make([]string, 0, len(issues))
	for _, issue := range issues {
		result = append(result, issue.Code)
	}
	return result
}

func TestInspectRepositoryDetectsNonGitDirectory(t *testing.T) {
	outside := t.TempDir()
	inspection, err := InspectRepository(InspectOptions{CWD: outside})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	if inspection.Repository.IsRepository {
		t.Fatalf("expected non-git directory")
	}
	if !contains(issueCodes(inspection.BlockingIssues), "not_git_repository") {
		t.Fatalf("expected not_git_repository issue, got %v", issueCodes(inspection.BlockingIssues))
	}
}

func TestInspectRepositoryBlocksBareRepository(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, nil, "init", "--bare")
	inspection, err := InspectRepository(InspectOptions{CWD: bare})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	if !inspection.Repository.IsRepository || !inspection.Repository.IsBare {
		t.Fatalf("expected bare repository detection")
	}
	if !contains(issueCodes(inspection.BlockingIssues), "bare_repository") {
		t.Fatalf("expected bare_repository issue, got %v", issueCodes(inspection.BlockingIssues))
	}
}

func TestInspectRepositoryScopesStagedDiff(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "staged.txt", []byte("staged secret-free content\n"))
	runGit(t, repo.path, nil, "add", "staged.txt")
	repo.writeFile(t, "README.md", []byte("unstaged content must stay out\n"))

	inspection, err := InspectRepository(InspectOptions{CWD: repo.path})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	if !inspection.HasStagedChanges || !inspection.HasUnstagedChanges {
		t.Fatalf("expected staged and unstaged changes")
	}
	if got := filePaths(inspection.StagedFiles); len(got) != 1 || got[0] != "staged.txt" {
		t.Fatalf("unexpected staged files: %v", got)
	}
	if !strings.Contains(inspection.StagedDiff, "staged.txt") || strings.Contains(inspection.StagedDiff, "README.md") {
		t.Fatalf("staged diff was not scoped correctly: %q", inspection.StagedDiff)
	}
	if !contains(issueCodes(inspection.Warnings), "unstaged_changes_present") {
		t.Fatalf("expected unstaged_changes_present warning, got %v", issueCodes(inspection.Warnings))
	}
	for _, warning := range inspection.Warnings {
		if warning.Code == "unstaged_changes_present" && strings.Contains(warning.Message, "staged diff") {
			t.Fatalf("unstaged warning should not use old staged-first wording: %q", warning.Message)
		}
	}
}

func TestInspectRepositoryDistinguishesFileStates(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "staged.txt", []byte("staged\n"))
	runGit(t, repo.path, nil, "add", "staged.txt")
	repo.writeFile(t, "README.md", []byte("changed\n"))
	repo.writeFile(t, "untracked.txt", []byte("new\n"))

	inspection, err := InspectRepository(InspectOptions{CWD: repo.path})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	if got := filePaths(inspection.StagedFiles); len(got) != 1 || got[0] != "staged.txt" {
		t.Fatalf("unexpected staged files: %v", got)
	}
	if got := filePaths(inspection.UnstagedFiles); len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("unexpected unstaged files: %v", got)
	}
	if got := filePaths(inspection.UntrackedFiles); len(got) != 1 || got[0] != "untracked.txt" {
		t.Fatalf("unexpected untracked files: %v", got)
	}
	codes := issueCodes(inspection.Warnings)
	if !contains(codes, "unstaged_changes_present") || !contains(codes, "untracked_files_present") {
		t.Fatalf("unexpected warning codes: %v", codes)
	}
	for _, warning := range inspection.Warnings {
		if strings.Contains(warning.Message, "not included") || strings.Contains(warning.Message, "staged diff") || strings.Contains(warning.Message, "explicitly staged") {
			t.Fatalf("repository warnings should not imply staged-first product behavior: %q", warning.Message)
		}
	}
}

func TestInspectCommitScopeDefaultsToWorkingTreeWithUntracked(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, ".gitignore", []byte("ignored.txt\n"))
	runGit(t, repo.path, nil, "add", ".gitignore")
	runGit(t, repo.path, nil, "commit", "-m", "test: add ignore rules")
	repo.writeFile(t, "staged.txt", []byte("staged\n"))
	runGit(t, repo.path, nil, "add", "staged.txt")
	repo.writeFile(t, "README.md", []byte("changed\n"))
	repo.writeFile(t, "untracked.txt", []byte("new\n"))
	repo.writeFile(t, "ignored.txt", []byte("ignored\n"))

	scope, err := InspectCommitScope(CommitScopeOptions{CWD: repo.path, IncludeUntracked: true})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	got := filePaths(scope.Files)
	sort.Strings(got)
	if strings.Join(got, ",") != "README.md,staged.txt,untracked.txt" {
		t.Fatalf("unexpected scope files: %v", got)
	}
	if !scope.IncludesUntracked || scope.StagedOnly || !scope.HasSelectedChanges {
		t.Fatalf("unexpected scope metadata: %+v", scope)
	}
}

func TestInspectCommitScopeSupportsStagedOnlyNoUntrackedAndPathspec(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "src/staged.go", []byte("package src\n"))
	runGit(t, repo.path, nil, "add", "src/staged.go")
	repo.writeFile(t, "src/unstaged.go", []byte("package src\n"))
	repo.writeFile(t, "docs/readme.md", []byte("docs\n"))
	repo.writeFile(t, "src/new.go", []byte("package src\n"))

	scope, err := InspectCommitScope(CommitScopeOptions{CWD: repo.path, StagedOnly: true, IncludeUntracked: true, Pathspecs: []string{"src"}})
	if err != nil {
		t.Fatalf("InspectCommitScope staged-only error: %v", err)
	}
	if got := filePaths(scope.Files); len(got) != 1 || got[0] != "src/staged.go" {
		t.Fatalf("unexpected staged-only files: %v", got)
	}
	if !scope.StagedOnly || !scope.IncludesUntracked || strings.Join(scope.Pathspecs, ",") != "src" {
		t.Fatalf("unexpected staged-only metadata: %+v", scope)
	}

	scope, err = InspectCommitScope(CommitScopeOptions{CWD: repo.path, IncludeUntracked: false, Pathspecs: []string{"src"}})
	if err != nil {
		t.Fatalf("InspectCommitScope pathspec error: %v", err)
	}
	got := filePaths(scope.Files)
	sort.Strings(got)
	if strings.Join(got, ",") != "src/staged.go" {
		t.Fatalf("unexpected no-untracked pathspec files: %v", got)
	}
}

func TestInspectCommitScopeCapturesIndexSnapshot(t *testing.T) {
	repo := createTempRepo(t, true, true)
	head := strings.TrimSpace(runGit(t, repo.path, nil, "rev-parse", "HEAD"))
	repo.writeFile(t, "staged.txt", []byte("staged\n"))
	runGit(t, repo.path, nil, "add", "staged.txt")
	repo.writeFile(t, "unstaged.txt", []byte("unstaged\n"))

	scope, err := InspectCommitScope(CommitScopeOptions{CWD: repo.path, IncludeUntracked: true})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if scope.IndexSnapshot == nil || scope.IndexSnapshot.Head == nil || *scope.IndexSnapshot.Head != head {
		t.Fatalf("unexpected index snapshot head: %+v", scope.IndexSnapshot)
	}
	if got := filePaths(scope.IndexSnapshot.StagedFiles); len(got) != 1 || got[0] != "staged.txt" {
		t.Fatalf("unexpected index snapshot staged files: %v", got)
	}
}

func TestInspectCommitScopeReportsSecretBlockers(t *testing.T) {
	repo := createTempRepo(t, true, true)
	secret := "sk_" + strings.Repeat("b", 32)
	repo.writeFile(t, "secret.txt", []byte("api_key = '"+secret+"'\n"))

	scope, err := InspectCommitScope(CommitScopeOptions{CWD: repo.path, IncludeUntracked: true})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if len(scope.SecretBlockers) == 0 {
		t.Fatalf("expected secret blockers, got %+v", scope)
	}
	finding := scope.SecretBlockers[0]
	if finding.Path != "secret.txt" || finding.Code == "" || strings.Contains(finding.Excerpt, secret) {
		t.Fatalf("unexpected secret blocker: %+v", finding)
	}
}

func TestInspectCommitScopeMarksOpaqueBinaryChanges(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "asset.bin", []byte{0, 1, 2, 3, 0, 4})

	scope, err := InspectCommitScope(CommitScopeOptions{CWD: repo.path, IncludeUntracked: true})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if len(scope.Files) != 1 || scope.Files[0].Path != "asset.bin" || !scope.Files[0].Binary {
		t.Fatalf("expected binary opaque change, got %+v", scope.Files)
	}
}

func TestInspectCommitScopeEnforcesDiffAndReadBudgets(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "README.md", []byte("initial\n"+strings.Repeat("expanded content\n", 40)))
	repo.writeFile(t, "small.txt", []byte("small file\n"))
	repo.writeFile(t, "large.txt", []byte(strings.Repeat("large file content\n", 20)))

	scope, err := InspectCommitScope(CommitScopeOptions{
		CWD:              repo.path,
		IncludeUntracked: true,
		MaxDiffBytes:     80,
		MaxReadBytes:     32,
	})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if !scope.AIExposure.DiffBudget.Truncated || scope.AIExposure.DiffBudget.MaxBytes != 80 || scope.AIExposure.DiffBudget.OriginalBytes <= scope.AIExposure.DiffBudget.UsedBytes {
		t.Fatalf("expected truncated diff budget, got %+v", scope.AIExposure.DiffBudget)
	}
	if scope.AIExposure.ReadBudget.MaxBytes != 32 || scope.AIExposure.ReadBudget.UsedBytes > 32 {
		t.Fatalf("expected bounded read budget, got %+v", scope.AIExposure.ReadBudget)
	}
}

func TestInspectCommitScopeDiffOnlyPreventsWorkingTreeReads(t *testing.T) {
	repo := createTempRepo(t, true, true)
	secret := "sk_" + strings.Repeat("d", 32)
	repo.writeFile(t, "secret.txt", []byte("api_key = '"+secret+"'\n"))

	scope, err := InspectCommitScope(CommitScopeOptions{
		CWD:              repo.path,
		IncludeUntracked: true,
		DiffOnly:         true,
	})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if scope.ContextPolicy.Mode != ContextPolicyModeDiffOnly || scope.ContextPolicy.FileReadsAllowed {
		t.Fatalf("unexpected context policy: %+v", scope.ContextPolicy)
	}
	if len(scope.SecretBlockers) != 0 || scope.AIExposure.SecretBlockerCount != 0 || scope.AIExposure.ReadBudget.UsedBytes != 0 {
		t.Fatalf("diff-only should not read working tree files: %+v", scope)
	}
}

func TestInspectCommitScopeReportsProviderVisibleFilesAndPreferenceSources(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "README.md", []byte("changed\n"))
	repo.writeFile(t, "asset.bin", []byte{0, 1, 2, 3})

	scope, err := InspectCommitScope(CommitScopeOptions{
		CWD:              repo.path,
		IncludeUntracked: true,
		PreferenceSources: PreferenceExposure{
			Provider:            "user_config",
			Model:               "default",
			APIKey:              "env",
			PromptStyle:         "project_config",
			MessageLanguage:     "flag",
			StandingInstruction: "project_config,user_config",
		},
	})
	if err != nil {
		t.Fatalf("InspectCommitScope error: %v", err)
	}
	if len(scope.AIExposure.ProviderVisibleFiles) != 2 {
		t.Fatalf("unexpected provider visible files: %+v", scope.AIExposure.ProviderVisibleFiles)
	}
	var opaque *ProviderVisibleFile
	for index := range scope.AIExposure.ProviderVisibleFiles {
		file := &scope.AIExposure.ProviderVisibleFiles[index]
		if file.Path == "asset.bin" {
			opaque = file
		}
	}
	if opaque == nil || !opaque.Opaque || opaque.Source != "metadata" {
		t.Fatalf("expected binary file to be metadata-only exposure, got %+v", opaque)
	}
	if scope.AIExposure.PreferenceSources.Provider != "user_config" || scope.AIExposure.PreferenceSources.APIKey != "env" || scope.AIExposure.PreferenceSources.StandingInstruction != "project_config,user_config" {
		t.Fatalf("unexpected preference sources: %+v", scope.AIExposure.PreferenceSources)
	}
}

func TestRollbackCommitTransactionDetectsStatusChange(t *testing.T) {
	repo := createTempRepo(t, true, true)
	snapshot, err := CaptureCommitTransactionSnapshot(repo.path, nil, nil)
	if err != nil {
		t.Fatalf("CaptureCommitTransactionSnapshot error: %v", err)
	}
	repo.writeFile(t, "docs/guide.md", []byte("docs\n"))
	runGit(t, repo.path, nil, "add", "docs/guide.md")
	runGit(t, repo.path, nil, "commit", "-m", "docs: add guide")
	repo.writeFile(t, "concurrent.txt", []byte("concurrent\n"))

	rollback := RollbackCommitTransaction(repo.path, nil, snapshot, nil)
	if !rollback.RolledBack || rollback.Status != "rolled_back_with_status_change" {
		t.Fatalf("expected rollback with status change, got %+v", rollback)
	}
	if head := strings.TrimSpace(runGit(t, repo.path, nil, "rev-parse", "HEAD")); head != snapshot.Head {
		t.Fatalf("expected head rollback to %s, got %s", snapshot.Head, head)
	}
}

func TestInspectRepositoryDetectsMetadataStates(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "old-name.txt", []byte("rename me\n"))
	repo.writeFile(t, "remove-me.txt", []byte("delete me\n"))
	runGit(t, repo.path, nil, "add", "old-name.txt", "remove-me.txt")
	runGit(t, repo.path, nil, "commit", "-m", "test: add metadata fixtures")
	runGit(t, repo.path, nil, "mv", "old-name.txt", "new-name.txt")
	runGit(t, repo.path, nil, "rm", "remove-me.txt")
	repo.writeFile(t, "image.bin", []byte{0, 159, 146, 150, 0, 1, 2, 3})
	runGit(t, repo.path, nil, "add", "image.bin")

	inspection, err := InspectRepository(InspectOptions{CWD: repo.path})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	var renamed, deleted, binary *FileStatus
	for i := range inspection.StagedFiles {
		file := &inspection.StagedFiles[i]
		switch file.Path {
		case "new-name.txt":
			renamed = file
		case "remove-me.txt":
			deleted = file
		case "image.bin":
			binary = file
		}
	}
	if renamed == nil || renamed.Staged == nil || *renamed.Staged != ChangeRenamed || renamed.OriginalPath == nil || *renamed.OriginalPath != "old-name.txt" {
		t.Fatalf("unexpected rename metadata: %+v", renamed)
	}
	if deleted == nil || deleted.Staged == nil || *deleted.Staged != ChangeDeleted {
		t.Fatalf("unexpected delete metadata: %+v", deleted)
	}
	if binary == nil || !binary.Binary {
		t.Fatalf("expected binary metadata: %+v", binary)
	}
}

func TestInspectRepositoryDetectsMarkerStates(t *testing.T) {
	repo := createTempRepo(t, true, true)
	gitDir := filepath.Join(repo.path, ".git")
	if err := os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte("marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("marker\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inspection, err := InspectRepository(InspectOptions{CWD: repo.path})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	codes := issueCodes(inspection.BlockingIssues)
	for _, expected := range []string{"merge_in_progress", "rebase_in_progress", "cherry_pick_in_progress"} {
		if !contains(codes, expected) {
			t.Fatalf("expected issue %s, got %v", expected, codes)
		}
	}
}

func TestInspectRepositoryWarnsOnDetachedHead(t *testing.T) {
	repo := createTempRepo(t, true, true)
	head := strings.TrimSpace(runGit(t, repo.path, nil, "rev-parse", "HEAD"))
	runGit(t, repo.path, nil, "checkout", "--detach", head)

	inspection, err := InspectRepository(InspectOptions{CWD: repo.path})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	codes := issueCodes(inspection.Warnings)
	if !contains(codes, "detached_head") {
		t.Fatalf("expected detached_head warning, got %v", codes)
	}
}

func TestInspectRepositoryDetectsMissingIdentityAndSecrets(t *testing.T) {
	repo := createTempRepo(t, false, false)
	providerKey := "sk_" + strings.Repeat("a", 32)
	repo.writeFile(t, "secret.txt", []byte("api_key = '"+providerKey+"'\nlarge=content\n"))
	runGit(t, repo.path, nil, "add", "secret.txt")
	home := filepath.Join(repo.path, "isolated-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	inspection, err := InspectRepository(InspectOptions{
		CWD:          repo.path,
		MaxDiffBytes: 40,
		Env: map[string]string{
			"HOME":            home,
			"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		},
	})
	if err != nil {
		t.Fatalf("InspectRepository error: %v", err)
	}
	if inspection.Repository.HasGitIdentity {
		t.Fatalf("expected missing git identity")
	}
	if !contains(issueCodes(inspection.BlockingIssues), "git_identity_missing") {
		t.Fatalf("expected git_identity_missing, got %v", issueCodes(inspection.BlockingIssues))
	}
	if !inspection.Diff.Truncated || inspection.Diff.OmittedBytes <= 0 {
		t.Fatalf("expected diff truncation metadata: %+v", inspection.Diff)
	}
	if len(inspection.SecretScan.Findings) == 0 || inspection.SecretScan.HasBlockingFindings {
		t.Fatalf("unexpected secret scan result: %+v", inspection.SecretScan)
	}
	if !contains(issueCodes(inspection.Warnings), "secret_scan_match") {
		t.Fatalf("expected secret_scan_match warning, got %v", issueCodes(inspection.Warnings))
	}
	for _, warning := range inspection.Warnings {
		if strings.Contains(warning.Message, "staged diff") {
			t.Fatalf("repository warnings should not use old staged-first wording: %q", warning.Message)
		}
	}
}

func TestInspectRepositoryBlocksWhenGitCommandsFail(t *testing.T) {
	tests := []struct {
		code       string
		shouldFail func(args []string) bool
	}{
		{code: "git_status_failed", shouldFail: func(args []string) bool { return len(args) > 0 && args[0] == "status" }},
		{code: "git_diff_numstat_failed", shouldFail: func(args []string) bool { return len(args) > 0 && args[0] == "diff" && contains(args, "--numstat") }},
		{code: "git_diff_patch_failed", shouldFail: func(args []string) bool { return len(args) > 0 && args[0] == "diff" && contains(args, "--patch") }},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			repo := createTempRepo(t, true, true)
			repo.writeFile(t, "staged.txt", []byte("content\n"))
			runGit(t, repo.path, nil, "add", "staged.txt")
			inspection, err := InspectRepository(InspectOptions{CWD: repo.path, GitRunner: createFailingGitRunner(tc.shouldFail)})
			if err != nil {
				t.Fatalf("InspectRepository error: %v", err)
			}
			if !contains(issueCodes(inspection.BlockingIssues), tc.code) {
				t.Fatalf("expected %s issue, got %v", tc.code, issueCodes(inspection.BlockingIssues))
			}
		})
	}
}

func TestStageAllChanges(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "new-file.txt", []byte("new\n"))
	result, err := StageAllChanges(StageAllChangesOptions{CWD: repo.path, Confirmed: false, IsTTY: true})
	if err != nil {
		t.Fatalf("StageAllChanges error: %v", err)
	}
	if result.Staged || result.Reason != StageReasonNotConfirmed {
		t.Fatalf("unexpected result: %+v", result)
	}

	result, err = StageAllChanges(StageAllChangesOptions{CWD: repo.path, Confirmed: true, IsTTY: false})
	if err != nil {
		t.Fatalf("StageAllChanges error: %v", err)
	}
	if result.Staged || result.Reason != StageReasonNonTTY {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := filePaths(result.Inspection.UntrackedFiles); len(got) != 1 || got[0] != "new-file.txt" {
		t.Fatalf("unexpected untracked files: %v", got)
	}

	repo = createTempRepo(t, true, true)
	repo.writeFile(t, "new-file.txt", []byte("new\n"))
	repo.writeFile(t, "README.md", []byte("changed\n"))
	result, err = StageAllChanges(StageAllChangesOptions{CWD: repo.path, Confirmed: true, IsTTY: true})
	if err != nil {
		t.Fatalf("StageAllChanges error: %v", err)
	}
	if !result.Staged || result.Reason != StageReasonConfirmed {
		t.Fatalf("unexpected success result: %+v", result)
	}
	got := filePaths(result.Inspection.StagedFiles)
	sort.Strings(got)
	if strings.Join(got, ",") != "README.md,new-file.txt" {
		t.Fatalf("unexpected staged files: %v", got)
	}

	repo = createTempRepo(t, true, true)
	repo.writeFile(t, "new-file.txt", []byte("new\n"))
	result, err = StageAllChanges(StageAllChangesOptions{CWD: repo.path, Confirmed: true, IsTTY: true, GitRunner: createFailingGitRunner(func(args []string) bool {
		return len(args) >= 2 && args[0] == "add" && args[1] == "-A"
	})})
	if err != nil {
		t.Fatalf("StageAllChanges error: %v", err)
	}
	if result.Staged || result.Reason != StageReasonGitAddFailed {
		t.Fatalf("unexpected failure result: %+v", result)
	}
	if !contains(issueCodes(result.Inspection.BlockingIssues), "git_add_failed") {
		t.Fatalf("expected git_add_failed issue, got %v", issueCodes(result.Inspection.BlockingIssues))
	}
}

func TestGetRecentCommits(t *testing.T) {
	repo := createTempRepo(t, true, true)
	repo.writeFile(t, "a.txt", []byte("a\n"))
	runGit(t, repo.path, nil, "add", "a.txt")
	runGit(t, repo.path, nil, "commit", "-m", "feat: add a")
	repo.writeFile(t, "b.txt", []byte("b\n"))
	runGit(t, repo.path, nil, "add", "b.txt")
	runGit(t, repo.path, nil, "commit", "-m", "fix: add b")

	commits, err := GetRecentCommits(repo.path, nil, 10, nil)
	if err != nil {
		t.Fatalf("GetRecentCommits error: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected recent commits, got %v", commits)
	}
	joined := strings.Join(commits, "\n---\n")
	if !strings.Contains(joined, "feat: add a") || !strings.Contains(joined, "fix: add b") {
		t.Fatalf("unexpected commits: %v", commits)
	}
}

func contains[T comparable](items []T, value T) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
