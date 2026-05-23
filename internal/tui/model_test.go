package tui

import (
	"strings"
	"testing"

	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelViewShowsCommitScopeExposureAndGrouping(t *testing.T) {
	model := NewModel(ModelInput{
		Scope: gitpkg.CommitScope{
			Files: []gitpkg.FileStatus{{Path: "README.md"}, {Path: "src/app.go"}},
			AIExposure: gitpkg.AIExposureSummary{
				SelectedFileCount: 2,
				DiffBudget:        gitpkg.BudgetUsage{UsedBytes: 120, MaxBytes: 200000},
				ReadBudget:        gitpkg.BudgetUsage{UsedBytes: 32, MaxBytes: 80000},
				PreferenceSources: gitpkg.PreferenceExposure{Provider: "user_config", APIKey: "env"},
			},
		},
		CommitPlan: CommitPlanView{Kind: "file_split", Commits: []CommitView{{Message: "docs: update readme", Files: []string{"README.md"}}, {Message: "chore: update app", Files: []string{"src/app.go"}}}},
	})

	view := model.View()
	for _, expected := range []string{"cnm", "Interactive Commit", "Scope", "README.md", "Exposure", "provider user_config", "Plan", "docs: update readme", "enter continue"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected view to contain %q:\n%s", expected, view)
		}
	}
}

func TestModelViewUsesFocusedTUIVisualStructure(t *testing.T) {
	model := NewModel(ModelInput{
		Width: 72,
		Scope: gitpkg.CommitScope{
			Files: []gitpkg.FileStatus{{Path: "README.md"}, {Path: "src/app.go"}},
			AIExposure: gitpkg.AIExposureSummary{
				SelectedFileCount: 2,
				DiffBudget:        gitpkg.BudgetUsage{UsedBytes: 120, MaxBytes: 200000},
				ReadBudget:        gitpkg.BudgetUsage{UsedBytes: 32, MaxBytes: 80000},
				PreferenceSources: gitpkg.PreferenceExposure{Provider: "user_config", APIKey: "env"},
			},
		},
		CommitPlan: CommitPlanView{Kind: "file_split", Commits: []CommitView{{Message: "docs: update readme", Files: []string{"README.md"}}, {Message: "chore: update app", Files: []string{"src/app.go"}}}},
	})

	view := model.View()
	for _, expected := range []string{"cnm", "Interactive Commit", "Scope", "Agent Instruction", "Exposure", "Plan", "[1/4]", "2 files", "file_split"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected focused TUI view to contain %q:\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(view, "\n"), "\n") {
		if len(stripANSIForTUITest(line)) > 92 {
			t.Fatalf("focused TUI line should stay readable, length=%d line=%q\n%s", len(stripANSIForTUITest(line)), line, view)
		}
	}
}

func TestModelTransitionsThroughCoreScreens(t *testing.T) {
	model := NewModel(ModelInput{Scope: gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "README.md"}}}})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	if model.Screen != ScreenAgentInstruction {
		t.Fatalf("expected agent instruction screen, got %s", model.Screen)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("split docs separately")})
	model = next.(Model)
	if !strings.Contains(model.AgentInstruction, "split docs") {
		t.Fatalf("expected agent instruction text, got %q", model.AgentInstruction)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	if model.Screen != ScreenAIActivity {
		t.Fatalf("expected ai activity screen, got %s", model.Screen)
	}

	next, _ = model.Update(AIActivityDoneMsg{Plan: CommitPlanView{Kind: "single", Commits: []CommitView{{Message: "chore: update readme", Files: []string{"README.md"}}}}})
	model = next.(Model)
	if model.Screen != ScreenMessageReview || len(model.CommitPlan.Commits) != 1 {
		t.Fatalf("expected message review with plan, got %+v", model)
	}
}

func TestModelLetsDeveloperAdjustCommitScopeBeforePlanning(t *testing.T) {
	model := NewModel(ModelInput{
		Scope: gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "README.md"}, {Path: "scratch.txt"}}},
		CommitPlan: CommitPlanView{Kind: "file_split", Commits: []CommitView{
			{Message: "docs: update readme", Files: []string{"README.md"}},
			{Message: "chore: update scratch", Files: []string{"scratch.txt"}},
		}},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = next.(Model)
	view := model.View()
	if !strings.Contains(view, "[ ] untracked") && !strings.Contains(view, "[ ] modified") {
		t.Fatalf("expected scope view to show excluded file:\n%s", view)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	if model.Screen != ScreenAgentInstruction {
		t.Fatalf("expected agent instruction screen, got %s", model.Screen)
	}
	if len(model.CommitPlan.Commits) != 1 || model.CommitPlan.Commits[0].Files[0] != "README.md" {
		t.Fatalf("expected commit plan to exclude scratch.txt, got %+v", model.CommitPlan)
	}
	result := model.Result()
	if len(result.ScopeFiles) != 1 || result.ScopeFiles[0] != "README.md" {
		t.Fatalf("expected selected scope files in result, got %+v", result.ScopeFiles)
	}
}

func TestModelLetsDeveloperEditCommitMessageBeforeAccepting(t *testing.T) {
	model := NewModel(ModelInput{
		Scope:      gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "README.md"}}},
		CommitPlan: CommitPlanView{Kind: "single", Commits: []CommitView{{Message: "chore: update README.md", Files: []string{"README.md"}}}},
	})
	model.Screen = ScreenMessageReview

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	model = next.(Model)
	if model.Screen != ScreenMessageEdit {
		t.Fatalf("expected message edit screen, got %s", model.Screen)
	}

	for range []rune("README.md") {
		next, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		model = next.(Model)
	}
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("readme")})
	model = next.(Model)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	if model.Screen != ScreenMessageReview || model.CommitPlan.Commits[0].Message != "chore: update readme" {
		t.Fatalf("expected edited message in review, got %+v", model)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	result := model.Result()
	if !result.Accepted || result.CommitPlan.Commits[0].Message != "chore: update readme" {
		t.Fatalf("expected accepted edited message result, got %+v", result)
	}
}

func TestModelSupportsNarrowNoColorFallback(t *testing.T) {
	model := NewModel(ModelInput{Width: 42, NoColor: true, Scope: gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "very/long/path/that/needs/to/remain/readable.go"}}}})
	view := model.View()
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("no-color view should not contain ANSI escapes: %q", view)
	}
	if !strings.Contains(view, "very/long/path") || !strings.Contains(view, "q quit") || !strings.Contains(view, "Scope") {
		t.Fatalf("unexpected narrow view:\n%s", view)
	}
	for _, line := range strings.Split(strings.TrimRight(view, "\n"), "\n") {
		if len(line) > 84 {
			t.Fatalf("narrow no-color line should not sprawl, length=%d line=%q\n%s", len(line), line, view)
		}
	}
}

func TestModelUpdatesWidthFromTerminalWindowSize(t *testing.T) {
	model := NewModel(ModelInput{
		Width: 88,
		Scope: gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "very/long/path/that/should/be-clipped-when-the-terminal-is-narrow.go"}}},
	})

	next, _ := model.Update(tea.WindowSizeMsg{Width: 44, Height: 20})
	model = next.(Model)
	if model.Width != 44 {
		t.Fatalf("expected model width from terminal, got %d", model.Width)
	}
	for _, line := range strings.Split(strings.TrimRight(model.View(), "\n"), "\n") {
		if len(stripANSIForTUITest(line)) > 44 {
			t.Fatalf("line should fit updated terminal width, length=%d line=%q\n%s", len(stripANSIForTUITest(line)), line, model.View())
		}
	}
}

func TestRunAppliesRuntimeNoColorToModel(t *testing.T) {
	var output strings.Builder
	result, err := Run(ModelInput{
		Scope: gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "README.md"}}},
	}, Runtime{Input: strings.NewReader("q"), Output: &output, NoColor: true})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Cancelled {
		t.Fatalf("expected scripted q to cancel TUI, got %+v", result)
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("runtime no-color output should not contain ANSI escapes: %q", output.String())
	}
}

func TestModelShowsInteractiveRepairContext(t *testing.T) {
	model := NewModel(ModelInput{
		RepairContext: &RepairContext{Reason: "Conflicts are present", EligibleFiles: []string{"conflict.txt"}},
		Scope:         gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "conflict.txt"}}},
	})
	if model.Screen != ScreenRepairReview {
		t.Fatalf("expected repair review screen, got %s", model.Screen)
	}
	view := model.View()
	for _, expected := range []string{"Interactive Repair", "Conflicts are present", "conflict.txt", "Repair Review"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected repair view to contain %q:\n%s", expected, view)
		}
	}
}

func stripANSIForTUITest(value string) string {
	var builder strings.Builder
	inEscape := false
	for index := 0; index < len(value); index++ {
		char := value[index]
		if inEscape {
			if char >= '@' && char <= '~' {
				inEscape = false
			}
			continue
		}
		if char == 0x1b {
			inEscape = true
			continue
		}
		builder.WriteByte(char)
	}
	return builder.String()
}

func TestModelAcceptsInteractiveRepairFromRepairReview(t *testing.T) {
	model := NewModel(ModelInput{
		RepairContext: &RepairContext{Reason: "Conflicts are present", EligibleFiles: []string{"conflict.txt"}},
		Scope:         gitpkg.CommitScope{Files: []gitpkg.FileStatus{{Path: "conflict.txt"}}},
	})

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(Model)
	result := model.Result()
	if !result.Accepted || result.Cancelled {
		t.Fatalf("expected repair review enter to accept Interactive Repair, got %+v", result)
	}
}
