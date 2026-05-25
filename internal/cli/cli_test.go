package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
	"github.com/ByteTrue/commit-now-myfriend/internal/tui"
)

func TestExecuteWithRuntimePrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"--version"}, Runtime{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d", exitCode)
	}
	if stdout.String() != "dev\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestExecuteWithRuntimeUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"nope"}, Runtime{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected error exit, got %d", exitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: unknown command 'nope'") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestExecuteWithRuntimeHelpShowsRedesignedCommandSurface(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"--help"}, Runtime{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	output := stdout.String()
	for _, expected := range []string{"cnm auto", "cnm init", "cnm config", "cnm doctor"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected help to contain %q, got:\n%s", expected, output)
		}
	}
	for _, expected := range []string{"--json", "--staged", "--no-untracked", "--diff-only", "--no-verify", "--provider", "--model", "--base-url", "--prompt-style", "--message-language", "--standing-instruction", "-- <pathspec...>"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected help to document root flag %q, got:\n%s", expected, output)
		}
	}
	for _, removed := range []string{"cnm split", "cnm repair", "cnm check", "cnm onboard"} {
		if strings.Contains(output, removed) {
			t.Fatalf("help should not document removed command %q:\n%s", removed, output)
		}
	}
}

func TestExecuteWithRuntimeRemovedCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"split", "repair", "check", "onboard"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := ExecuteWithRuntime([]string{command}, Runtime{
				Stdout: &stdout,
				Stderr: &stderr,
				Stdin:  strings.NewReader(""),
				IsTTY:  false,
			})

			if exitCode != 1 {
				t.Fatalf("expected error exit, got %d", exitCode)
			}
			if stdout.String() != "" {
				t.Fatalf("unexpected stdout: %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), "removed command '"+command+"'") {
				t.Fatalf("unexpected stderr: %q", stderr.String())
			}
		})
	}
}

func TestExecuteWithRuntimeAutoHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"auto", "--help"}, Runtime{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	output := stdout.String()
	for _, expected := range []string{"Usage: cnm auto", "Autonomous Commit", "--dry-run", "--json", "--tui", "--staged", "--no-untracked", "--diff-only", "--no-verify", "--verbose"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected auto help to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestExecuteWithRuntimeRootNonTTYRendersInteractiveCommitPreview(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected interactive preview success exit, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	view := stdout.String()
	for _, expected := range []string{"Interactive Commit", "Commit Scope", "README.md", "new.txt", "AI Exposure", "Commit Plan"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected preview to contain %q:\n%s", expected, view)
		}
	}
	headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if headAfter != headBefore || statusAfter != statusBefore {
		t.Fatalf("interactive preview mutated repo: head %q -> %q status %q -> %q", headBefore, headAfter, statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeRootJSONReportsTUIPreview(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected tui preview success exit, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		SchemaVersion int    `json:"schemaVersion"`
		Command       string `json:"command"`
		OK            bool   `json:"ok"`
		Status        string `json:"status"`
		Scope         struct {
			HasSelectedChanges bool `json:"hasSelectedChanges"`
			IncludesUntracked  bool `json:"includesUntracked"`
			Files              []struct {
				Path string `json:"path"`
			} `json:"files"`
		} `json:"scope"`
		AIExposure struct {
			SelectedFileCount int `json:"selectedFileCount"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != 1 || payload.Command != "cnm" || payload.Status != "tui_preview" || !payload.OK {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if !payload.Scope.HasSelectedChanges || !payload.Scope.IncludesUntracked || payload.AIExposure.SelectedFileCount != 2 {
		t.Fatalf("unexpected preview scope: %+v exposure=%+v", payload.Scope, payload.AIExposure)
	}
	paths := make([]string, 0, len(payload.Scope.Files))
	for _, file := range payload.Scope.Files {
		paths = append(paths, file.Path)
	}
	if !containsString(paths, "README.md") || !containsString(paths, "new.txt") {
		t.Fatalf("unexpected scope files: %v", paths)
	}
}

func TestExecuteWithRuntimeRootJSONReportsScopeError(t *testing.T) {
	temp := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected scope error exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["schemaVersion"] != float64(1) || payload["command"] != "cnm" || payload["status"] != "scope_error" || payload["ok"] != false {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeRootJSONRespectsCommitScopeFlagsAndPathspecs(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"--json", "--staged", "--no-untracked", "--", "README.md", "docs/guide.md"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected tui preview success exit, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		Scope  struct {
			StagedOnly        bool     `json:"stagedOnly"`
			IncludesUntracked bool     `json:"includesUntracked"`
			Pathspecs         []string `json:"pathspecs"`
			Files             []struct {
				Path string `json:"path"`
			} `json:"files"`
		} `json:"scope"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "tui_preview" || !payload.Scope.StagedOnly || payload.Scope.IncludesUntracked {
		t.Fatalf("unexpected scoped preview metadata: %+v", payload)
	}
	if len(payload.Scope.Pathspecs) != 2 || payload.Scope.Pathspecs[0] != "README.md" || payload.Scope.Pathspecs[1] != "docs/guide.md" {
		t.Fatalf("unexpected pathspecs: %v", payload.Scope.Pathspecs)
	}
	if len(payload.Scope.Files) != 1 || payload.Scope.Files[0].Path != "README.md" {
		t.Fatalf("unexpected scoped files: %+v", payload.Scope.Files)
	}
}

func TestExecuteWithRuntimeRootTUIRequiresConfirmationBeforeCommit(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tuiCalled := false
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			tuiCalled = true
			return tui.Result{Cancelled: true, CommitPlan: input.CommitPlan}, nil
		},
	})

	if exitCode != 130 {
		t.Fatalf("expected user cancel exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !tuiCalled {
		t.Fatalf("expected TUI runner to be called")
	}
	if !strings.Contains(stderr.String(), "Cancelled") {
		t.Fatalf("expected cancellation output, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("cancelled TUI should not create commit: %s -> %s", headBefore, headAfter)
	}
	if statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); statusAfter != statusBefore {
		t.Fatalf("cancelled TUI should not mutate status: before %q after %q", statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeRootTTYRunsFirstRunOnboardingBeforeTUI(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{}
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tuiCalled := false
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader("openai-responses\ngpt-5-mini\nauto\nzh-CN\nPrefer concise subjects.\nsk_test_first_run_secret_1234567890\n"),
		IsTTY:       true,
		SecretStore: secretStore,
		OnboardingRunner: func(input tui.OnboardingInput, runtime tui.Runtime) (tui.OnboardingResult, error) {
			fmt.Fprintln(runtime.Output, "Onboarding")
			return tui.OnboardingResult{
				Provider:            config.ProviderOpenAIResponses,
				Model:               "gpt-5-mini",
				PromptStyle:         config.PromptStyleAuto,
				MessageLanguage:     config.MessageLanguage("zh-CN"),
				StandingInstruction: "Prefer concise subjects.",
				APIKey:              "sk_test_first_run_secret_1234567890",
			}, nil
		},
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			tuiCalled = true
			if input.Scope.AIExposure.PreferenceSources.APIKey != "secret_store" {
				t.Fatalf("expected TUI scope to see secret store API key source, got %+v", input.Scope.AIExposure.PreferenceSources)
			}
			return tui.Result{Cancelled: true, CommitPlan: input.CommitPlan}, nil
		},
	})

	if exitCode != 130 {
		t.Fatalf("expected cancel after onboarding and TUI, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !tuiCalled {
		t.Fatalf("expected TUI runner after onboarding")
	}
	if !strings.Contains(stdout.String(), "Onboarding") {
		t.Fatalf("expected onboarding output, got %q", stdout.String())
	}
	if secretStore.keys["openai-responses"] != "sk_test_first_run_secret_1234567890" {
		t.Fatalf("expected API key in secret store, got %+v", secretStore.keys)
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(loaded)
	for _, expected := range []string{"openai-responses", "gpt-5-mini", "zh-CN", "Prefer concise subjects."} {
		if !strings.Contains(configText, expected) {
			t.Fatalf("expected config to contain %q: %s", expected, configText)
		}
	}
	if strings.Contains(configText, "apiKey") || strings.Contains(configText, "sk_test") {
		t.Fatalf("onboarding should not write plaintext API key: %s", configText)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("cancelled TUI after onboarding should not commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeRootJSONDoesNotRunFirstRunOnboarding(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{}
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"--json"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader("openai-responses\ngpt-5-mini\nauto\nzh-CN\nsk_test_should_not_be_read\n"),
		IsTTY:       true,
		SecretStore: secretStore,
	})

	if exitCode != 0 {
		t.Fatalf("expected JSON preview success, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Onboarding") {
		t.Fatalf("JSON mode must not include onboarding prompts: %s", stdout.String())
	}
	if len(secretStore.keys) != 0 {
		t.Fatalf("JSON mode should not save credentials: %+v", secretStore.keys)
	}
	var payload struct {
		Status     string `json:"status"`
		AIExposure struct {
			PreferenceSources struct {
				APIKey string `json:"apiKey"`
			} `json:"preferenceSources"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "tui_preview" || payload.AIExposure.PreferenceSources.APIKey != "missing" {
		t.Fatalf("unexpected JSON preview payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeRootTUIAcceptedPlanCreatesCommit(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "docs: refresh readme", Files: []string{"README.md"}}}}}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected accepted TUI commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Committed ") || !strings.Contains(stdout.String(), "docs: refresh readme") {
		t.Fatalf("unexpected commit output: %q", stdout.String())
	}
	headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	if headAfter == headBefore {
		t.Fatalf("expected a new commit")
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != "docs: refresh readme" {
		t.Fatalf("unexpected commit message: %q", message)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); status != "" {
		t.Fatalf("expected clean selected changes after TUI commit, got %q", status)
	}
}

func TestExecuteWithRuntimeRootTUIAIActivityPlansThroughToolCallLoop(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed through tui planner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	providerMessage := "docs: plan readme through tools"
	provider := autoCommitProviderForCLITest(providerMessage, []string{"README.md"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          true,
		CommitProvider: provider,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			if input.PlanCommits == nil {
				t.Fatalf("expected TUI model input to expose Tool Call Loop planner")
			}
			plan, err := input.PlanCommits(tui.PlanCommitsInput{Scope: input.Scope, ScopeFiles: []string{"README.md"}, AgentInstruction: "prefer docs wording"})
			if err != nil {
				t.Fatalf("PlanCommits error: %v", err)
			}
			return tui.Result{Accepted: true, ScopeFiles: []string{"README.md"}, CommitPlan: plan}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected TUI tool-loop planned commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if len(provider.ReceivedResults) != 3 || provider.ReceivedResults[2].Name != runtimex.ToolCreateCommits || !provider.ReceivedResults[2].OK {
		t.Fatalf("expected TUI planning to use create_commits tool call, got %+v", provider.ReceivedResults)
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != providerMessage {
		t.Fatalf("expected provider-planned commit message, got %q", message)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter == headBefore {
		t.Fatal("expected TUI commit to create a new commit")
	}
}

func TestExecuteWithRuntimeRootTUIRequiresCommitPlanBeforeCreatingCommit(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed without plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true}, nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected missing-plan failure exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "commit plan is required") {
		t.Fatalf("expected missing commit plan error, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("missing commit plan should not create commit: %s -> %s", headBefore, headAfter)
	}
	if statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); statusAfter != statusBefore {
		t.Fatalf("missing commit plan should not mutate repo: before %q after %q", statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeRootTUIAcceptedPlanCreatesMultipleCommits(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "file_split", Commits: []tui.CommitView{
				{Message: "docs: add guide", Files: []string{"docs/guide.md"}},
				{Message: "feat: add app", Files: []string{"src/app.go"}},
			}}}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected accepted multi-commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if count := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-list", "--count", headBefore+"..HEAD")); count != "2" {
		t.Fatalf("expected two TUI commits, got %s", count)
	}
	messages := runGitOutputForCLITest(t, temp, "log", "--max-count=2", "--pretty=%s")
	if !strings.Contains(messages, "feat: add app") || !strings.Contains(messages, "docs: add guide") {
		t.Fatalf("unexpected TUI commit messages: %q", messages)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); status != "" {
		t.Fatalf("expected clean working tree after TUI multi-commit, got %q", status)
	}
}

func TestExecuteWithRuntimeRootTUICommitsAdjustedScopeOnly(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "scratch.txt"), []byte("scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{
				Accepted:   true,
				ScopeFiles: []string{"README.md"},
				CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "docs: update readme", Files: []string{"README.md"}}}},
			}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected adjusted-scope TUI commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != "docs: update readme" {
		t.Fatalf("unexpected commit message: %q", message)
	}
	show := runGitOutputForCLITest(t, temp, "show", "--name-only", "--pretty=", "HEAD")
	if !strings.Contains(show, "README.md") || strings.Contains(show, "scratch.txt") {
		t.Fatalf("adjusted scope commit included wrong files: %q", show)
	}
	status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "--untracked-files=all")
	if !strings.Contains(status, "?? scratch.txt") {
		t.Fatalf("excluded file should remain untracked, got status %q", status)
	}
}

func TestExecuteWithRuntimeRootTUISurfacesHookFailureWithoutRepair(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho hook failed >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "docs: refresh readme", Files: []string{"README.md"}}}}}, nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected hook failure exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "hook failed") {
		t.Fatalf("expected hook output, got %q", stderr.String())
	}
	if strings.Contains(strings.ToLower(stderr.String()), "repair") || strings.Contains(strings.ToLower(stdout.String()), "repair") {
		t.Fatalf("hook failure should not enter repair path: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("hook failure should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeRootTUINoVerifyBypassesGitHooks(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho hook failed >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"--no-verify"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "docs: refresh readme", Files: []string{"README.md"}}}}}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected no-verify TUI commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != "docs: refresh readme" {
		t.Fatalf("unexpected no-verify commit message: %q", message)
	}
}

func TestExecuteWithRuntimeRootTUIRollsBackPartialMultiCommitFailure(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\nif grep -q 'feat: add app' \"$1\"; then echo second TUI commit rejected >&2; exit 1; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "commit-msg"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime(nil, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "file_split", Commits: []tui.CommitView{
				{Message: "docs: add guide", Files: []string{"docs/guide.md"}},
				{Message: "feat: add app", Files: []string{"src/app.go"}},
			}}}, nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected TUI split failure exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "second TUI commit rejected") {
		t.Fatalf("expected second commit failure output, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("expected rollback to original head: %s -> %s", headBefore, headAfter)
	}
	if statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); statusAfter != statusBefore {
		t.Fatalf("expected rollback to restore status: before %q after %q", statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONReportsScopeError(t *testing.T) {
	temp := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected scope error exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["schemaVersion"] != float64(1) || payload["command"] != "cnm auto" || payload["status"] != "scope_error" || payload["ok"] != false {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeAutoJSONFailsPredictablyWhenAPIKeyMissing(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("openai-responses\ngpt-5-mini\nauto\nzh-CN\nsk_test_should_not_be_read\n"),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected config missing failure, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr in json mode: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Onboarding") {
		t.Fatalf("cnm auto must not run onboarding: %s", stdout.String())
	}
	var payload struct {
		Status      string `json:"status"`
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		ConfigIssue struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"configIssue"`
		AIExposure struct {
			PreferenceSources struct {
				APIKey string `json:"apiKey"`
			} `json:"preferenceSources"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "config_missing" || payload.OK || payload.ConfigIssue.Code != "api_key_missing" || payload.AIExposure.PreferenceSources.APIKey != "missing" {
		t.Fatalf("unexpected config missing payload: %+v", payload)
	}
	if !strings.Contains(payload.Error, "API key") || !strings.Contains(payload.ConfigIssue.Message, "cnm init") {
		t.Fatalf("expected actionable config message, got %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("config missing path should not commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONFailsPredictablyWhenAPIKeyMissing(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected dry-run config missing failure, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status      string `json:"status"`
		OK          bool   `json:"ok"`
		DryRun      bool   `json:"dryRun"`
		ConfigIssue struct {
			Code string `json:"code"`
		} `json:"configIssue"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "config_missing" || payload.OK || !payload.DryRun || payload.ConfigIssue.Code != "api_key_missing" {
		t.Fatalf("unexpected dry-run config missing payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("dry-run config missing path should not commit: %s -> %s", headBefore, headAfter)
	}
	if statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); statusAfter != statusBefore {
		t.Fatalf("dry-run config missing path mutated repo: before %q after %q", statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONReportsCommitPlanPreviewWithoutMutatingRepo(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("chore: update selected files", []string{"README.md", "new.txt"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected plan preview success exit, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
		DryRun bool   `json:"dryRun"`
		Scope  struct {
			HasSelectedChanges bool `json:"hasSelectedChanges"`
			IncludesUntracked  bool `json:"includesUntracked"`
			Files              []struct {
				Path      string `json:"path"`
				Untracked bool   `json:"untracked"`
			} `json:"files"`
		} `json:"scope"`
		CommitPlan struct {
			Kind    string `json:"kind"`
			Commits []struct {
				Message string `json:"message"`
				Files   []struct {
					Path string `json:"path"`
				} `json:"files"`
			} `json:"commits"`
		} `json:"commitPlan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "plan_preview" || !payload.OK || !payload.DryRun {
		t.Fatalf("unexpected preview metadata: %+v", payload)
	}
	if !payload.Scope.HasSelectedChanges || !payload.Scope.IncludesUntracked {
		t.Fatalf("unexpected scope metadata: %+v", payload.Scope)
	}
	paths := make([]string, 0, len(payload.Scope.Files))
	for _, file := range payload.Scope.Files {
		paths = append(paths, file.Path)
	}
	if !containsString(paths, "README.md") || !containsString(paths, "new.txt") {
		t.Fatalf("unexpected scope files: %v", paths)
	}
	if payload.CommitPlan.Kind != "single" || len(payload.CommitPlan.Commits) != 1 || strings.TrimSpace(payload.CommitPlan.Commits[0].Message) == "" {
		t.Fatalf("unexpected commit plan: %+v", payload.CommitPlan)
	}
	headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if headAfter != headBefore || statusAfter != statusBefore {
		t.Fatalf("dry-run mutated repo: head %q -> %q status %q -> %q", headBefore, headAfter, statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONSplitsClearlyIndependentTopLevelFiles(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
		CommitProvider: autoCommitProviderWithPlanForCLITest("file_split", []runtimex.CreateCommitInput{
			{Message: "docs: add guide", Files: []string{"docs/guide.md"}},
			{Message: "feat: add app", Files: []string{"src/app.go"}},
		}, nil),
	})

	if exitCode != 0 {
		t.Fatalf("expected split preview success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status     string `json:"status"`
		CommitPlan struct {
			Kind    string `json:"kind"`
			Commits []struct {
				Message string `json:"message"`
				Files   []struct {
					Path string `json:"path"`
				} `json:"files"`
			} `json:"commits"`
		} `json:"commitPlan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "plan_preview" || payload.CommitPlan.Kind != "file_split" || len(payload.CommitPlan.Commits) != 2 {
		t.Fatalf("unexpected split plan: %+v", payload)
	}
	seen := map[string]bool{}
	for _, commit := range payload.CommitPlan.Commits {
		if len(commit.Files) != 1 {
			t.Fatalf("expected one file per independent top-level group: %+v", commit)
		}
		seen[commit.Files[0].Path] = true
	}
	if !seen["docs/guide.md"] || !seen["src/app.go"] {
		t.Fatalf("unexpected split files: %+v", seen)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONKeepsSameTopLevelFilesTogether(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app_test.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
		CommitProvider: autoCommitProviderWithPlanForCLITest("single", []runtimex.CreateCommitInput{
			{Message: "feat: add src package", Files: []string{"src/app.go", "src/app_test.go"}},
		}, []runtimex.CreateSplitLimitation{{Code: "same_top_level", Message: "AI kept related src files together because they support one change.", Fallback: "single_commit"}}),
	})

	if exitCode != 0 {
		t.Fatalf("expected conservative preview success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		CommitPlan struct {
			Kind             string `json:"kind"`
			SplitLimitations []struct {
				Code     string `json:"code"`
				Fallback string `json:"fallback"`
			} `json:"splitLimitations"`
			Commits []struct {
				Files []struct {
					Path string `json:"path"`
				} `json:"files"`
			} `json:"commits"`
		} `json:"commitPlan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.CommitPlan.Kind != "single" || len(payload.CommitPlan.Commits) != 1 || len(payload.CommitPlan.Commits[0].Files) != 2 {
		t.Fatalf("expected same top-level files to stay together: %+v", payload.CommitPlan)
	}
	if len(payload.CommitPlan.SplitLimitations) != 1 || payload.CommitPlan.SplitLimitations[0].Code != "same_top_level" || payload.CommitPlan.SplitLimitations[0].Fallback != "single_commit" {
		t.Fatalf("expected conservative split limitation: %+v", payload.CommitPlan.SplitLimitations)
	}
}

func TestExecuteWithRuntimeAutoJSONCreatesSingleLocalCommit(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("chore: update selected files", []string{"README.md", "new.txt"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected commit success exit, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
		DryRun bool   `json:"dryRun"`
		Commit struct {
			Hash    string `json:"hash"`
			Message string `json:"message"`
		} `json:"commit"`
		CommitPlan struct {
			Kind    string `json:"kind"`
			Commits []struct {
				Message string `json:"message"`
			} `json:"commits"`
		} `json:"commitPlan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "committed" || !payload.OK || payload.DryRun || payload.Commit.Hash == "" {
		t.Fatalf("unexpected commit payload: %+v", payload)
	}
	if payload.CommitPlan.Kind != "single" || len(payload.CommitPlan.Commits) != 1 || payload.Commit.Message != payload.CommitPlan.Commits[0].Message {
		t.Fatalf("unexpected commit plan payload: %+v", payload)
	}
	headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	if headAfter == headBefore || headAfter != payload.Commit.Hash {
		t.Fatalf("expected new commit hash %q after %q payload %+v", headBefore, headAfter, payload.Commit)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); status != "" {
		t.Fatalf("expected clean working tree after auto commit, got %q", status)
	}
}

func TestExecuteWithRuntimeAutoJSONCreatesCommitThroughToolCallLoop(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed by provider loop\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	providerMessage := "docs: refresh readme through tools"
	provider := &runtimex.FakeProviderTracer{Steps: []runtimex.FakeProviderStep{
		{ToolCalls: []runtimex.ToolCallRequest{
			{ID: "inspect", Name: runtimex.ToolInspectCommitScope},
			{ID: "diff", Name: runtimex.ToolGetDiff},
		}},
		{ToolCalls: []runtimex.ToolCallRequest{{
			ID:   "create",
			Name: runtimex.ToolCreateCommits,
			Arguments: map[string]any{
				"commits": []any{map[string]any{
					"message": providerMessage,
					"files":   []any{"README.md"},
				}},
			},
		}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "finish", Name: runtimex.ToolFinish, Arguments: map[string]any{"message": "done"}}}},
	}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: provider,
	})

	if exitCode != 0 {
		t.Fatalf("expected provider tool-loop commit success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
		Commit struct {
			Hash    string `json:"hash"`
			Message string `json:"message"`
		} `json:"commit"`
		CommitPlan struct {
			Kind    string `json:"kind"`
			Commits []struct {
				Message string `json:"message"`
			} `json:"commits"`
		} `json:"commitPlan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "committed" || !payload.OK || payload.Commit.Message != providerMessage || payload.CommitPlan.Commits[0].Message != providerMessage {
		t.Fatalf("expected commit message to come from create_commits tool call, got %+v", payload)
	}
	if strings.Contains(stdout.String(), "chore: update README.md") {
		t.Fatalf("auto commit should not use local heuristic message: %s", stdout.String())
	}
	if len(provider.ReceivedResults) != 3 || provider.ReceivedResults[0].Name != runtimex.ToolInspectCommitScope || provider.ReceivedResults[1].Name != runtimex.ToolGetDiff || provider.ReceivedResults[2].Name != runtimex.ToolCreateCommits || !provider.ReceivedResults[2].OK {
		t.Fatalf("expected inspect/get_diff/create_commits tool results, got %+v", provider.ReceivedResults)
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != providerMessage {
		t.Fatalf("unexpected git commit message: %q", message)
	}
}

func TestExecuteWithRuntimeAutoJSONCreatesMultipleFileLevelCommits(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
		CommitProvider: autoCommitProviderWithPlanForCLITest("file_split", []runtimex.CreateCommitInput{
			{Message: "docs: add guide", Files: []string{"docs/guide.md"}},
			{Message: "feat: add app", Files: []string{"src/app.go"}},
		}, nil),
	})

	if exitCode != 0 {
		t.Fatalf("expected multi-commit success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status     string `json:"status"`
		OK         bool   `json:"ok"`
		CommitPlan struct {
			Kind    string `json:"kind"`
			Commits []struct {
				Message string `json:"message"`
			} `json:"commits"`
		} `json:"commitPlan"`
		Commits []struct {
			Hash    string `json:"hash"`
			Message string `json:"message"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "committed" || !payload.OK || payload.CommitPlan.Kind != "file_split" || len(payload.CommitPlan.Commits) != 2 || len(payload.Commits) != 2 {
		t.Fatalf("unexpected multi-commit payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter == headBefore || headAfter != payload.Commits[1].Hash {
		t.Fatalf("unexpected final head: before=%s after=%s payload=%+v", headBefore, headAfter, payload.Commits)
	}
	if count := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-list", "--count", headBefore+"..HEAD")); count != "2" {
		t.Fatalf("expected two new commits, got %s", count)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); status != "" {
		t.Fatalf("expected clean working tree after split commit, got %q", status)
	}
}

func TestExecuteWithRuntimeAutoJSONRollsBackPartialSplitCommitsOnFailure(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.MkdirAll(filepath.Join(temp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(temp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "docs", "guide.md"), []byte("docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "src", "app.go"), []byte("package src\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\nif grep -q 'feat: add app' \"$1\"; then echo second split rejected >&2; exit 1; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "commit-msg"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	statusBefore := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
		CommitProvider: autoCommitProviderWithPlanForCLITest("file_split", []runtimex.CreateCommitInput{
			{Message: "docs: add guide", Files: []string{"docs/guide.md"}},
			{Message: "feat: add app", Files: []string{"src/app.go"}},
		}, nil),
	})

	if exitCode != 1 {
		t.Fatalf("expected split failure exit, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status      string `json:"status"`
		OK          bool   `json:"ok"`
		Transaction struct {
			RolledBack bool   `json:"rolledBack"`
			Status     string `json:"status"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "commit_failed" || payload.OK || !payload.Transaction.RolledBack || payload.Transaction.Status != "rolled_back" {
		t.Fatalf("unexpected rollback payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("expected rollback to original head: %s -> %s", headBefore, headAfter)
	}
	if statusAfter := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); statusAfter != statusBefore {
		t.Fatalf("expected index/worktree restoration: before %q after %q", statusBefore, statusAfter)
	}
}

func TestExecuteWithRuntimeAutoShowsCompactRunOutput(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("docs: refresh readme", []string{"README.md"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected commit success exit, got %d stderr=%q stdout=%q", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Committed ") || strings.Contains(stdout.String(), "schemaVersion") {
		t.Fatalf("unexpected compact output: %q", stdout.String())
	}
}

func TestExecuteWithRuntimeAutoJSONRespectsGitHooksByDefault(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho hook failed >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("docs: refresh readme", []string{"README.md"}),
	})

	if exitCode != 1 {
		t.Fatalf("expected hook failure exit, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr in json mode: %q", stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "commit_failed" || payload.OK || !strings.Contains(payload.Error, "hook failed") {
		t.Fatalf("unexpected hook failure payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("hook failure should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoJSONNoVerifyBypassesGitHooks(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "pre-commit"), []byte("#!/bin/sh\necho hook failed >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json", "--no-verify"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("docs: refresh readme", []string{"README.md"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected no-verify commit success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "committed" || !payload.OK {
		t.Fatalf("unexpected no-verify payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeAutoJSONRetriesOnceWhenCommitMessageHookRejects(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	hook := "#!/bin/sh\ncount_file=.git/hooks/commit-msg-count\ncount=0\nif [ -f $count_file ]; then count=$(cat $count_file); fi\ncount=$((count + 1))\necho $count > $count_file\nif [ $count -eq 1 ]; then echo message rejected >&2; exit 1; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(temp, ".git", "hooks", "commit-msg"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	firstMessage := "docs: refresh readme"
	retryMessage := "docs: refresh readme retry"
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
		CommitProvider: autoCommitProviderWithRetryForCLITest(
			runtimex.CreateCommitInput{Message: firstMessage, Files: []string{"README.md"}},
			runtimex.CreateCommitInput{Message: retryMessage, Files: []string{"README.md"}},
		),
	})

	if exitCode != 0 {
		t.Fatalf("expected message retry success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status       string `json:"status"`
		OK           bool   `json:"ok"`
		MessageRetry struct {
			Attempted bool `json:"attempted"`
			Count     int  `json:"count"`
		} `json:"messageRetry"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "committed" || !payload.OK || !payload.MessageRetry.Attempted || payload.MessageRetry.Count != 1 {
		t.Fatalf("unexpected retry payload: %+v", payload)
	}
	if payload.Commit.Message != retryMessage {
		t.Fatalf("expected regenerated retry message, got %+v", payload.Commit)
	}
	countBytes, err := os.ReadFile(filepath.Join(temp, ".git", "hooks", "commit-msg-count"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(countBytes)) != "2" {
		t.Fatalf("expected exactly two commit-msg attempts, got %q", string(countBytes))
	}
}

func TestExecuteWithRuntimeAutoJSONRejectsEmptyCommit(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("no changes should be a clean no-op, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "no_changes" || !payload.OK {
		t.Fatalf("unexpected empty commit payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("empty commit path should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoJSONFailsNonInteractivelyOnConflicts(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected conflict failure exit, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr in json mode: %q", stderr.String())
	}
	var payload struct {
		Status string `json:"status"`
		OK     bool   `json:"ok"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "conflict" || payload.OK || !strings.Contains(strings.ToLower(payload.Error), "conflict") {
		t.Fatalf("unexpected conflict payload: %+v", payload)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("conflict path should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoTUIHandsOffConflictsToInteractiveRepairContext(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tuiCalled := false
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:    temp,
		Env:    configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			tuiCalled = true
			if input.RepairContext == nil {
				t.Fatalf("expected repair context")
			}
			if input.RepairContext.Reason == "" || len(input.RepairContext.EligibleFiles) != 1 || input.RepairContext.EligibleFiles[0] != "conflict.txt" {
				t.Fatalf("unexpected repair context: %+v", input.RepairContext)
			}
			return tui.Result{Cancelled: true}, nil
		},
	})

	if exitCode != 130 {
		t.Fatalf("expected TUI conflict handoff cancellation exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !tuiCalled {
		t.Fatalf("expected TUI handoff")
	}
	if !strings.Contains(stderr.String(), "Cancelled") {
		t.Fatalf("expected cancellation output, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("conflict handoff should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoTUICancelledRepairDoesNotRunToolCallRuntime(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	provider := &runtimex.FakeProviderTracer{Steps: []runtimex.FakeProviderStep{
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "read-conflict", Name: runtimex.ToolReadFile, Arguments: map[string]any{"path": "conflict.txt"}}}},
	}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          true,
		RepairProvider: provider,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Cancelled: true}, nil
		},
	})

	if exitCode != 130 {
		t.Fatalf("expected cancelled TUI repair exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if len(provider.ReceivedResults) != 0 {
		t.Fatalf("cancelled TUI repair should not run tool calls, got %+v", provider.ReceivedResults)
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("cancelled repair should not create commit: %s -> %s", headBefore, headAfter)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1"); !strings.Contains(status, "UU conflict.txt") {
		t.Fatalf("cancelled repair should leave conflict unresolved, got %q", status)
	}
}

func TestExecuteWithRuntimeAutoTUIAcceptedRepairRunsToolCallRuntimeAndCommits(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("safe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1"); !strings.Contains(status, "UU conflict.txt") {
		t.Fatalf("expected an unmerged conflict before repair, got %q", status)
	}
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))
	provider := &runtimex.FakeProviderTracer{Steps: []runtimex.FakeProviderStep{
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "read-conflict", Name: runtimex.ToolReadFile, Arguments: map[string]any{"path": "conflict.txt"}}}},
		{ToolCalls: []runtimex.ToolCallRequest{
			{ID: "repair-readme", Name: runtimex.ToolRepairFile, Arguments: map[string]any{"path": "README.md", "content": "should not write\n"}},
			{ID: "repair-conflict", Name: runtimex.ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "resolved\n"}},
		}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "finish", Name: runtimex.ToolFinish, Arguments: map[string]any{"message": "repaired"}}}},
	}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          true,
		RepairProvider: provider,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "merge: resolve conflict", Files: []string{"conflict.txt"}}}}}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected accepted TUI repair commit success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Committed ") || !strings.Contains(stdout.String(), "merge: resolve conflict") {
		t.Fatalf("unexpected repair commit output: %q", stdout.String())
	}
	if content := string(mustReadFileForCLITest(t, filepath.Join(temp, "conflict.txt"))); content != "resolved\n" {
		t.Fatalf("expected conflict file to be repaired, got %q", content)
	}
	if content := string(mustReadFileForCLITest(t, filepath.Join(temp, "README.md"))); content != "safe\n" {
		t.Fatalf("non-eligible file should not be repaired, got %q", content)
	}
	if len(provider.ReceivedResults) < 3 || provider.ReceivedResults[1].OK || provider.ReceivedResults[1].Error == nil {
		t.Fatalf("expected non-eligible repair tool call to be rejected, got %+v", provider.ReceivedResults)
	}
	if !provider.ReceivedResults[2].OK {
		t.Fatalf("expected eligible conflict repair to succeed, got %+v", provider.ReceivedResults[2])
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter == headBefore {
		t.Fatalf("expected repair flow to create a commit")
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != "merge: resolve conflict" {
		t.Fatalf("unexpected repair commit message: %q", message)
	}
	if status := runGitOutputForCLITest(t, temp, "status", "--porcelain=v1", "-z", "--untracked-files=all"); status != "" {
		t.Fatalf("expected clean working tree after repair commit, got %q", status)
	}
}

func TestExecuteWithRuntimeAutoTUIAcceptedRepairCreatesProviderFromConfig(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	requests := []string{}
	httpClient := roundTripFuncForCLITest(func(request *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(request.Body)
		requests = append(requests, string(body))
		switch len(requests) {
		case 1:
			if !strings.Contains(string(body), "repair_file") || !strings.Contains(string(body), "read_file") {
				t.Fatalf("initial provider request should expose native repair tools: %s", string(body))
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[{"type":"function_call","call_id":"read","name":"read_file","arguments":"{\"path\":\"conflict.txt\"}"}]}`)), Header: make(http.Header)}, nil
		case 2:
			if !strings.Contains(string(body), "function_call_output") || !strings.Contains(string(body), `"call_id":"read"`) {
				t.Fatalf("second provider request should send native tool result context: %s", string(body))
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_2","output":[{"type":"function_call","call_id":"repair","name":"repair_file","arguments":"{\"path\":\"conflict.txt\",\"content\":\"resolved\\n\"}"}]}`)), Header: make(http.Header)}, nil
		case 3:
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_3","output":[{"type":"function_call","call_id":"finish","name":"finish","arguments":"{\"message\":\"repaired\"}"}]}`)), Header: make(http.Header)}, nil
		default:
			t.Fatalf("unexpected provider request: %s", string(body))
			return nil, nil
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:                temp,
		Env:                configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:             &stdout,
		Stderr:             &stderr,
		Stdin:              strings.NewReader(""),
		IsTTY:              true,
		ProviderHTTPClient: httpClient,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			return tui.Result{Accepted: true, CommitPlan: tui.CommitPlanView{Kind: "single", Commits: []tui.CommitView{{Message: "merge: resolve conflict", Files: []string{"conflict.txt"}}}}}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected configured provider repair success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if len(requests) != 3 {
		t.Fatalf("expected three provider requests, got %d", len(requests))
	}
	if content := string(mustReadFileForCLITest(t, filepath.Join(temp, "conflict.txt"))); content != "resolved\n" {
		t.Fatalf("expected conflict file repaired by configured provider, got %q", content)
	}
	if message := strings.TrimSpace(runGitOutputForCLITest(t, temp, "log", "-1", "--pretty=%s")); message != "merge: resolve conflict" {
		t.Fatalf("unexpected repair commit message: %q", message)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONReportsAIExposureSummary(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "asset.bin"), []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("chore: update selected files", []string{"README.md", "asset.bin"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected plan preview success exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		ContextPolicy struct {
			Mode             string `json:"mode"`
			FileReadsAllowed bool   `json:"fileReadsAllowed"`
		} `json:"contextPolicy"`
		AIExposure struct {
			SelectedFileCount    int `json:"selectedFileCount"`
			OpaqueChangeCount    int `json:"opaqueChangeCount"`
			SecretBlockerCount   int `json:"secretBlockerCount"`
			ProviderVisibleFiles []struct {
				Path   string `json:"path"`
				Source string `json:"source"`
				Opaque bool   `json:"opaque"`
			} `json:"providerVisibleFiles"`
			DiffBudget struct {
				MaxBytes      int  `json:"maxBytes"`
				UsedBytes     int  `json:"usedBytes"`
				OriginalBytes int  `json:"originalBytes"`
				Truncated     bool `json:"truncated"`
			} `json:"diffBudget"`
			ReadBudget struct {
				MaxBytes  int `json:"maxBytes"`
				UsedBytes int `json:"usedBytes"`
			} `json:"readBudget"`
			OpaqueChanges []struct {
				Path   string `json:"path"`
				Reason string `json:"reason"`
			} `json:"opaqueChanges"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.ContextPolicy.Mode != "bounded" || !payload.ContextPolicy.FileReadsAllowed {
		t.Fatalf("unexpected context policy: %+v", payload.ContextPolicy)
	}
	if payload.AIExposure.SelectedFileCount != 2 || payload.AIExposure.OpaqueChangeCount != 1 || payload.AIExposure.SecretBlockerCount != 0 {
		t.Fatalf("unexpected exposure counts: %+v", payload.AIExposure)
	}
	if len(payload.AIExposure.ProviderVisibleFiles) != 2 {
		t.Fatalf("unexpected provider visible files: %+v", payload.AIExposure.ProviderVisibleFiles)
	}
	for _, file := range payload.AIExposure.ProviderVisibleFiles {
		if file.Path == "asset.bin" && (!file.Opaque || file.Source != "metadata") {
			t.Fatalf("opaque provider visible file should expose metadata only: %+v", file)
		}
	}
	if payload.AIExposure.DiffBudget.MaxBytes <= 0 || payload.AIExposure.DiffBudget.UsedBytes <= 0 || payload.AIExposure.DiffBudget.OriginalBytes <= 0 || payload.AIExposure.DiffBudget.Truncated {
		t.Fatalf("unexpected diff budget: %+v", payload.AIExposure.DiffBudget)
	}
	if payload.AIExposure.ReadBudget.MaxBytes <= 0 {
		t.Fatalf("unexpected read budget: %+v", payload.AIExposure.ReadBudget)
	}
	if len(payload.AIExposure.OpaqueChanges) != 1 || payload.AIExposure.OpaqueChanges[0].Path != "asset.bin" || payload.AIExposure.OpaqueChanges[0].Reason != "binary" {
		t.Fatalf("unexpected opaque changes: %+v", payload.AIExposure.OpaqueChanges)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONNoSelectedChanges(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "README.md")
	runGitForCLITest(t, temp, "commit", "-m", "chore: initial")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected no_changes success exit, got %d stderr=%q", exitCode, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["status"] != "no_changes" || payload["ok"] != true {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONBlocksSelectedSecret(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	secret := "sk_" + strings.Repeat("a", 32)
	if err := os.WriteFile(filepath.Join(temp, "secret.txt"), []byte("api_key = '"+secret+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected secret blocker error exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Status  string `json:"status"`
		OK      bool   `json:"ok"`
		Secrets []struct {
			Path    string `json:"path"`
			Code    string `json:"code"`
			Excerpt string `json:"excerpt"`
		} `json:"secretBlockers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "secret_blocked" || payload.OK || len(payload.Secrets) == 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Secrets[0].Path != "secret.txt" || strings.Contains(payload.Secrets[0].Excerpt, secret) {
		t.Fatalf("secret blocker was not attributed or redacted: %+v", payload.Secrets[0])
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONDiffOnlyDoesNotReadWorkingTreeFiles(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	secret := "sk_" + strings.Repeat("c", 32)
	if err := os.WriteFile(filepath.Join(temp, "secret.txt"), []byte("api_key = '"+secret+"'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json", "--diff-only"}, Runtime{
		CWD:            temp,
		Env:            configuredCLITestEnv(filepath.Join(temp, ".cnm-home")),
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("chore: document secret placeholder", []string{"secret.txt"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected plan preview success exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Status        string `json:"status"`
		ContextPolicy struct {
			Mode             string `json:"mode"`
			FileReadsAllowed bool   `json:"fileReadsAllowed"`
		} `json:"contextPolicy"`
		AIExposure struct {
			SecretBlockerCount int `json:"secretBlockerCount"`
			ReadBudget         struct {
				UsedBytes int `json:"usedBytes"`
			} `json:"readBudget"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Status != "plan_preview" {
		t.Fatalf("unexpected status: %+v", payload)
	}
	if payload.ContextPolicy.Mode != "diff_only" || payload.ContextPolicy.FileReadsAllowed {
		t.Fatalf("unexpected context policy: %+v", payload.ContextPolicy)
	}
	if payload.AIExposure.SecretBlockerCount != 0 || payload.AIExposure.ReadBudget.UsedBytes != 0 {
		t.Fatalf("diff-only should not read working tree files: %+v", payload.AIExposure)
	}
}

func TestExecuteWithRuntimeAutoDryRunJSONReportsPreferenceSources(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-compatible\",\n  \"model\": \"user-model\",\n  \"standingInstruction\": \"private user instruction\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseURL := "https://api.example.test/v1"
	if err := os.WriteFile(filepath.Join(temp, ".cnmrc.json"), []byte("{\n  \"promptStyle\": \"google\",\n  \"standingInstruction\": \"shared project instruction\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("change\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"auto", "--dry-run", "--json"}, Runtime{
		CWD: temp,
		Env: map[string]string{
			"CNM_HOME":             home,
			"CNM_API_KEY":          "sk_" + strings.Repeat("e", 32),
			"CNM_BASE_URL":         baseURL,
			"CNM_MESSAGE_LANGUAGE": "zh-CN",
		},
		Stdout:         &stdout,
		Stderr:         &stderr,
		Stdin:          strings.NewReader(""),
		IsTTY:          false,
		CommitProvider: autoCommitProviderForCLITest("docs: update readme", []string{"README.md"}),
	})

	if exitCode != 0 {
		t.Fatalf("expected plan preview success exit, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "sk_") || strings.Contains(stdout.String(), "private user instruction") || strings.Contains(stdout.String(), "shared project instruction") {
		t.Fatalf("machine output should not expose secret or instruction text:\n%s", stdout.String())
	}
	var payload struct {
		AIExposure struct {
			PreferenceSources struct {
				Provider            string `json:"provider"`
				Model               string `json:"model"`
				APIKey              string `json:"apiKey"`
				PromptStyle         string `json:"promptStyle"`
				MessageLanguage     string `json:"messageLanguage"`
				StandingInstruction string `json:"standingInstruction"`
			} `json:"preferenceSources"`
		} `json:"aiExposure"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	sources := payload.AIExposure.PreferenceSources
	if sources.Provider != "user_config" || sources.Model != "user_config" || sources.APIKey != "env" {
		t.Fatalf("unexpected private preference sources: %+v", sources)
	}
	if sources.PromptStyle != "project_config" || sources.MessageLanguage != "env" || sources.StandingInstruction != "project_config,user_config" {
		t.Fatalf("unexpected shared preference sources: %+v", sources)
	}
}

func TestExecuteWithRuntimeDoesNotWriteDebugLogByDefault(t *testing.T) {
	temp := t.TempDir()
	debugPath := filepath.Join(temp, "cnm-debug.jsonl")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"--version"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected version success, got %d", exitCode)
	}
	if _, err := os.Stat(debugPath); !os.IsNotExist(err) {
		t.Fatalf("debug log should not be written by default, stat err=%v", err)
	}
}

func TestExecuteWithRuntimeDebugLogIsExplicitLocalAndConservative(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	debugPath := filepath.Join(temp, "logs", "cnm-debug.jsonl")
	secret := "sk_test_debug_secret_1234567890"
	standingInstruction := "private standing instruction with business context"
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-responses\",\n  \"model\": \"gpt-5-mini\",\n  \"standingInstruction\": \""+standingInstruction+"\",\n  \"apiKey\": \""+secret+"\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "list", "--json"}, Runtime{
		CWD: temp,
		Env: map[string]string{
			"CNM_HOME":      home,
			"CNM_DEBUG_LOG": debugPath,
		},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected config list success, got %d stdout=%s stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	content, err := os.ReadFile(debugPath)
	if err != nil {
		t.Fatalf("expected explicit debug log at %s: %v", debugPath, err)
	}
	logText := string(content)
	for _, forbidden := range []string{secret, standingInstruction, "apiKey", "standingInstruction", "diff --git", "prompt", "response"} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf("debug log leaked %q: %s", forbidden, logText)
		}
	}
	var event struct {
		SchemaVersion int    `json:"schemaVersion"`
		Event         string `json:"event"`
		Command       string `json:"command"`
		ExitCode      int    `json:"exitCode"`
		Timestamp     string `json:"timestamp"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(content), &event); err != nil {
		t.Fatalf("debug log should be JSONL event: %v\n%s", err, logText)
	}
	if event.SchemaVersion != 1 || event.Event != "command_finished" || event.Command != "cnm config" || event.ExitCode != 0 || event.Timestamp == "" {
		t.Fatalf("unexpected debug event: %+v", event)
	}
}

func runGitForCLITest(t *testing.T, cwd string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = cwd
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func runGitOutputForCLITest(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = cwd
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func mustReadFileForCLITest(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func autoCommitProviderForCLITest(message string, files []string) *runtimex.FakeProviderTracer {
	return autoCommitProviderWithStepsForCLITest([]runtimex.CreateCommitInput{{Message: message, Files: files}})
}

func autoCommitProviderWithStepsForCLITest(commits []runtimex.CreateCommitInput) *runtimex.FakeProviderTracer {
	return autoCommitProviderWithPlanForCLITest("", commits, nil)
}

func autoCommitProviderWithPlanForCLITest(kind string, commits []runtimex.CreateCommitInput, limitations []runtimex.CreateSplitLimitation) *runtimex.FakeProviderTracer {
	return &runtimex.FakeProviderTracer{Steps: []runtimex.FakeProviderStep{
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "inspect", Name: runtimex.ToolInspectCommitScope}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "diff", Name: runtimex.ToolGetDiff}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "create", Name: runtimex.ToolCreateCommits, Arguments: createCommitsPlanArgsForCLITest(kind, commits, limitations)}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "finish", Name: runtimex.ToolFinish, Arguments: map[string]any{"message": "done"}}}},
	}}
}

func autoCommitProviderWithRetryForCLITest(first, second runtimex.CreateCommitInput) *runtimex.FakeProviderTracer {
	return &runtimex.FakeProviderTracer{Steps: []runtimex.FakeProviderStep{
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "inspect", Name: runtimex.ToolInspectCommitScope}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "create-first", Name: runtimex.ToolCreateCommits, Arguments: createCommitsArgsForCLITest([]runtimex.CreateCommitInput{first})}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "create-second", Name: runtimex.ToolCreateCommits, Arguments: createCommitsArgsForCLITest([]runtimex.CreateCommitInput{second})}}},
		{ToolCalls: []runtimex.ToolCallRequest{{ID: "finish", Name: runtimex.ToolFinish, Arguments: map[string]any{"message": "done"}}}},
	}}
}

func createCommitsArgsForCLITest(commits []runtimex.CreateCommitInput) map[string]any {
	return createCommitsPlanArgsForCLITest("", commits, nil)
}

func createCommitsPlanArgsForCLITest(kind string, commits []runtimex.CreateCommitInput, limitations []runtimex.CreateSplitLimitation) map[string]any {
	items := make([]any, 0, len(commits))
	for _, commit := range commits {
		files := make([]any, 0, len(commit.Files))
		for _, file := range commit.Files {
			files = append(files, file)
		}
		items = append(items, map[string]any{"message": commit.Message, "files": files})
	}
	args := map[string]any{"commits": items}
	if strings.TrimSpace(kind) != "" {
		args["kind"] = kind
	}
	if len(limitations) > 0 {
		limitationItems := make([]any, 0, len(limitations))
		for _, limitation := range limitations {
			limitationItems = append(limitationItems, map[string]any{"code": limitation.Code, "message": limitation.Message, "fallback": limitation.Fallback})
		}
		args["splitLimitations"] = limitationItems
	}
	return args
}

type roundTripFuncForCLITest func(request *http.Request) (*http.Response, error)

func (f roundTripFuncForCLITest) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func configuredCLITestEnv(home string) map[string]string {
	return map[string]string{
		"CNM_HOME":    home,
		"CNM_API_KEY": "sk_test_cli_configured_1234567890",
	}
}

type cliWritableSecretStore struct {
	keys map[string]string
}

func (s *cliWritableSecretStore) GetAPIKey(provider config.ProviderType) (*string, error) {
	if s.keys == nil {
		return nil, nil
	}
	value, ok := s.keys[string(provider)]
	if !ok {
		return nil, nil
	}
	return &value, nil
}

func (s *cliWritableSecretStore) SetAPIKey(provider config.ProviderType, apiKey string) error {
	if s.keys == nil {
		s.keys = map[string]string{}
	}
	s.keys[string(provider)] = apiKey
	return nil
}

func TestExecuteWithRuntimeInitJSONDryRun(t *testing.T) {
	temp := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init", "--json", "--dry-run"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["status"] != "dry_run" || payload["command"] != "cnm init" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeInitNonTTYRequiresExplicitConfig(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 2 {
		t.Fatalf("expected non-TTY init usage error, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "cnm init requires interactive TTY input or explicit flags") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("non-TTY init without explicit config should not write config, stat err=%v", err)
	}
}

func TestExecuteWithRuntimeInitJSONDryRunAcceptsRedesignedPreferences(t *testing.T) {
	temp := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init", "--json", "--dry-run", "--message-language", "zh-CN", "--standing-instruction", "Prefer concise subjects."}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exitCode, stderr.String())
	}
	var payload struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Config["messageLanguage"] != "zh-CN" || payload.Config["standingInstruction"] != "Prefer concise subjects." {
		t.Fatalf("unexpected init config: %+v", payload.Config)
	}
}

func TestExecuteWithRuntimeInitTTYRunsInteractiveOnboarding(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader("openai-compatible\ngpt-5.1\nhttps://api.example.test/v1\nconventional\nzh-CN\nPrefer concise subjects.\nsk_test_init_secret_1234567890\n"),
		IsTTY:       true,
		SecretStore: secretStore,
	})

	if exitCode != 0 {
		t.Fatalf("expected interactive init success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	for _, expected := range []string{"Onboarding", "Provider", "Model", "Base URL", "Prompt style", "Message language", "Standing Instruction", "API key", "Initialized user config"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("expected interactive init output to contain %q:\n%s", expected, stdout.String())
		}
	}
	if secretStore.keys["openai-compatible"] != "sk_test_init_secret_1234567890" {
		t.Fatalf("expected API key in Secret Store, got %+v", secretStore.keys)
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(loaded)
	for _, expected := range []string{"openai-compatible", "gpt-5.1", "https://api.example.test/v1", "conventional", "zh-CN", "Prefer concise subjects."} {
		if !strings.Contains(configText, expected) {
			t.Fatalf("expected stored config to contain %q:\n%s", expected, configText)
		}
	}
	if strings.Contains(configText, "apiKey") || strings.Contains(configText, "sk_test") {
		t.Fatalf("interactive init should not write plaintext API key: %s", configText)
	}
}

func TestExecuteWithRuntimeInitStoresAPIKeyInSecretStoreByDefault(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init", "--json", "--provider", "openai-responses", "--api-key", "sk_test_secret_store_1234567890"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader(""),
		IsTTY:       false,
		SecretStore: secretStore,
	})

	if exitCode != 0 {
		t.Fatalf("expected init success, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if strings.Contains(stderr.String(), "plaintext") {
		t.Fatalf("default Secret Store path should not warn about plaintext: %q", stderr.String())
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(loaded), "apiKey") || strings.Contains(string(loaded), "sk_test") {
		t.Fatalf("api key should not be written to user config by default: %s", string(loaded))
	}
	if secretStore.keys["openai-responses"] != "sk_test_secret_store_1234567890" {
		t.Fatalf("expected api key in secret store, got %+v", secretStore.keys)
	}
	var payload struct {
		APIKeySave struct {
			Source string `json:"source"`
			Stored bool   `json:"stored"`
		} `json:"apiKeySave"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.APIKeySave.Source != "secret_store" || !payload.APIKeySave.Stored {
		t.Fatalf("unexpected api key save payload: %+v", payload.APIKeySave)
	}
}

func TestExecuteWithRuntimeConfigListReadsAPIKeyFromSecretStore(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{keys: map[string]string{"openai-responses": "sk_test_secret_store_1234567890"}}
	providerConfig := "{\n  \"provider\": \"openai-responses\",\n  \"model\": \"gpt-5-mini\"\n}\n"
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(providerConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "list", "--json"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader(""),
		IsTTY:       false,
		SecretStore: secretStore,
	})

	if exitCode != 0 {
		t.Fatalf("expected config list success, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["apiKey"] != "[redacted]" || payload["apiKeySource"] != "secret_store" {
		t.Fatalf("expected secret store API key source, got %+v", payload)
	}
}

func TestExecuteWithRuntimeInitPlaintextAPIKeyRequiresExplicitOptIn(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"init", "--json", "--provider", "openai-responses", "--api-key", "sk_test_plaintext_1234567890", "--plaintext-api-key"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected init success, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "plaintext") {
		t.Fatalf("expected plaintext warning, got %q", stderr.String())
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(loaded), "apiKey") || !strings.Contains(string(loaded), "sk_test_plaintext_1234567890") {
		t.Fatalf("expected explicit plaintext api key in user config: %s", string(loaded))
	}
}

func TestExecuteWithRuntimeDoctorJSON(t *testing.T) {
	temp := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"doctor", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected error exit when doctor finds issues, got %d", exitCode)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Issues []struct {
			Code string `json:"code"`
		} `json:"issues"`
		Summary struct {
			Errors int `json:"errors"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload.Summary.Errors == 0 {
		t.Fatalf("expected doctor errors, got %+v", payload)
	}
	codes := make([]string, 0, len(payload.Issues))
	for _, issue := range payload.Issues {
		codes = append(codes, issue.Code)
	}
	if !containsString(codes, "provider_config_missing") || !containsString(codes, "api_key_missing") {
		t.Fatalf("unexpected issue codes: %v", codes)
	}
}

func TestExecuteWithRuntimeDoctorJSONIncludesProviderCapabilityAndExplicitProbe(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	home := filepath.Join(temp, ".cnm-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"anthropic-messages\",\n  \"model\": \"claude-sonnet-4-20250514\",\n  \"apiKey\": \"sk_test_doctor_probe_1234567890\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := ExecuteWithRuntime([]string{"doctor", "--json", "--probe-provider"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected doctor success, got %d stderr=%q stdout=%s", exitCode, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload struct {
		Checks struct {
			ProviderCapability struct {
				Status     string `json:"status"`
				Capability struct {
					Provider          string `json:"provider"`
					Protocol          string `json:"protocol"`
					NativeToolCalls   bool   `json:"nativeToolCalls"`
					InteractiveRepair bool   `json:"interactiveRepair"`
				} `json:"capability"`
			} `json:"providerCapability"`
		} `json:"checks"`
		Probe *struct {
			Status string `json:"status"`
		} `json:"probe"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	capability := payload.Checks.ProviderCapability.Capability
	if payload.Checks.ProviderCapability.Status != "pass" || capability.Provider != "anthropic-messages" || capability.Protocol != "anthropic_messages" || !capability.NativeToolCalls || !capability.InteractiveRepair {
		t.Fatalf("unexpected capability payload: %+v", payload.Checks.ProviderCapability)
	}
	if payload.Probe == nil || payload.Probe.Status != "skipped" {
		t.Fatalf("expected explicit probe result, got %+v", payload.Probe)
	}
}

func TestExecuteWithRuntimeConfigListJSON(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-responses\",\n  \"model\": \"gpt-5-mini\",\n  \"apiKey\": \"sk_test_1234567890\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "list", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["provider"] != "openai-responses" || payload["apiKey"] != "[redacted]" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestExecuteWithRuntimeConfigListJSONIncludesRedesignedPreferences(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-responses\",\n  \"model\": \"gpt-5-mini\",\n  \"messageLanguage\": \"zh-CN\",\n  \"standingInstruction\": \"Prefer concise subjects.\",\n  \"apiKey\": \"sk_test_1234567890\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, ".cnmrc.json"), []byte("{\n  \"recommendedProvider\": \"google-gemini\",\n  \"recommendedModel\": \"gemini-team\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "list", "--json"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exitCode, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json parse failed: %v\n%s", err, stdout.String())
	}
	if payload["messageLanguage"] != "zh-CN" || payload["standingInstruction"] != "Prefer concise subjects." {
		t.Fatalf("unexpected redesigned preferences: %+v", payload)
	}
	if payload["recommendedProvider"] != "google-gemini" || payload["recommendedModel"] != "gemini-team" {
		t.Fatalf("unexpected provider recommendation: %+v", payload)
	}
	if payload["apiKey"] != "[redacted]" || payload["apiKeySource"] != "plaintext_config" {
		t.Fatalf("unexpected api key source: %+v", payload)
	}
}

func TestExecuteWithRuntimeConfigSetStandingInstructionJSON(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "set", "--json", "standingInstruction", "Prefer concise subjects."}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exitCode, stderr.String())
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(loaded), "standingInstruction") || strings.Contains(string(loaded), "customPrompt") {
		t.Fatalf("unexpected stored config: %s", string(loaded))
	}
}

func TestExecuteWithRuntimeConfigRejectsCustomPromptKey(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config", "set", "customPrompt", "legacy prompt"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": home},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  false,
	})

	if exitCode != 1 {
		t.Fatalf("expected customPrompt rejection, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Unsupported config key `customPrompt`") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("customPrompt rejection should not write config, stat err=%v", err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestExecuteWithRuntimeAutoTUIConflictChecksConfigBeforeHandoffWhenAPIKeyMissing(t *testing.T) {
	temp := t.TempDir()
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tuiCalled := false
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:    temp,
		Env:    map[string]string{"CNM_HOME": filepath.Join(temp, ".cnm-home")},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
		IsTTY:  true,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			tuiCalled = true
			return tui.Result{Cancelled: true}, nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected config gate failure exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if tuiCalled {
		t.Fatalf("expected config gate to prevent TUI handoff")
	}
	if !strings.Contains(stderr.String(), "No API key is configured") {
		t.Fatalf("expected api key config error, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("config gate path should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeAutoTUIConflictChecksConfigBeforeHandoffWhenBaseURLMissing(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	runGitForCLITest(t, temp, "init")
	runGitForCLITest(t, temp, "config", "user.name", "CNM Test")
	runGitForCLITest(t, temp, "config", "user.email", "cnm@example.test")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-compatible\",\n  \"model\": \"compat-model\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "add", "conflict.txt")
	runGitForCLITest(t, temp, "commit", "-m", "chore: base")
	initialBranch := strings.TrimSpace(runGitOutputForCLITest(t, temp, "branch", "--show-current"))
	runGitForCLITest(t, temp, "checkout", "-b", "other")
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: other")
	runGitForCLITest(t, temp, "checkout", initialBranch)
	if err := os.WriteFile(filepath.Join(temp, "conflict.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitForCLITest(t, temp, "commit", "-am", "chore: main")
	merge := exec.Command("git", "merge", "other")
	merge.Dir = temp
	_ = merge.Run()
	headBefore := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tuiCalled := false
	secretStore := &cliWritableSecretStore{keys: map[string]string{"openai-compatible": "test_store_value_1234567890"}}
	exitCode := ExecuteWithRuntime([]string{"auto", "--tui"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader(""),
		IsTTY:       true,
		SecretStore: secretStore,
		TUIRunner: func(input tui.ModelInput, runtime tui.Runtime) (tui.Result, error) {
			tuiCalled = true
			return tui.Result{Cancelled: true}, nil
		},
	})

	if exitCode != 1 {
		t.Fatalf("expected config gate failure exit, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if tuiCalled {
		t.Fatalf("expected config gate to prevent TUI handoff")
	}
	if !strings.Contains(stderr.String(), "requires baseURL") {
		t.Fatalf("expected baseURL config error, got %q", stderr.String())
	}
	if headAfter := strings.TrimSpace(runGitOutputForCLITest(t, temp, "rev-parse", "HEAD")); headAfter != headBefore {
		t.Fatalf("config gate path should not create commit: %s -> %s", headBefore, headAfter)
	}
}

func TestExecuteWithRuntimeConfigPanelStoresAPIKeyForUpdatedProvider(t *testing.T) {
	temp := t.TempDir()
	home := filepath.Join(temp, ".cnm-home")
	secretStore := &cliWritableSecretStore{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := ExecuteWithRuntime([]string{"config"}, Runtime{
		CWD:         temp,
		Env:         map[string]string{"CNM_HOME": home},
		Stdout:      &stdout,
		Stderr:      &stderr,
		Stdin:       strings.NewReader(""),
		IsTTY:       true,
		SecretStore: secretStore,
		ConfigPanelRunner: func(input tui.ConfigPanelInput, runtime tui.Runtime) (tui.ConfigPanelResult, error) {
			if err := input.WriteValue(config.ConfigKeyProvider, "anthropic-messages"); err != nil {
				return tui.ConfigPanelResult{}, err
			}
			if _, _, err := input.Reload(); err != nil {
				return tui.ConfigPanelResult{}, err
			}
			if err := input.WriteValue(config.ConfigKeyAPIKey, "test_panel_value_1234567890"); err != nil {
				return tui.ConfigPanelResult{}, err
			}
			return tui.ConfigPanelResult{Saved: 2}, nil
		},
	})

	if exitCode != 0 {
		t.Fatalf("expected config panel success, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if secretStore.keys["anthropic-messages"] != "test_panel_value_1234567890" {
		t.Fatalf("expected api key in updated provider slot, got %+v", secretStore.keys)
	}
	if _, ok := secretStore.keys["openai-responses"]; ok {
		t.Fatalf("did not expect api key in stale provider slot: %+v", secretStore.keys)
	}
	loaded, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(loaded), "anthropic-messages") {
		t.Fatalf("expected updated provider in user config: %s", string(loaded))
	}
}
