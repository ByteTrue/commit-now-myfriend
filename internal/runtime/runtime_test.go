package runtime

import (
	"errors"
	"testing"
	"time"

	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
)

func TestToolCallRuntimeRunsFakeProviderThroughInspectScopeAndFinish(t *testing.T) {
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "call-1", Name: ToolInspectCommitScope}}},
			{ToolCalls: []ToolCallRequest{{ID: "call-2", Name: ToolFinish, Arguments: map[string]any{"status": "completed", "message": "ready"}}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Tools: NewDomainToolSet(DomainToolSetOptions{
			InspectCommitScope: func() (gitpkg.CommitScope, error) {
				return gitpkg.CommitScope{
					Files:              []gitpkg.FileStatus{{Path: "README.md"}},
					HasSelectedChanges: true,
				}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted || result.Message != "ready" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(provider.ReceivedResults) != 1 {
		t.Fatalf("expected one tool result before finish, got %+v", provider.ReceivedResults)
	}
	if provider.ReceivedResults[0].CallID != "call-1" || !provider.ReceivedResults[0].OK {
		t.Fatalf("unexpected tool result: %+v", provider.ReceivedResults[0])
	}
}

func TestToolCallRuntimeRejectsInvalidToolsWithStructuredFeedbackAndContinues(t *testing.T) {
	mutated := false
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{
				{ID: "call-shell", Name: "shell", Arguments: map[string]any{"command": "git status"}},
				{ID: "call-git", Name: "git_commit", Arguments: map[string]any{"args": []string{"commit"}}},
			}},
			{ToolCalls: []ToolCallRequest{{ID: "call-finish", Name: ToolFinish, Arguments: map[string]any{"message": "recovered"}}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Tools: NewDomainToolSet(DomainToolSetOptions{
			InspectCommitScope: func() (gitpkg.CommitScope, error) {
				mutated = true
				return gitpkg.CommitScope{}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted || result.Message != "recovered" {
		t.Fatalf("runtime did not continue after invalid calls: %+v", result)
	}
	if mutated {
		t.Fatal("invalid tool calls should not execute domain tools")
	}
	if len(provider.ReceivedResults) != 2 {
		t.Fatalf("expected structured errors for two invalid calls, got %+v", provider.ReceivedResults)
	}
	for _, toolResult := range provider.ReceivedResults {
		if toolResult.OK || toolResult.Error == nil || toolResult.Error.Code == "" {
			t.Fatalf("expected structured invalid-call feedback, got %+v", toolResult)
		}
	}
}

func TestToolCallRuntimeExposesRepositoryDomainTools(t *testing.T) {
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{
				{ID: "diff", Name: ToolGetDiff},
				{ID: "read", Name: ToolReadFile, Arguments: map[string]any{"path": "README.md"}},
				{ID: "preview", Name: ToolPreviewCommit, Arguments: map[string]any{"message": "docs: update readme"}},
			}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish, Arguments: map[string]any{"message": "done"}}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Tools: NewDomainToolSet(DomainToolSetOptions{
			GetDiff: func() (DiffResult, error) {
				return DiffResult{Content: "diff --git a/README.md b/README.md\n+updated", Bytes: 42}, nil
			},
			ReadFile: func(path string) (FileReadResult, error) {
				return FileReadResult{Path: path, Content: "updated\n", Bytes: 8}, nil
			},
			PreviewCommit: func(input CommitPreviewInput) (CommitPreviewResult, error) {
				return CommitPreviewResult{Message: input.Message, FileCount: 1}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(provider.ReceivedResults) != 3 {
		t.Fatalf("expected three domain tool results, got %+v", provider.ReceivedResults)
	}
	for _, toolResult := range provider.ReceivedResults {
		if !toolResult.OK {
			t.Fatalf("domain tool failed: %+v", toolResult)
		}
	}
}

func TestToolCallRuntimeEnforcesDiffOnlyContextPolicy(t *testing.T) {
	readCalled := false
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "read", Name: ToolReadFile, Arguments: map[string]any{"path": "README.md"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider:      provider,
		ContextPolicy: gitpkg.ContextPolicy{Mode: gitpkg.ContextPolicyModeDiffOnly, FileReadsAllowed: false},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			ReadFile: func(path string) (FileReadResult, error) {
				readCalled = true
				return FileReadResult{Path: path}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("runtime should continue after policy rejection: %+v", result)
	}
	if readCalled {
		t.Fatal("read_file should not run under diff-only context policy")
	}
	if len(provider.ReceivedResults) != 1 || provider.ReceivedResults[0].OK || provider.ReceivedResults[0].Error == nil || provider.ReceivedResults[0].Error.Code != "context_policy_denied" {
		t.Fatalf("expected context policy denial, got %+v", provider.ReceivedResults)
	}
}

func TestToolCallRuntimeRequiresInspectBeforeCreateCommitsAndContinues(t *testing.T) {
	created := 0
	createArgs := map[string]any{
		"commits": []any{map[string]any{
			"message": "docs: update readme",
			"files":   []any{"README.md"},
		}},
	}
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "create-before-inspect", Name: ToolCreateCommits, Arguments: createArgs}}},
			{ToolCalls: []ToolCallRequest{{ID: "inspect", Name: ToolInspectCommitScope}}},
			{ToolCalls: []ToolCallRequest{{ID: "create-after-inspect", Name: ToolCreateCommits, Arguments: createArgs}}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Tools: NewDomainToolSet(DomainToolSetOptions{
			InspectCommitScope: func() (gitpkg.CommitScope, error) {
				return gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "README.md"}}, HasSelectedChanges: true}, nil
			},
			CreateCommits: func(input CreateCommitsInput) (CreateCommitsResult, error) {
				created++
				return CreateCommitsResult{Plan: CreateCommitPlanResult{Kind: "single", Commits: []CreateCommitPlanCommit{{Message: input.Commits[0].Message, Files: input.Commits[0].Files}}}}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if created != 1 {
		t.Fatalf("create_commits should run only after inspect_commit_scope, ran %d times", created)
	}
	if len(provider.ReceivedResults) != 3 || provider.ReceivedResults[0].OK || provider.ReceivedResults[0].Error == nil || provider.ReceivedResults[0].Error.Code != "inspect_before_create_required" || !provider.ReceivedResults[2].OK {
		t.Fatalf("unexpected provider feedback: %+v", provider.ReceivedResults)
	}
}

func TestToolCallRuntimeEnforcesToolCallLimit(t *testing.T) {
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "one", Name: ToolGetDiff}}},
			{ToolCalls: []ToolCallRequest{{ID: "two", Name: ToolGetDiff}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Limits:   LoopLimits{MaxToolCalls: 1},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			GetDiff: func() (DiffResult, error) {
				return DiffResult{Content: "diff"}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusLimited || result.LimitReason != "max_tool_calls" {
		t.Fatalf("expected max tool calls limit, got %+v", result)
	}
	if len(result.Calls) != 1 {
		t.Fatalf("expected only first call to execute, got %+v", result.Calls)
	}
}

func TestToolCallRuntimeEnforcesProviderRetryLimit(t *testing.T) {
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{Err: errors.New("temporary provider failure")},
			{Err: errors.New("temporary provider failure")},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Limits:   LoopLimits{MaxProviderRetries: 1},
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("provider retry exhaustion should return a recovery-ready result, got %v", err)
	}
	if result.Status != RunStatusLimited || result.LimitReason != "max_provider_retries" {
		t.Fatalf("expected provider retry limit, got %+v", result)
	}
}

func TestToolCallRuntimeEnforcesDurationLimit(t *testing.T) {
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "one", Name: ToolGetDiff}}},
			{ToolCalls: []ToolCallRequest{{ID: "two", Name: ToolGetDiff}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		Limits:   LoopLimits{MaxDuration: time.Nanosecond},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			GetDiff: func() (DiffResult, error) {
				time.Sleep(time.Millisecond)
				return DiffResult{Content: "diff"}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusLimited || result.LimitReason != "max_duration" {
		t.Fatalf("expected duration limit, got %+v", result)
	}
}

func TestToolCallRuntimeEnforcesReadBeforeWriteGuardrailForRepairWrites(t *testing.T) {
	repairs := 0
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "repair-before-read", Name: ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "resolved\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "read", Name: ToolReadFile, Arguments: map[string]any{"path": "conflict.txt"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "repair-after-read", Name: ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "resolved\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		RepairPolicy: RepairPolicy{
			ConfirmWrite: func(input RepairFileInput) (bool, error) { return true, nil },
		},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			ReadFile: func(path string) (FileReadResult, error) {
				return FileReadResult{Path: path, Content: "<<<<<<< HEAD\n", Bytes: 13}, nil
			},
			RepairFile: func(input RepairFileInput) (RepairFileResult, error) {
				repairs++
				return RepairFileResult{Path: input.Path, Applied: true}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repairs != 1 {
		t.Fatalf("expected only repair after read to run, got %d", repairs)
	}
	if len(provider.ReceivedResults) < 3 || provider.ReceivedResults[0].OK || provider.ReceivedResults[0].Error == nil || provider.ReceivedResults[0].Error.Code != "read_before_write_required" {
		t.Fatalf("expected read-before-write rejection first, got %+v", provider.ReceivedResults)
	}
	if !provider.ReceivedResults[2].OK {
		t.Fatalf("expected repair after read to succeed, got %+v", provider.ReceivedResults[2])
	}
}

func TestToolCallRuntimeRestrictsRepairWritesToEligibleConflictFiles(t *testing.T) {
	repairedPaths := []string{}
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "read-readme", Name: ToolReadFile, Arguments: map[string]any{"path": "README.md"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "repair-readme", Name: ToolRepairFile, Arguments: map[string]any{"path": "README.md", "content": "not allowed\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "read-conflict", Name: ToolReadFile, Arguments: map[string]any{"path": "conflict.txt"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "repair-conflict", Name: ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "resolved\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		RepairPolicy: RepairPolicy{
			AllowedPaths: []string{"conflict.txt"},
			ConfirmWrite: func(input RepairFileInput) (bool, error) { return true, nil },
		},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			ReadFile: func(path string) (FileReadResult, error) {
				return FileReadResult{Path: path, Content: "content\n", Bytes: 8}, nil
			},
			RepairFile: func(input RepairFileInput) (RepairFileResult, error) {
				repairedPaths = append(repairedPaths, input.Path)
				return RepairFileResult{Path: input.Path, Applied: true}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(repairedPaths) != 1 || repairedPaths[0] != "conflict.txt" {
		t.Fatalf("expected only conflict file repair, got %v", repairedPaths)
	}
	if len(provider.ReceivedResults) < 4 || provider.ReceivedResults[1].OK || provider.ReceivedResults[1].Error == nil || provider.ReceivedResults[1].Error.Code != "repair_path_not_allowed" {
		t.Fatalf("expected README repair rejection, got %+v", provider.ReceivedResults)
	}
	if !provider.ReceivedResults[3].OK {
		t.Fatalf("expected allowed conflict repair to succeed, got %+v", provider.ReceivedResults[3])
	}
}

func TestToolCallRuntimeRequiresDeveloperConfirmationBeforeRepairWrites(t *testing.T) {
	repairs := 0
	confirmations := 0
	provider := &FakeProviderTracer{
		Steps: []FakeProviderStep{
			{ToolCalls: []ToolCallRequest{{ID: "read", Name: ToolReadFile, Arguments: map[string]any{"path": "conflict.txt"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "repair-denied", Name: ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "denied\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "repair-confirmed", Name: ToolRepairFile, Arguments: map[string]any{"path": "conflict.txt", "content": "resolved\n"}}}},
			{ToolCalls: []ToolCallRequest{{ID: "finish", Name: ToolFinish}}},
		},
	}
	runtime := NewToolCallRuntime(ToolCallRuntimeOptions{
		Provider: provider,
		RepairPolicy: RepairPolicy{
			AllowedPaths: []string{"conflict.txt"},
			ConfirmWrite: func(input RepairFileInput) (bool, error) {
				confirmations++
				return input.Content == "resolved\n", nil
			},
		},
		Tools: NewDomainToolSet(DomainToolSetOptions{
			ReadFile: func(path string) (FileReadResult, error) {
				return FileReadResult{Path: path, Content: "<<<<<<< HEAD\n", Bytes: 13}, nil
			},
			RepairFile: func(input RepairFileInput) (RepairFileResult, error) {
				repairs++
				return RepairFileResult{Path: input.Path, Applied: true}, nil
			},
		}),
	})

	result, err := runtime.Run()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repairs != 1 || confirmations != 2 {
		t.Fatalf("expected one confirmed repair after two confirmation prompts, repairs=%d confirmations=%d", repairs, confirmations)
	}
	if len(provider.ReceivedResults) < 3 || provider.ReceivedResults[1].OK || provider.ReceivedResults[1].Error == nil || provider.ReceivedResults[1].Error.Code != "repair_confirmation_required" {
		t.Fatalf("expected denied repair to require confirmation, got %+v", provider.ReceivedResults)
	}
	if !provider.ReceivedResults[2].OK {
		t.Fatalf("expected confirmed repair to succeed, got %+v", provider.ReceivedResults[2])
	}
}
