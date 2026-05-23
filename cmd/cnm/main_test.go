package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIHelpSubprocess(t *testing.T) {
	stdout, stderr, err := runCNM(t, t.TempDir(), nil, "--help")
	if err != nil {
		t.Fatalf("runCNM help error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Usage: cnm") || !strings.Contains(stdout, "cnm auto") || !strings.Contains(stdout, "cnm doctor") {
		t.Fatalf("unexpected help output: %s", stdout)
	}
	if strings.Contains(stdout, "cnm split") {
		t.Fatalf("help should not document removed split command: %s", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestCLIAutoHelpSubprocess(t *testing.T) {
	stdout, stderr, err := runCNM(t, t.TempDir(), nil, "auto", "--help")
	if err != nil {
		t.Fatalf("runCNM auto help error: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Usage: cnm auto") || !strings.Contains(stdout, "Autonomous Commit") || !strings.Contains(stdout, "--tui") {
		t.Fatalf("unexpected auto help output: %s", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestCLIRemovedSplitSubprocess(t *testing.T) {
	stdout, stderr, err := runCNM(t, t.TempDir(), nil, "split")
	if err == nil {
		t.Fatal("expected removed split command to exit non-zero")
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if !strings.Contains(stderr, "removed command 'split'") {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
}

func TestCLIDoctorJSONSubprocess(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	stdout, stderr, err := runCNM(t, cwd, map[string]string{"CNM_HOME": home}, "doctor", "--json")
	if err == nil {
		t.Fatal("expected doctor to exit non-zero when issues exist")
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	var payload struct {
		Summary struct {
			Errors int `json:"errors"`
		} `json:"summary"`
		Issues []struct {
			Code string `json:"code"`
		} `json:"issues"`
	}
	if parseErr := json.Unmarshal([]byte(stdout), &payload); parseErr != nil {
		t.Fatalf("failed to parse doctor json: %v\nstdout=%s", parseErr, stdout)
	}
	if payload.Summary.Errors == 0 {
		t.Fatalf("expected doctor errors, got %+v", payload)
	}
	codes := make([]string, 0, len(payload.Issues))
	for _, issue := range payload.Issues {
		codes = append(codes, issue.Code)
	}
	if !contains(codes, "provider_config_missing") {
		t.Fatalf("unexpected issue codes: %v", codes)
	}
}

func TestCLIConfigListJSONSubprocess(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"provider\": \"openai-responses\",\n  \"model\": \"gpt-5-mini\",\n  \"apiKey\": \"sk_test_1234567890\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runCNM(t, cwd, map[string]string{"CNM_HOME": home}, "config", "list", "--json")
	if err != nil {
		t.Fatalf("runCNM config list error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	var payload map[string]any
	if parseErr := json.Unmarshal([]byte(stdout), &payload); parseErr != nil {
		t.Fatalf("failed to parse config list json: %v\nstdout=%s", parseErr, stdout)
	}
	if payload["provider"] != "openai-responses" || payload["apiKey"] != "[redacted]" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestCLIInitJSONDryRunSubprocess(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	stdout, stderr, err := runCNM(t, cwd, map[string]string{"CNM_HOME": home}, "init", "--json", "--dry-run")
	if err != nil {
		t.Fatalf("runCNM init dry-run error: %v\nstderr=%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %s", stderr)
	}
	var payload map[string]any
	if parseErr := json.Unmarshal([]byte(stdout), &payload); parseErr != nil {
		t.Fatalf("failed to parse init dry-run json: %v\nstdout=%s", parseErr, stdout)
	}
	if payload["status"] != "dry_run" || payload["command"] != "cnm init" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func runCNM(t *testing.T, cwd string, extraEnv map[string]string, args ...string) (string, string, error) {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "cnm-test-binary")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/cnm")
	build.Dir = projectRoot(t)
	build.Env = append(os.Environ(), "GO111MODULE=on")
	buildOutput, buildErr := build.CombinedOutput()
	if buildErr != nil {
		t.Fatalf("failed to build test binary: %v\n%s", buildErr, string(buildOutput))
	}

	cmd := exec.Command(binaryPath, args...)
	if cwd != "" {
		cmd.Dir = cwd
	} else {
		cmd.Dir = projectRoot(t)
	}
	cmd.Env = os.Environ()
	for key, value := range extraEnv {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve caller")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
