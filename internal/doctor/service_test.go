package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
)

func TestRunReportsMissingProviderSetupOutsideGitRepos(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	report, err := Run(RunOptions{CWD: cwd, Env: map[string]string{"CNM_HOME": home}})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if report.Command != "cnm doctor" {
		t.Fatalf("unexpected command: %+v", report)
	}
	codes := issueCodes(report.Issues)
	if !contains(codes, "not_git_repository") || !contains(codes, "provider_config_missing") || !contains(codes, "api_key_missing") {
		t.Fatalf("unexpected issues: %v", codes)
	}
	if report.Checks.Repository.IsRepository {
		t.Fatalf("expected non-git repository")
	}
	if report.Checks.EffectiveConfig.Config.APIKey != nil {
		t.Fatalf("expected redacted nil api key")
	}
	serialized, _ := json.Marshal(report)
	if string(serialized) == "" {
		t.Fatal("expected serializable report")
	}
}

func TestRunRedactsUserAndProjectAPIKeys(t *testing.T) {
	repo := createDoctorRepo(t, true, true)
	home := filepath.Join(repo.path, ".cnm-home")
	env := map[string]string{"CNM_HOME": home}

	_, err := config.WriteUserConfigPatch(config.ConfigValues{
		APIKey: ptrString("sk_live_user_secret_1234567890"),
		Model:  ptrString("gpt-5-mini"),
		Provider: func() *config.ProviderType {
			value := config.ProviderOpenAIResponses
			return &value
		}(),
	}, config.ConfigEnvironment{CWD: repo.path, Env: env})
	if err != nil {
		t.Fatalf("WriteUserConfigPatch error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.path, ".cnmrc.json"), []byte("{\n  \"apiKey\": \"sk_project_secret_1234567890\",\n  \"model\": \"project-model\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(RunOptions{CWD: repo.path, Env: env})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if report.Checks.EffectiveConfig.Config.APIKey == nil || *report.Checks.EffectiveConfig.Config.APIKey != "[redacted]" {
		t.Fatalf("expected redacted api key: %+v", report.Checks.EffectiveConfig.Config)
	}
	if !contains(issueCodes(report.Issues), "project_api_key_ignored") {
		t.Fatalf("expected project_api_key_ignored: %v", issueCodes(report.Issues))
	}
	serialized, _ := json.Marshal(report)
	if containsString(string(serialized), "sk_live_user_secret_1234567890") || containsString(string(serialized), "sk_project_secret_1234567890") {
		t.Fatalf("secret leaked in report: %s", string(serialized))
	}
}

func TestRunWarnsOnBroadUserConfigPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission mode check is unix-specific")
	}
	repo := createDoctorWorkspace(t, true)
	home := filepath.Join(repo.path, ".cnm-home")
	env := map[string]string{"CNM_HOME": home}
	result, err := config.WriteUserConfigPatch(config.ConfigValues{
		APIKey:   func() *string { v := "sk_permissions_secret_1234567890"; return &v }(),
		Provider: func() *config.ProviderType { v := config.ProviderOpenAIResponses; return &v }(),
	}, config.ConfigEnvironment{CWD: repo.path, Env: env})
	if err != nil {
		t.Fatalf("WriteUserConfigPatch error: %v", err)
	}
	if err := os.Chmod(result.Path, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Run(RunOptions{CWD: repo.path, Env: env})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !contains(issueCodes(report.Issues), "user_config_permissions_insecure") {
		t.Fatalf("expected permission warning: %v", issueCodes(report.Issues))
	}
}

func TestRunSurfacesMissingGitIdentity(t *testing.T) {
	repo := createDoctorRepo(t, false, false)
	repo.writeFile(t, "file.txt", []byte("content\n"))
	repo.git(t, "add", "file.txt")
	isolatedHome := filepath.Join(repo.path, "isolated-home")
	if err := os.MkdirAll(isolatedHome, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := Run(RunOptions{CWD: repo.path, Env: map[string]string{"CNM_HOME": filepath.Join(isolatedHome, ".cnm-home"), "HOME": isolatedHome, "XDG_CONFIG_HOME": filepath.Join(isolatedHome, ".config")}})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !contains(issueCodes(report.Issues), "git_identity_missing") {
		t.Fatalf("expected git identity issue: %v", issueCodes(report.Issues))
	}
	if report.Paths.UserConfigHome != filepath.Join(isolatedHome, ".cnm-home") {
		t.Fatalf("unexpected user config home: %+v", report.Paths)
	}
	if report.Checks.EffectiveConfig.Sources.APIKey != "missing" {
		t.Fatalf("expected missing api key source: %+v", report.Checks.EffectiveConfig.Sources)
	}
}

func TestRunReportsProviderCapabilityMetadataWithoutProbe(t *testing.T) {
	repo := createDoctorRepo(t, true, true)
	home := filepath.Join(repo.path, ".cnm-home")
	provider := config.ProviderAnthropic
	if _, err := config.WriteUserConfigPatch(config.ConfigValues{
		Provider: &provider,
		Model:    ptrString("claude-sonnet-4-20250514"),
		APIKey:   ptrString("sk_test_doctor_capability_1234567890"),
	}, config.ConfigEnvironment{CWD: repo.path, Env: map[string]string{"CNM_HOME": home}}); err != nil {
		t.Fatalf("WriteUserConfigPatch error: %v", err)
	}

	report, err := Run(RunOptions{CWD: repo.path, Env: map[string]string{"CNM_HOME": home}})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	capability := report.Checks.ProviderCapability.Capability
	if report.Checks.ProviderCapability.Status != CheckStatusPass || capability.Provider != config.ProviderAnthropic || capability.Protocol != "anthropic_messages" || !capability.NativeToolCalls || !capability.InteractiveRepair {
		t.Fatalf("unexpected provider capability: %+v", report.Checks.ProviderCapability)
	}
	if report.Probe != nil {
		t.Fatalf("doctor should not probe provider by default: %+v", report.Probe)
	}
}

func TestRunExplicitProviderProbeUsesFixedNonRepositoryContent(t *testing.T) {
	repo := createDoctorRepo(t, true, true)
	home := filepath.Join(repo.path, ".cnm-home")
	provider := config.ProviderOpenAIResponses
	if _, err := config.WriteUserConfigPatch(config.ConfigValues{
		Provider: &provider,
		Model:    ptrString("gpt-5-mini"),
		APIKey:   ptrString("sk_test_doctor_probe_1234567890"),
	}, config.ConfigEnvironment{CWD: repo.path, Env: map[string]string{"CNM_HOME": home}}); err != nil {
		t.Fatalf("WriteUserConfigPatch error: %v", err)
	}
	called := false
	report, err := Run(RunOptions{
		CWD:           repo.path,
		Env:           map[string]string{"CNM_HOME": home},
		ProbeProvider: true,
		Probe: func(input ProbeInput) ProbeResult {
			called = true
			if strings.Contains(input.Content, "README") || strings.Contains(input.Content, repo.path) {
				t.Fatalf("probe content should not include repository data: %+v", input)
			}
			if input.Provider != config.ProviderOpenAIResponses || input.Model != "gpt-5-mini" {
				t.Fatalf("unexpected probe input: %+v", input)
			}
			return ProbeResult{Status: "ok", Message: "provider accepted fixed probe"}
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !called || report.Probe == nil || report.Probe.Status != "ok" {
		t.Fatalf("expected explicit provider probe result, called=%v report=%+v", called, report.Probe)
	}
}

type doctorRepo struct{ path string }

func createDoctorRepo(t *testing.T, initialCommit bool, identity bool) *doctorRepo {
	t.Helper()
	repo := &doctorRepo{path: t.TempDir()}
	repo.git(t, "init")
	if identity {
		repo.git(t, "config", "user.name", "CNM Test")
		repo.git(t, "config", "user.email", "cnm@example.test")
	}
	if initialCommit {
		repo.writeFile(t, "README.md", []byte("initial\n"))
		repo.git(t, "add", "README.md")
		repo.git(t, "commit", "-m", "chore: initial")
	}
	return repo
}

func createDoctorWorkspace(t *testing.T, identity bool) *doctorRepo {
	t.Helper()
	repo := &doctorRepo{path: t.TempDir()}
	repo.git(t, "init")
	if identity {
		repo.git(t, "config", "user.name", "CNM Test")
		repo.git(t, "config", "user.email", "cnm@example.test")
	}
	return repo
}

func (r *doctorRepo) git(t *testing.T, args ...string) {
	t.Helper()
	result, err := gitpkg.DefaultCommandRunner(r.path, args, nil)
	if err != nil {
		t.Fatalf("git runner error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("git %v failed: %s %s", args, result.Stdout, result.Stderr)
	}
}

func (r *doctorRepo) writeFile(t *testing.T, relative string, content []byte) {
	t.Helper()
	target := filepath.Join(r.path, relative)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func issueCodes(issues []Issue) []string {
	result := make([]string, 0, len(issues))
	for _, issue := range issues {
		result = append(result, issue.Code)
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsString(value string, target string) bool {
	return strings.Contains(value, target)
}

func ptrString(value string) *string { return &value }
