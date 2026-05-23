package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ByteTrue/commit-now-myfriend/internal/commands"
	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/doctor"
	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
	"github.com/ByteTrue/commit-now-myfriend/internal/interactive"
	"github.com/ByteTrue/commit-now-myfriend/internal/output"
	"github.com/ByteTrue/commit-now-myfriend/internal/providers"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
	"github.com/ByteTrue/commit-now-myfriend/internal/tui"
	"golang.org/x/term"
)

var version = "dev"

type Runtime struct {
	CWD                string
	Env                map[string]string
	Stdin              io.Reader
	Stdout             io.Writer
	Stderr             io.Writer
	IsTTY              bool
	SecretStore        config.WritableSecretStore
	TUIRunner          func(tui.ModelInput, tui.Runtime) (tui.Result, error)
	ConfigPanelRunner  func(tui.ConfigPanelInput, tui.Runtime) (tui.ConfigPanelResult, error)
	OnboardingRunner   func(tui.OnboardingInput, tui.Runtime) (tui.OnboardingResult, error)
	CommitProvider     runtimex.ToolCallProvider
	RepairProvider     runtimex.ToolCallProvider
	ProviderHTTPClient providers.HTTPDoer
}

func Execute(args []string, stdout, stderr io.Writer) int {
	return ExecuteWithRuntime(args, Runtime{
		Stdin:             os.Stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		IsTTY:             isTTYDefault(),
		SecretStore:       config.NewSystemSecretStore(),
		TUIRunner:         tui.Run,
		ConfigPanelRunner: tui.RunConfigPanel,
		OnboardingRunner:  tui.RunOnboarding,
	})
}

func ExecuteWithRuntime(args []string, runtime Runtime) int {
	runtime = normalizeRuntime(runtime)
	exitCode := executeWithRuntime(args, runtime)
	writeDebugCommandFinished(runtime, args, exitCode)
	return exitCode
}

func executeWithRuntime(args []string, runtime Runtime) int {
	if len(args) == 0 {
		return runRoot(nil, runtime)
	}

	for _, arg := range args {
		if arg == "--json" {
			continue
		}
		if arg == "--help" || arg == "-h" {
			if hasJSONFlag(args) {
				fmt.Fprintln(runtime.Stderr, "error: --json cannot be combined with --help.")
				return int(output.Error)
			}
			printUsage(runtime.Stdout)
			return int(output.Success)
		}
		if arg == "--version" || arg == "-version" {
			fmt.Fprintln(runtime.Stdout, version)
			return int(output.Success)
		}
		break
	}

	command, commandIndex := resolveCommand(args)
	switch command {
	case "":
		return runRoot(args, runtime)
	case "auto":
		return runAuto(args[commandIndex+1:], runtime)
	case "init":
		return commands.RunInit(args[commandIndex+1:], commands.InitRuntime{
			CWD:              runtime.CWD,
			Env:              runtime.Env,
			Stdin:            runtime.Stdin,
			Stdout:           runtime.Stdout,
			Stderr:           runtime.Stderr,
			IsTTY:            runtime.IsTTY,
			SecretStore:      runtime.SecretStore,
			OnboardingRunner: runtimeOnboardingRunner(runtime),
		})
	case "config":
		return runConfigCommand(args[commandIndex+1:], runtime)
	case "doctor":
		return commands.RunDoctor(args[commandIndex+1:], commands.DoctorRuntime{
			CWD:         runtime.CWD,
			Env:         runtime.Env,
			Stdout:      runtime.Stdout,
			Stderr:      runtime.Stderr,
			IsTTY:       runtime.IsTTY,
			SecretStore: runtime.SecretStore,
			RenderRich:  doctorRichRenderer(),
		})
	case "split", "repair", "check", "onboard":
		fmt.Fprintf(runtime.Stderr, "error: removed command '%s'\n", command)
		fmt.Fprintln(runtime.Stderr, "Use `cnm` for Interactive Commit or `cnm auto` for Autonomous Commit.")
		return int(output.Error)
	case "help":
		printUsage(runtime.Stdout)
		return int(output.Success)
	default:
		fmt.Fprintf(runtime.Stderr, "error: unknown command '%s'\n", command)
		fmt.Fprintln(runtime.Stderr, "Run cnm --help to inspect available commands.")
		return int(output.Error)
	}
}

type debugCommandFinishedEvent struct {
	SchemaVersion int    `json:"schemaVersion"`
	Event         string `json:"event"`
	Command       string `json:"command"`
	ExitCode      int    `json:"exitCode"`
	Timestamp     string `json:"timestamp"`
}

func writeDebugCommandFinished(runtime Runtime, args []string, exitCode int) {
	path := strings.TrimSpace(runtimeEnvValue(runtime.Env, "CNM_DEBUG_LOG"))
	if path == "" {
		return
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(runtime.CWD, path)
	}
	if dir := filepath.Dir(path); dir != "." && strings.TrimSpace(dir) != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(debugCommandFinishedEvent{
		SchemaVersion: 1,
		Event:         "command_finished",
		Command:       debugCommandName(args),
		ExitCode:      exitCode,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	})
}

func debugCommandName(args []string) string {
	if len(args) == 0 {
		return "cnm"
	}
	for _, arg := range args {
		if arg == "--json" {
			continue
		}
		if arg == "--help" || arg == "-h" {
			return "cnm help"
		}
		if arg == "--version" || arg == "-version" {
			return "cnm version"
		}
		break
	}
	command, _ := resolveCommand(args)
	switch command {
	case "":
		return "cnm"
	case "auto":
		return "cnm auto"
	case "init":
		return "cnm init"
	case "config":
		return "cnm config"
	case "doctor":
		return "cnm doctor"
	case "help":
		return "cnm help"
	case "split", "repair", "check", "onboard":
		return "removed"
	default:
		return "unknown"
	}
}

func runtimeEnvValue(env map[string]string, key string) string {
	if env != nil {
		if value, ok := env[key]; ok {
			return value
		}
	}
	return os.Getenv(key)
}

type autoArgs struct {
	DryRun      bool
	JSON        bool
	TUI         bool
	Staged      bool
	NoUntracked bool
	DiffOnly    bool
	NoVerify    bool
	Verbose     bool
	Instruction string
	Pathspecs   []string
}

type autoCommitPlan struct {
	Kind             string                `json:"kind"`
	Commits          []autoPlannedCommit   `json:"commits"`
	SplitLimitations []autoSplitLimitation `json:"splitLimitations,omitempty"`
}

type autoPlannedCommit struct {
	Message string              `json:"message"`
	Files   []gitpkg.FileStatus `json:"files"`
}

type autoSplitLimitation struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Fallback string `json:"fallback"`
}

type autoCommitExecution struct {
	Results      []gitpkg.CommitScopeCommitResult
	Plan         autoCommitPlan
	Transaction  gitpkg.CommitTransactionRollbackResult
	RetryAttempt bool
	RetryCount   int
}

func runConfigCommand(args []string, runtime Runtime) int {
	if len(args) == 0 && runtime.IsTTY {
		return runConfigPanel(runtime)
	}
	return commands.RunConfig(args, commands.ConfigRuntime{
		CWD:         runtime.CWD,
		Env:         runtime.Env,
		Stdout:      runtime.Stdout,
		Stderr:      runtime.Stderr,
		SecretStore: runtime.SecretStore,
	})
}

func doctorRichRenderer() func(doctor.Report) string {
	return func(report doctor.Report) string {
		view := buildDoctorView(report)
		width := terminalWidth()
		return tui.RenderDoctorRich(view, width, false)
	}
}

func terminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 96
}

func buildDoctorView(report doctor.Report) tui.DoctorView {
	view := tui.DoctorView{
		Status:            report.Status,
		Errors:            report.Summary.Errors,
		Warnings:          report.Summary.Warnings,
		OK:                report.OK,
		GitMessage:        report.Checks.Git.Message,
		GitStatus:         string(report.Checks.Git.Status),
		RepoMessage:       report.Checks.Repository.Message,
		RepoStatus:        string(report.Checks.Repository.Status),
		ConfigMessage:     report.Checks.EffectiveConfig.Message,
		ConfigStatus:      string(report.Checks.EffectiveConfig.Status),
		CapabilityMessage: report.Checks.ProviderCapability.Message,
		CapabilityStatus:  string(report.Checks.ProviderCapability.Status),
		ProviderName:      report.Checks.EffectiveConfig.Config.Provider,
		ModelName:         report.Checks.EffectiveConfig.Config.Model,
		APIKeySource:      report.Checks.EffectiveConfig.Sources.APIKey,
		UserConfig:        report.Paths.UserConfigPath,
		ProjectConfig:     report.Paths.ProjectConfigPath,
	}
	if report.Probe != nil {
		view.ProbeAttempted = true
		view.ProbeStatus = report.Probe.Status
		if report.Probe.Message != "" {
			view.ProbeMessage = report.Probe.Message
		} else if report.Probe.Error != "" {
			view.ProbeMessage = report.Probe.Error
		}
	}
	for _, issue := range report.Issues {
		view.Issues = append(view.Issues, tui.DoctorIssue{
			Severity: string(issue.Severity),
			Check:    issue.Check,
			Message:  issue.Message,
		})
	}
	return view
}

func runtimeOnboardingRunner(runtime Runtime) commands.OnboardingRunnerFunc {
	if !runtime.IsTTY {
		return nil
	}
	if runtime.OnboardingRunner == nil {
		return nil
	}
	return func(prefill commands.OnboardingPrefill) (commands.OnboardingAnswers, error) {
		input := tui.OnboardingInput{
			CurrentProvider: prefill.Provider,
			CurrentModel:    prefill.Model,
			CurrentBaseURL:  prefill.BaseURL,
			CurrentStyle:    prefill.PromptStyle,
			CurrentLanguage: prefill.MessageLanguage,
			CurrentStanding: prefill.StandingInstruction,
		}
		result, err := runtime.OnboardingRunner(input, tui.Runtime{Input: runtime.Stdin, Output: runtime.Stdout})
		if err != nil {
			return commands.OnboardingAnswers{}, err
		}
		return commands.OnboardingAnswers{
			Cancelled:           result.Cancelled,
			Provider:            result.Provider,
			Model:               result.Model,
			BaseURL:             result.BaseURL,
			PromptStyle:         result.PromptStyle,
			MessageLanguage:     result.MessageLanguage,
			StandingInstruction: result.StandingInstruction,
			APIKey:              result.APIKey,
		}, nil
	}
}

func runConfigPanel(runtime Runtime) int {
	resolved, err := config.ResolveEffectiveConfig(config.ResolveConfigOptions{
		CWD:         runtime.CWD,
		Env:         runtime.Env,
		SecretStore: runtime.SecretStore,
	})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range resolved.Warnings {
		fmt.Fprintln(runtime.Stderr, "Warning: "+warning)
	}
	paths := config.GetConfigPaths(config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	sources := config.SummarizeConfigSources(resolved)
	input := tui.ConfigPanelInput{
		Effective: resolved.Values,
		Sources:   sources,
		UserPath:  paths.UserConfigPath,
		WriteValue: func(key config.ConfigKey, value string) error {
			patch, err := config.ParseKeyValue(string(key), value)
			if err != nil {
				return err
			}
			if key == config.ConfigKeyAPIKey && runtime.SecretStore != nil {
				provider := resolved.Values.Provider
				if patch.APIKey != nil {
					if err := runtime.SecretStore.SetAPIKey(provider, *patch.APIKey); err != nil {
						return err
					}
				}
				return nil
			}
			_, err = config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
			return err
		},
		UnsetValue: func(key config.ConfigKey) error {
			_, err := config.UnsetUserConfigKey(key, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
			return err
		},
		Reload: func() (config.EffectiveConfig, config.ConfigSourceSummary, error) {
			next, err := config.ResolveEffectiveConfig(config.ResolveConfigOptions{
				CWD:         runtime.CWD,
				Env:         runtime.Env,
				SecretStore: runtime.SecretStore,
			})
			if err != nil {
				return config.EffectiveConfig{}, config.ConfigSourceSummary{}, err
			}
			return next.Values, config.SummarizeConfigSources(next), nil
		},
	}
	runner := runtime.ConfigPanelRunner
	if runner == nil {
		runner = tui.RunConfigPanel
	}
	_, err = runner(input, tui.Runtime{Input: runtime.Stdin, Output: runtime.Stdout})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	return int(output.Success)
}

func runAuto(args []string, runtime Runtime) int {
	parsed, err := parseAutoArgs(args)
	if err != nil {
		if errors.Is(err, errAutoHelp) {
			printAutoUsage(runtime.Stdout)
			return int(output.Success)
		}
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.UsageError)
	}

	resolvedConfig, configErr := config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: runtime.CWD, Env: runtime.Env, SecretStore: runtime.SecretStore})
	preferenceSources := gitpkg.PreferenceExposure{}
	if configErr == nil {
		sourceSummary := config.SummarizeConfigSources(resolvedConfig)
		preferenceSources = gitpkg.PreferenceExposure{
			Provider:            string(sourceSummary.Provider),
			Model:               string(sourceSummary.Model),
			APIKey:              string(sourceSummary.APIKey),
			PromptStyle:         string(sourceSummary.PromptStyle),
			MessageLanguage:     string(sourceSummary.MessageLanguage),
			StandingInstruction: sourceSummary.StandingInstruction,
		}
	}

	scope, scopeErr := gitpkg.InspectCommitScope(gitpkg.CommitScopeOptions{
		CWD:               runtime.CWD,
		Env:               runtime.Env,
		StagedOnly:        parsed.Staged,
		IncludeUntracked:  !parsed.NoUntracked,
		DiffOnly:          parsed.DiffOnly,
		PreferenceSources: preferenceSources,
		Pathspecs:         parsed.Pathspecs,
	})

	if scopeErr != nil {
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			_ = router.WriteJSON(map[string]any{
				"schemaVersion": 1,
				"command":       "cnm auto",
				"ok":            false,
				"status":        "scope_error",
				"scopeError":    scopeErr.Error(),
			})
		} else {
			fmt.Fprintln(runtime.Stderr, scopeErr.Error())
		}
		return int(output.Error)
	}
	if !scope.HasSelectedChanges {
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			_ = router.WriteJSON(map[string]any{
				"schemaVersion": 1,
				"command":       "cnm auto",
				"ok":            true,
				"status":        "no_changes",
				"scope":         scope,
				"contextPolicy": scope.ContextPolicy,
				"aiExposure":    scope.AIExposure,
			})
		} else {
			fmt.Fprintln(runtime.Stdout, "No selected changes.")
		}
		return int(output.Success)
	}
	if scopeHasConflict(scope) {
		message := "Conflicts are present; cnm auto cannot repair or commit conflicts non-interactively."
		if parsed.TUI {
			return runConflictTUIHandoff(runtime, scope, parsed, resolvedConfig.Values, message)
		}
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			_ = router.WriteJSON(map[string]any{
				"schemaVersion": 1,
				"command":       "cnm auto",
				"ok":            false,
				"status":        "conflict",
				"error":         message,
				"scope":         scope,
				"contextPolicy": scope.ContextPolicy,
				"aiExposure":    scope.AIExposure,
			})
		} else {
			fmt.Fprintln(runtime.Stderr, message)
		}
		return int(output.Error)
	}
	if len(scope.SecretBlockers) > 0 {
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			_ = router.WriteJSON(map[string]any{
				"schemaVersion":  1,
				"command":        "cnm auto",
				"ok":             false,
				"status":         "secret_blocked",
				"scope":          scope,
				"contextPolicy":  scope.ContextPolicy,
				"aiExposure":     scope.AIExposure,
				"secretBlockers": scope.SecretBlockers,
			})
		} else {
			fmt.Fprintln(runtime.Stderr, "Secret Blocker: selected changes contain potential secrets.")
		}
		return int(output.Error)
	}
	if configErr != nil {
		return writeAutoConfigMissing(runtime, parsed, scope, autoConfigIssue{Code: "config_error", Message: configErr.Error()})
	}
	if issue := autoRequiredConfigIssue(resolvedConfig.Values); issue != nil {
		return writeAutoConfigMissing(runtime, parsed, scope, *issue)
	}
	if parsed.DryRun {
		execution, err := executeAutoCommit(runtime, scope, parsed, resolvedConfig.Values)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		commitPlan := execution.Plan
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			_ = router.WriteJSON(map[string]any{
				"schemaVersion": 1,
				"command":       "cnm auto",
				"ok":            true,
				"status":        "plan_preview",
				"dryRun":        true,
				"scope":         scope,
				"contextPolicy": scope.ContextPolicy,
				"aiExposure":    scope.AIExposure,
				"commitPlan":    commitPlan,
			})
		} else {
			fmt.Fprintf(runtime.Stdout, "Plan: %s\n", commitPlan.Commits[0].Message)
		}
		return int(output.Success)
	}

	if !parsed.DryRun && scope.HasSelectedChanges && len(scope.SecretBlockers) == 0 {
		execution, err := executeAutoCommit(runtime, scope, parsed, resolvedConfig.Values)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		commitPlan := execution.Plan
		commitResult := execution.lastResult()
		if commitResult.Git.ExitCode != 0 {
			if parsed.JSON {
				router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
				_ = router.WriteJSON(map[string]any{
					"schemaVersion": 1,
					"command":       "cnm auto",
					"ok":            false,
					"status":        "commit_failed",
					"error":         firstNonEmptyString(commitResult.Git.Stderr, commitResult.Git.Stdout, "git commit failed"),
					"scope":         scope,
					"contextPolicy": scope.ContextPolicy,
					"aiExposure":    scope.AIExposure,
					"commitPlan":    commitPlan,
					"transaction":   execution.Transaction,
				})
			} else {
				fmt.Fprintln(runtime.Stderr, firstNonEmptyString(commitResult.Git.Stderr, commitResult.Git.Stdout, "git commit failed"))
			}
			return int(output.Error)
		}
		if parsed.JSON {
			router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
			commits := execution.commitSummaries()
			_ = router.WriteJSON(map[string]any{
				"schemaVersion": 1,
				"command":       "cnm auto",
				"ok":            true,
				"status":        "committed",
				"dryRun":        false,
				"scope":         scope,
				"contextPolicy": scope.ContextPolicy,
				"aiExposure":    scope.AIExposure,
				"commitPlan":    commitPlan,
				"commit": map[string]any{
					"hash":    commitResult.Hash,
					"message": commitResult.Message,
				},
				"commits": commits,
				"messageRetry": map[string]any{
					"attempted": execution.RetryAttempt,
					"count":     execution.RetryCount,
				},
			})
		} else {
			fmt.Fprintf(runtime.Stdout, "Committed %s %s\n", commitResult.Hash, commitResult.Message)
		}
		return int(output.Success)
	}

	fmt.Fprintln(runtime.Stderr, "error: invalid Autonomous Commit state")
	return int(output.Error)
}

type autoConfigIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func autoRequiredConfigIssue(values config.EffectiveConfig) *autoConfigIssue {
	if values.APIKey == nil || strings.TrimSpace(*values.APIKey) == "" {
		return &autoConfigIssue{Code: "api_key_missing", Message: "No API key is configured. Run `cnm init` to save one in Secret Store, or set CNM_API_KEY for this process."}
	}
	if strings.TrimSpace(values.Model) == "" {
		return &autoConfigIssue{Code: "model_missing", Message: "No model is configured. Run `cnm init` or `cnm config set model <model>` before using cnm auto."}
	}
	if values.Provider == config.ProviderOpenAICompatible && (values.BaseURL == nil || strings.TrimSpace(*values.BaseURL) == "") {
		return &autoConfigIssue{Code: "base_url_missing", Message: "The openai-compatible provider requires baseURL. Run `cnm init` or `cnm config set baseURL <url>` before using cnm auto."}
	}
	return nil
}

func writeAutoConfigMissing(runtime Runtime, parsed autoArgs, scope gitpkg.CommitScope, issue autoConfigIssue) int {
	if parsed.JSON {
		router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
		_ = router.WriteJSON(map[string]any{
			"schemaVersion": 1,
			"command":       "cnm auto",
			"ok":            false,
			"status":        "config_missing",
			"dryRun":        parsed.DryRun,
			"error":         issue.Message,
			"configIssue":   issue,
			"scope":         scope,
			"contextPolicy": scope.ContextPolicy,
			"aiExposure":    scope.AIExposure,
		})
	} else {
		fmt.Fprintln(runtime.Stderr, issue.Message)
	}
	return int(output.Error)
}

func runConflictTUIHandoff(runtime Runtime, scope gitpkg.CommitScope, parsed autoArgs, resolvedConfig config.EffectiveConfig, message string) int {
	tuiRunner := runtime.TUIRunner
	if tuiRunner == nil {
		tuiRunner = tui.Run
	}
	result, err := tuiRunner(tui.ModelInput{
		RepairContext: &tui.RepairContext{Reason: message, EligibleFiles: conflictFilePaths(scope)},
		Scope:         scope,
		FileReader:    repairFileReader(runtime),
	}, tui.Runtime{Input: runtime.Stdin, Output: runtime.Stdout})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	if !result.Accepted || result.Cancelled {
		fmt.Fprintln(runtime.Stderr, "Cancelled. No commit was created.")
		return int(output.UserCancel)
	}
	if err := executeInteractiveRepair(runtime, scope, resolvedConfig, message); err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	executionScope := filterScopeByPaths(scope, result.ScopeFiles)
	commitPlan, err := autoCommitPlanFromTUIPlan(result.CommitPlan, executionScope)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	execution, err := executeCommitPlan(runtime, executionScope, autoArgs{NoVerify: parsed.NoVerify}, commitPlan)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	commitResult := execution.lastResult()
	if commitResult.Git.ExitCode != 0 {
		fmt.Fprintln(runtime.Stderr, firstNonEmptyString(commitResult.Git.Stderr, commitResult.Git.Stdout, "git commit failed"))
		return int(output.Error)
	}
	fmt.Fprintf(runtime.Stdout, "Committed %s %s\n", commitResult.Hash, commitResult.Message)
	return int(output.Success)
}

func conflictFilePaths(scope gitpkg.CommitScope) []string {
	paths := make([]string, 0)
	for _, file := range scope.Files {
		if scopeFileHasConflict(file) {
			paths = append(paths, file.Path)
		}
	}
	return paths
}

func executeInteractiveRepair(runtime Runtime, scope gitpkg.CommitScope, resolvedConfig config.EffectiveConfig, reason string) error {
	provider, err := resolveRepairProvider(runtime, scope, resolvedConfig, reason)
	if err != nil {
		return err
	}
	eligiblePaths := conflictFilePaths(scope)
	if len(eligiblePaths) == 0 {
		return fmt.Errorf("Interactive Repair requires at least one eligible conflicted file")
	}
	repairRuntime := runtimex.NewToolCallRuntime(runtimex.ToolCallRuntimeOptions{
		Provider:      provider,
		ContextPolicy: scope.ContextPolicy,
		Tools:         interactiveRepairDomainTools(runtime, scope, eligiblePaths),
		RepairPolicy: runtimex.RepairPolicy{
			AllowedPaths: eligiblePaths,
			ConfirmWrite: func(input runtimex.RepairFileInput) (bool, error) {
				return true, nil
			},
		},
	})
	result, err := repairRuntime.Run()
	if err != nil {
		return err
	}
	if result.Status != runtimex.RunStatusCompleted {
		return fmt.Errorf("Interactive Repair did not complete: %s", firstNonEmptyString(result.Message, result.LimitReason, string(result.Status)))
	}
	return nil
}

func resolveRepairProvider(runtime Runtime, scope gitpkg.CommitScope, resolvedConfig config.EffectiveConfig, reason string) (runtimex.ToolCallProvider, error) {
	if runtime.RepairProvider != nil {
		return runtime.RepairProvider, nil
	}
	providerConfig := providers.ProviderConfig{
		Provider:   resolvedConfig.Provider,
		APIKey:     resolvedConfig.APIKey,
		BaseURL:    resolvedConfig.BaseURL,
		Model:      resolvedConfig.Model,
		HTTPClient: runtime.ProviderHTTPClient,
	}
	return providers.CreateToolCallProvider(providers.ToolCallProviderOptions{
		Config:       providerConfig,
		Instructions: interactiveRepairInstructions(resolvedConfig),
		Input:        interactiveRepairInput(scope, reason),
		Tools: []runtimex.ToolName{
			runtimex.ToolInspectCommitScope,
			runtimex.ToolGetDiff,
			runtimex.ToolReadFile,
			runtimex.ToolPreviewCommit,
			runtimex.ToolRepairFile,
			runtimex.ToolFinish,
			runtimex.ToolAbort,
		},
	})
}

func interactiveRepairInstructions(resolvedConfig config.EffectiveConfig) string {
	parts := []string{
		"You are running cnm Interactive Repair inside the Full-screen TUI.",
		"Use only the provided Domain Tools. Do not request shell access or low-level git commands.",
		"Read each file with read_file before repair_file. Repair only eligible conflicted files.",
		"After applying necessary repairs, call finish. If the conflict cannot be resolved safely, call abort with a clear message.",
	}
	if resolvedConfig.StandingInstruction != nil && strings.TrimSpace(*resolvedConfig.StandingInstruction) != "" {
		parts = append(parts, "Standing Instruction: "+strings.TrimSpace(*resolvedConfig.StandingInstruction))
	}
	if strings.TrimSpace(string(resolvedConfig.MessageLanguage)) != "" {
		parts = append(parts, "Message Language: "+string(resolvedConfig.MessageLanguage))
	}
	return strings.Join(parts, "\n")
}

func interactiveRepairInput(scope gitpkg.CommitScope, reason string) string {
	lines := []string{"Interactive Repair request.", "Reason: " + reason, "Eligible conflicted files:"}
	for _, path := range conflictFilePaths(scope) {
		lines = append(lines, "- "+path)
	}
	if len(scope.Pathspecs) > 0 {
		lines = append(lines, "Commit Scope pathspecs:")
		for _, pathspec := range scope.Pathspecs {
			lines = append(lines, "- "+pathspec)
		}
	}
	return strings.Join(lines, "\n")
}

func interactiveRepairDomainTools(runtime Runtime, scope gitpkg.CommitScope, eligiblePaths []string) runtimex.DomainToolSet {
	allowed := allowedPathSet(eligiblePaths)
	return runtimex.NewDomainToolSet(runtimex.DomainToolSetOptions{
		InspectCommitScope: func() (gitpkg.CommitScope, error) {
			return gitpkg.InspectCommitScope(gitpkg.CommitScopeOptions{CWD: runtime.CWD, Env: runtime.Env, IncludeUntracked: scope.IncludesUntracked, StagedOnly: scope.StagedOnly, DiffOnly: scope.ContextPolicy.Mode == gitpkg.ContextPolicyModeDiffOnly, Pathspecs: scope.Pathspecs})
		},
		GetDiff: func() (runtimex.DiffResult, error) {
			result, err := gitpkg.DefaultCommandRunner(runtime.CWD, scopedDiffArgs(eligiblePaths), runtime.Env)
			if err != nil {
				return runtimex.DiffResult{}, err
			}
			if result.ExitCode != 0 {
				return runtimex.DiffResult{}, fmt.Errorf("%s", firstNonEmptyString(result.Stderr, result.Stdout, "git diff failed"))
			}
			return runtimex.DiffResult{Content: result.Stdout, Bytes: len([]byte(result.Stdout))}, nil
		},
		ReadFile: func(path string) (runtimex.FileReadResult, error) {
			relativePath, err := requireAllowedRepairPath(path, allowed)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			absolutePath, err := repoFilePath(runtime.CWD, relativePath)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			content, err := os.ReadFile(absolutePath)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			return runtimex.FileReadResult{Path: relativePath, Content: string(content), Bytes: len(content)}, nil
		},
		PreviewCommit: func(input runtimex.CommitPreviewInput) (runtimex.CommitPreviewResult, error) {
			return runtimex.CommitPreviewResult{Message: input.Message, FileCount: len(scope.Files)}, nil
		},
		RepairFile: func(input runtimex.RepairFileInput) (runtimex.RepairFileResult, error) {
			relativePath, err := requireAllowedRepairPath(input.Path, allowed)
			if err != nil {
				return runtimex.RepairFileResult{}, err
			}
			absolutePath, err := repoFilePath(runtime.CWD, relativePath)
			if err != nil {
				return runtimex.RepairFileResult{}, err
			}
			if err := os.WriteFile(absolutePath, []byte(input.Content), 0o644); err != nil {
				return runtimex.RepairFileResult{}, err
			}
			return runtimex.RepairFileResult{Path: relativePath, Applied: true}, nil
		},
	})
}

func scopedDiffArgs(paths []string) []string {
	args := []string{"diff", "--"}
	args = append(args, paths...)
	return args
}

func allowedPathSet(paths []string) map[string]bool {
	allowed := map[string]bool{}
	for _, path := range paths {
		if normalized, err := normalizeRepoRelativePath(path); err == nil {
			allowed[normalized] = true
		}
	}
	return allowed
}

func requireAllowedRepairPath(rawPath string, allowed map[string]bool) (string, error) {
	path, err := normalizeRepoRelativePath(rawPath)
	if err != nil {
		return "", err
	}
	if !allowed[path] {
		return "", fmt.Errorf("path is not eligible for Interactive Repair: %s", path)
	}
	return path, nil
}

func normalizeRepoRelativePath(rawPath string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(trimmed) || strings.HasPrefix(strings.ReplaceAll(trimmed, "\\", "/"), "/") {
		return "", fmt.Errorf("absolute paths are not allowed: %s", rawPath)
	}
	cleaned := pathpkg.Clean(strings.ReplaceAll(trimmed, "\\", "/"))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path must stay inside the repository: %s", rawPath)
	}
	return cleaned, nil
}

func repoFilePath(cwd string, relativePath string) (string, error) {
	base, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(base, filepath.FromSlash(relativePath))
	absolute, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, absolute)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must stay inside the repository: %s", relativePath)
	}
	return absolute, nil
}

func workingTreeDiffProvider(runtime Runtime, scope gitpkg.CommitScope) tui.DiffProvider {
	pathSet := map[string]bool{}
	for _, file := range scope.Files {
		pathSet[file.Path] = true
	}
	return func(path string) (string, error) {
		if !pathSet[path] {
			return "", fmt.Errorf("path is outside Commit Scope: %s", path)
		}
		isUntracked := false
		for _, file := range scope.Files {
			if file.Path == path {
				isUntracked = file.Untracked
				break
			}
		}
		if isUntracked {
			absolute, err := repoFilePath(runtime.CWD, path)
			if err != nil {
				return "", err
			}
			content, err := os.ReadFile(absolute)
			if err != nil {
				return "", err
			}
			return formatUntrackedDiff(path, string(content)), nil
		}
		args := []string{"diff", "HEAD", "--", path}
		result, err := gitpkg.DefaultCommandRunner(runtime.CWD, args, runtime.Env)
		if err != nil {
			return "", err
		}
		if result.ExitCode != 0 {
			return "", fmt.Errorf("%s", firstNonEmptyString(result.Stderr, result.Stdout, "git diff failed"))
		}
		if strings.TrimSpace(result.Stdout) == "" {
			args = []string{"diff", "--cached", "--", path}
			result, err = gitpkg.DefaultCommandRunner(runtime.CWD, args, runtime.Env)
			if err != nil {
				return "", err
			}
			if result.ExitCode != 0 {
				return "", fmt.Errorf("%s", firstNonEmptyString(result.Stderr, result.Stdout, "git diff failed"))
			}
		}
		return result.Stdout, nil
	}
}

func formatUntrackedDiff(path, content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n", path, path)
	b.WriteString("new file mode 100644\n")
	fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, line := range lines {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func repairFileReader(runtime Runtime) tui.FileReader {
	return func(path string) (string, error) {
		absolute, err := repoFilePath(runtime.CWD, path)
		if err != nil {
			return "", err
		}
		content, err := os.ReadFile(absolute)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
}

func executeAutoCommit(runtime Runtime, scope gitpkg.CommitScope, parsed autoArgs, resolvedConfig config.EffectiveConfig) (autoCommitExecution, error) {
	provider, err := resolveCommitProvider(runtime, scope, parsed, resolvedConfig)
	if err != nil {
		return autoCommitExecution{}, err
	}
	commitRuntime := runtimex.NewToolCallRuntime(runtimex.ToolCallRuntimeOptions{
		Provider:      provider,
		ContextPolicy: scope.ContextPolicy,
		Tools:         autonomousCommitDomainTools(runtime, scope, parsed),
		Limits:        runtimex.LoopLimits{MaxToolCalls: 48, MaxProviderRetries: 1, MaxDuration: 5 * time.Minute},
	})
	result, err := commitRuntime.Run()
	if err != nil {
		return autoCommitExecution{}, err
	}
	execution, ok := autoCommitExecutionFromToolCalls(result.Calls)
	if !ok {
		return autoCommitExecution{}, fmt.Errorf("Autonomous Commit did not create or preview commits: %s", firstNonEmptyString(result.Message, result.LimitReason, string(result.Status)))
	}
	if result.Status != runtimex.RunStatusCompleted {
		last := execution.lastResult()
		if last.Git.ExitCode != 0 {
			return execution, nil
		}
		return autoCommitExecution{}, fmt.Errorf("Autonomous Commit did not complete: %s", firstNonEmptyString(result.Message, result.LimitReason, string(result.Status)))
	}
	return execution, nil
}

func resolveCommitProvider(runtime Runtime, scope gitpkg.CommitScope, parsed autoArgs, resolvedConfig config.EffectiveConfig) (runtimex.ToolCallProvider, error) {
	if runtime.CommitProvider != nil {
		return runtime.CommitProvider, nil
	}
	providerConfig := providers.ProviderConfig{
		Provider:   resolvedConfig.Provider,
		APIKey:     resolvedConfig.APIKey,
		BaseURL:    resolvedConfig.BaseURL,
		Model:      resolvedConfig.Model,
		HTTPClient: runtime.ProviderHTTPClient,
	}
	return providers.CreateToolCallProvider(providers.ToolCallProviderOptions{
		Config:       providerConfig,
		Instructions: autonomousCommitInstructions(resolvedConfig, parsed),
		Input:        autonomousCommitInput(scope, parsed),
		Tools: []runtimex.ToolName{
			runtimex.ToolInspectCommitScope,
			runtimex.ToolGetDiff,
			runtimex.ToolReadFile,
			runtimex.ToolPreviewCommit,
			runtimex.ToolCreateCommits,
			runtimex.ToolFinish,
			runtimex.ToolAbort,
		},
	})
}

func autonomousCommitInstructions(resolvedConfig config.EffectiveConfig, parsed autoArgs) string {
	parts := []string{
		"You are running cnm Autonomous Commit.",
		"Use only the provided Domain Tools. Do not request shell access, raw git commands, or JSON plans.",
		"Start by calling inspect_commit_scope. Use get_diff and bounded read_file when needed before choosing commits.",
		"Create local commits by calling create_commits. You may create one or more file-level commits; every selected file you commit must belong to exactly one commit.",
		"Use conservative splitting: split only when files clearly represent independent intentions; keep related code, tests, and docs together.",
		"If a tool call is rejected, adjust and call the correct tool again. Finish only after create_commits reports success or a dry-run preview.",
		"Never push to a remote.",
	}
	if parsed.DryRun {
		parts = append(parts, "This is a Commit Plan Preview. create_commits will validate and preview the plan without creating commits.")
	}
	if resolvedConfig.StandingInstruction != nil && strings.TrimSpace(*resolvedConfig.StandingInstruction) != "" {
		parts = append(parts, "Standing Instruction: "+strings.TrimSpace(*resolvedConfig.StandingInstruction))
	}
	if strings.TrimSpace(parsed.Instruction) != "" {
		parts = append(parts, "Developer Instruction for this run: "+strings.TrimSpace(parsed.Instruction))
	}
	if strings.TrimSpace(string(resolvedConfig.PromptStyle)) != "" {
		parts = append(parts, "Commit Style: "+string(resolvedConfig.PromptStyle))
	}
	if strings.TrimSpace(string(resolvedConfig.MessageLanguage)) != "" {
		parts = append(parts, "Message Language: "+string(resolvedConfig.MessageLanguage))
	}
	if parsed.NoVerify {
		parts = append(parts, "The developer explicitly chose --no-verify for commit creation.")
	} else {
		parts = append(parts, "Respect Git hooks. If a commit message is rejected, create_commits can be called once more with a corrected message.")
	}
	return strings.Join(parts, "\n")
}

func autonomousCommitInput(scope gitpkg.CommitScope, parsed autoArgs) string {
	lines := []string{"Autonomous Commit request."}
	if parsed.DryRun {
		lines = append(lines, "Mode: Commit Plan Preview / dry run")
	} else {
		lines = append(lines, "Mode: create local commit(s), no push")
	}
	if scope.StagedOnly {
		lines = append(lines, "Commit Scope: staged changes only")
	} else {
		lines = append(lines, "Commit Scope: working tree changes")
	}
	if scope.IncludesUntracked {
		lines = append(lines, "Untracked Inclusion: enabled")
	} else {
		lines = append(lines, "Untracked Inclusion: disabled")
	}
	if len(scope.Pathspecs) > 0 {
		lines = append(lines, "Pathspecs:")
		for _, pathspec := range scope.Pathspecs {
			lines = append(lines, "- "+pathspec)
		}
	}
	lines = append(lines, "Selected files:")
	for _, file := range scope.Files {
		lines = append(lines, "- "+file.Path)
	}
	return strings.Join(lines, "\n")
}

func autonomousCommitDomainTools(runtime Runtime, scope gitpkg.CommitScope, parsed autoArgs) runtimex.DomainToolSet {
	return runtimex.NewDomainToolSet(runtimex.DomainToolSetOptions{
		InspectCommitScope: func() (gitpkg.CommitScope, error) {
			return gitpkg.InspectCommitScope(gitpkg.CommitScopeOptions{CWD: runtime.CWD, Env: runtime.Env, IncludeUntracked: scope.IncludesUntracked, StagedOnly: scope.StagedOnly, DiffOnly: scope.ContextPolicy.Mode == gitpkg.ContextPolicyModeDiffOnly, Pathspecs: scope.Pathspecs})
		},
		GetDiff: func() (runtimex.DiffResult, error) {
			result, err := gitpkg.DefaultCommandRunner(runtime.CWD, scopedDiffArgs(commitScopePaths(scope)), runtime.Env)
			if err != nil {
				return runtimex.DiffResult{}, err
			}
			if result.ExitCode != 0 {
				return runtimex.DiffResult{}, fmt.Errorf("%s", firstNonEmptyString(result.Stderr, result.Stdout, "git diff failed"))
			}
			return runtimex.DiffResult{Content: result.Stdout, Bytes: len([]byte(result.Stdout))}, nil
		},
		ReadFile: func(path string) (runtimex.FileReadResult, error) {
			relativePath, err := requirePathInCommitScope(path, scope)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			absolutePath, err := repoFilePath(runtime.CWD, relativePath)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			content, err := os.ReadFile(absolutePath)
			if err != nil {
				return runtimex.FileReadResult{}, err
			}
			return runtimex.FileReadResult{Path: relativePath, Content: string(content), Bytes: len(content)}, nil
		},
		PreviewCommit: func(input runtimex.CommitPreviewInput) (runtimex.CommitPreviewResult, error) {
			return runtimex.CommitPreviewResult{Message: input.Message, FileCount: len(scope.Files)}, nil
		},
		CreateCommits: func(input runtimex.CreateCommitsInput) (runtimex.CreateCommitsResult, error) {
			plan, err := autoCommitPlanFromCreateCommitsInput(input, scope)
			if err != nil {
				return runtimex.CreateCommitsResult{}, err
			}
			if parsed.DryRun {
				return runtimex.CreateCommitsResult{DryRun: true, Status: "previewed", Plan: createCommitPlanResultFromAutoPlan(plan)}, nil
			}
			execution, err := executeCommitPlan(runtime, scope, parsed, plan)
			if err != nil {
				return runtimex.CreateCommitsResult{}, err
			}
			return createCommitsResultFromExecution(execution), nil
		},
	})
}

func createCommitsResultFromExecution(execution autoCommitExecution) runtimex.CreateCommitsResult {
	result := runtimex.CreateCommitsResult{
		Status:       "committed",
		Plan:         createCommitPlanResultFromAutoPlan(execution.Plan),
		Results:      execution.Results,
		Transaction:  execution.Transaction,
		RetryAttempt: execution.RetryAttempt,
		RetryCount:   execution.RetryCount,
	}
	for _, commit := range execution.Results {
		summary := runtimex.CreateCommitResultSummary{
			Hash:     commit.Hash,
			Message:  commit.Message,
			ExitCode: commit.Git.ExitCode,
		}
		if commit.Git.ExitCode != 0 {
			summary.Error = firstNonEmptyString(commit.Git.Stderr, commit.Git.Stdout, "git commit failed")
			result.Status = "commit_failed"
			if result.Error == "" {
				result.Error = summary.Error
			}
		}
		result.Commits = append(result.Commits, summary)
	}
	return result
}

func commitScopePaths(scope gitpkg.CommitScope) []string {
	paths := make([]string, 0, len(scope.Files))
	for _, file := range scope.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

func requirePathInCommitScope(rawPath string, scope gitpkg.CommitScope) (string, error) {
	path, err := normalizeRepoRelativePath(rawPath)
	if err != nil {
		return "", err
	}
	for _, file := range scope.Files {
		if file.Path == path {
			return path, nil
		}
	}
	return "", fmt.Errorf("path is outside Commit Scope: %s", path)
}

func autoCommitPlanFromCreateCommitsInput(input runtimex.CreateCommitsInput, scope gitpkg.CommitScope) (autoCommitPlan, error) {
	if len(input.Commits) == 0 {
		return autoCommitPlan{}, fmt.Errorf("create_commits requires at least one commit")
	}
	filesByPath := map[string]gitpkg.FileStatus{}
	for _, file := range scope.Files {
		filesByPath[file.Path] = file
	}
	commits := make([]autoPlannedCommit, 0, len(input.Commits))
	used := map[string]bool{}
	for _, commit := range input.Commits {
		if strings.TrimSpace(commit.Message) == "" {
			return autoCommitPlan{}, fmt.Errorf("commit message is required")
		}
		files := make([]gitpkg.FileStatus, 0, len(commit.Files))
		for _, path := range commit.Files {
			relativePath, err := normalizeRepoRelativePath(path)
			if err != nil {
				return autoCommitPlan{}, err
			}
			file, ok := filesByPath[relativePath]
			if !ok {
				return autoCommitPlan{}, fmt.Errorf("commit plan references file outside Commit Scope: %s", relativePath)
			}
			if used[relativePath] {
				return autoCommitPlan{}, fmt.Errorf("commit plan assigns file more than once: %s", relativePath)
			}
			used[relativePath] = true
			files = append(files, file)
		}
		if len(files) == 0 {
			return autoCommitPlan{}, fmt.Errorf("commit plan contains an empty commit")
		}
		commits = append(commits, autoPlannedCommit{Message: commit.Message, Files: files})
	}
	return autoCommitPlan{Kind: emptyDefaultString(input.Kind, defaultCommitPlanKind(len(commits))), Commits: commits, SplitLimitations: autoSplitLimitationsFromRuntime(input.SplitLimitations)}, nil
}

func autoSplitLimitationsFromRuntime(limitations []runtimex.CreateSplitLimitation) []autoSplitLimitation {
	if len(limitations) == 0 {
		return nil
	}
	result := make([]autoSplitLimitation, 0, len(limitations))
	for _, limitation := range limitations {
		result = append(result, autoSplitLimitation{Code: limitation.Code, Message: limitation.Message, Fallback: limitation.Fallback})
	}
	return result
}

func createCommitPlanResultFromAutoPlan(plan autoCommitPlan) runtimex.CreateCommitPlanResult {
	commits := make([]runtimex.CreateCommitPlanCommit, 0, len(plan.Commits))
	for _, commit := range plan.Commits {
		files := make([]string, 0, len(commit.Files))
		for _, file := range commit.Files {
			files = append(files, file.Path)
		}
		commits = append(commits, runtimex.CreateCommitPlanCommit{Message: commit.Message, Files: files})
	}
	return runtimex.CreateCommitPlanResult{Kind: plan.Kind, Commits: commits, SplitLimitations: runtimeSplitLimitationsFromAuto(plan.SplitLimitations)}
}

func runtimeSplitLimitationsFromAuto(limitations []autoSplitLimitation) []runtimex.CreateSplitLimitation {
	if len(limitations) == 0 {
		return nil
	}
	result := make([]runtimex.CreateSplitLimitation, 0, len(limitations))
	for _, limitation := range limitations {
		result = append(result, runtimex.CreateSplitLimitation{Code: limitation.Code, Message: limitation.Message, Fallback: limitation.Fallback})
	}
	return result
}

func autoCommitExecutionFromToolCalls(calls []runtimex.ToolCallResult) (autoCommitExecution, bool) {
	failedCreateCallsBeforeFinal := 0
	for index := len(calls) - 1; index >= 0; index-- {
		call := calls[index]
		if call.Name != runtimex.ToolCreateCommits || !call.OK {
			continue
		}
		created, ok := call.Result.(runtimex.CreateCommitsResult)
		if !ok {
			continue
		}
		for _, prior := range calls[:index] {
			if prior.Name != runtimex.ToolCreateCommits || !prior.OK {
				continue
			}
			priorCreated, ok := prior.Result.(runtimex.CreateCommitsResult)
			if ok && priorCreated.Status == "commit_failed" {
				failedCreateCallsBeforeFinal++
			}
		}
		retryAttempt := created.RetryAttempt
		retryCount := created.RetryCount
		if created.Status == "committed" && failedCreateCallsBeforeFinal > 0 {
			retryAttempt = true
			retryCount = failedCreateCallsBeforeFinal
		}
		return autoCommitExecution{
			Plan:         autoPlanFromCreateCommitPlanResult(created.Plan),
			Results:      created.Results,
			Transaction:  created.Transaction,
			RetryAttempt: retryAttempt,
			RetryCount:   retryCount,
		}, true
	}
	return autoCommitExecution{}, false
}

func autoPlanFromCreateCommitPlanResult(plan runtimex.CreateCommitPlanResult) autoCommitPlan {
	commits := make([]autoPlannedCommit, 0, len(plan.Commits))
	for _, commit := range plan.Commits {
		files := make([]gitpkg.FileStatus, 0, len(commit.Files))
		for _, path := range commit.Files {
			files = append(files, gitpkg.FileStatus{Path: path})
		}
		commits = append(commits, autoPlannedCommit{Message: commit.Message, Files: files})
	}
	return autoCommitPlan{Kind: plan.Kind, Commits: commits, SplitLimitations: autoSplitLimitationsFromRuntime(plan.SplitLimitations)}
}

func defaultCommitPlanKind(commitCount int) string {
	if commitCount > 1 {
		return "file_split"
	}
	return "single"
}

func executeCommitPlan(runtime Runtime, scope gitpkg.CommitScope, parsed autoArgs, commitPlan autoCommitPlan) (autoCommitExecution, error) {
	execution := autoCommitExecution{Plan: commitPlan}
	snapshot, snapshotErr := gitpkg.CaptureCommitTransactionSnapshot(runtime.CWD, runtime.Env, nil)
	if snapshotErr != nil {
		return autoCommitExecution{}, snapshotErr
	}
	for _, plannedCommit := range commitPlan.Commits {
		commitScope := scope
		commitScope.Files = append([]gitpkg.FileStatus{}, plannedCommit.Files...)
		commitResult, err := gitpkg.CommitScopeWithMessage(gitpkg.CommitScopeCommitOptions{
			CWD:      runtime.CWD,
			Env:      runtime.Env,
			Scope:    commitScope,
			Message:  plannedCommit.Message,
			NoVerify: parsed.NoVerify,
		})
		if err != nil {
			return autoCommitExecution{}, err
		}
		execution.Results = append(execution.Results, commitResult)
		if commitResult.Git.ExitCode != 0 {
			if len(execution.successfulResults()) > 0 {
				execution.Transaction = gitpkg.RollbackCommitTransaction(runtime.CWD, runtime.Env, snapshot, nil)
			}
			return execution, nil
		}
	}
	return execution, nil
}

func autoCommitPlanFromTUIPlan(plan tui.CommitPlanView, scope gitpkg.CommitScope) (autoCommitPlan, error) {
	if len(plan.Commits) == 0 {
		return autoCommitPlan{}, fmt.Errorf("commit plan is required")
	}
	filesByPath := map[string]gitpkg.FileStatus{}
	for _, file := range scope.Files {
		filesByPath[file.Path] = file
	}
	commits := make([]autoPlannedCommit, 0, len(plan.Commits))
	used := map[string]bool{}
	for _, commit := range plan.Commits {
		if strings.TrimSpace(commit.Message) == "" {
			return autoCommitPlan{}, fmt.Errorf("commit message is required")
		}
		files := make([]gitpkg.FileStatus, 0, len(commit.Files))
		for _, path := range commit.Files {
			file, ok := filesByPath[path]
			if !ok {
				return autoCommitPlan{}, fmt.Errorf("commit plan references file outside Commit Scope: %s", path)
			}
			if used[path] {
				return autoCommitPlan{}, fmt.Errorf("commit plan assigns file more than once: %s", path)
			}
			used[path] = true
			files = append(files, file)
		}
		if len(files) == 0 {
			return autoCommitPlan{}, fmt.Errorf("commit plan contains an empty commit")
		}
		commits = append(commits, autoPlannedCommit{Message: commit.Message, Files: files})
	}
	return autoCommitPlan{Kind: emptyDefaultString(plan.Kind, "single"), Commits: commits}, nil
}

func filterScopeByPaths(scope gitpkg.CommitScope, paths []string) gitpkg.CommitScope {
	if len(paths) == 0 {
		return scope
	}
	selected := map[string]bool{}
	for _, path := range paths {
		selected[path] = true
	}
	filtered := scope
	filtered.Files = make([]gitpkg.FileStatus, 0, len(scope.Files))
	for _, file := range scope.Files {
		if selected[file.Path] {
			filtered.Files = append(filtered.Files, file)
		}
	}
	filtered.HasSelectedChanges = len(filtered.Files) > 0
	return filtered
}

func (e autoCommitExecution) lastResult() gitpkg.CommitScopeCommitResult {
	if len(e.Results) == 0 {
		return gitpkg.CommitScopeCommitResult{Git: gitpkg.CommandResult{ExitCode: 1, Stderr: "no commit attempts were made"}}
	}
	return e.Results[len(e.Results)-1]
}

func (e autoCommitExecution) commitSummaries() []map[string]any {
	commits := make([]map[string]any, 0, len(e.Results))
	for _, result := range e.successfulResults() {
		commits = append(commits, map[string]any{"hash": result.Hash, "message": result.Message})
	}
	return commits
}

func (e autoCommitExecution) successfulResults() []gitpkg.CommitScopeCommitResult {
	results := make([]gitpkg.CommitScopeCommitResult, 0, len(e.Results))
	for _, result := range e.Results {
		if result.Git.ExitCode != 0 {
			continue
		}
		results = append(results, result)
	}
	return results
}

func tuiCommitPlanFromAutoPlan(plan autoCommitPlan) tui.CommitPlanView {
	commits := make([]tui.CommitView, 0, len(plan.Commits))
	for _, commit := range plan.Commits {
		files := make([]string, 0, len(commit.Files))
		for _, file := range commit.Files {
			files = append(files, file.Path)
		}
		commits = append(commits, tui.CommitView{Message: commit.Message, Files: files})
	}
	return tui.CommitPlanView{Kind: plan.Kind, Commits: commits}
}

func scopeHasConflict(scope gitpkg.CommitScope) bool {
	for _, file := range scope.Files {
		if scopeFileHasConflict(file) {
			return true
		}
	}
	return false
}

func scopeFileHasConflict(file gitpkg.FileStatus) bool {
	return isUnmergedChange(file.Staged) || isUnmergedChange(file.Unstaged)
}

func isUnmergedChange(change *gitpkg.ChangeKind) bool {
	return change != nil && *change == gitpkg.ChangeUnmerged
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func emptyDefaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseAutoArgs(args []string) (autoArgs, error) {
	parsed := autoArgs{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			parsed.Pathspecs = append(parsed.Pathspecs, args[index+1:]...)
			return parsed, nil
		}
		switch arg {
		case "--help", "-h":
			return parsed, errAutoHelp
		case "--dry-run":
			parsed.DryRun = true
		case "--json":
			parsed.JSON = true
		case "--tui", "-i":
			parsed.TUI = true
		case "--staged":
			parsed.Staged = true
		case "--no-untracked":
			parsed.NoUntracked = true
		case "--diff-only":
			parsed.DiffOnly = true
		case "--no-verify":
			parsed.NoVerify = true
		case "--verbose", "-v":
			parsed.Verbose = true
		case "--instruction", "-m":
			if index+1 >= len(args) {
				return parsed, fmt.Errorf("%s requires a value", arg)
			}
			parsed.Instruction = args[index+1]
			index++
		default:
			if strings.HasPrefix(arg, "--instruction=") {
				parsed.Instruction = strings.TrimPrefix(arg, "--instruction=")
				continue
			}
			if strings.HasPrefix(arg, "-m=") {
				parsed.Instruction = strings.TrimPrefix(arg, "-m=")
				continue
			}
			if strings.HasPrefix(arg, "-") {
				return parsed, fmt.Errorf("unknown auto option %q", arg)
			}
			return parsed, fmt.Errorf("unexpected auto argument %q; pass pathspecs after --", arg)
		}
	}
	return parsed, nil
}

var errAutoHelp = fmt.Errorf("auto help requested")

func runRoot(args []string, runtime Runtime) int {
	parsed, err := parseRootArgs(args)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.UsageError)
	}

	flagOverrides, err := rootFlagOverrides(parsed)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}

	resolvedConfig, configErr := config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: runtime.CWD, Env: runtime.Env, FlagOverrides: flagOverrides, SecretStore: runtime.SecretStore})
	if !parsed.JSON && runtime.IsTTY && configErr == nil && needsOnboarding(resolvedConfig.Values) {
		if err := runFirstRunOnboarding(runtime, resolvedConfig.Values); err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		resolvedConfig, configErr = config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: runtime.CWD, Env: runtime.Env, FlagOverrides: flagOverrides, SecretStore: runtime.SecretStore})
	}
	preferenceSources := gitpkg.PreferenceExposure{}
	if configErr == nil {
		sourceSummary := config.SummarizeConfigSources(resolvedConfig)
		preferenceSources = gitpkg.PreferenceExposure{
			Provider:            string(sourceSummary.Provider),
			Model:               string(sourceSummary.Model),
			APIKey:              string(sourceSummary.APIKey),
			PromptStyle:         string(sourceSummary.PromptStyle),
			MessageLanguage:     string(sourceSummary.MessageLanguage),
			StandingInstruction: sourceSummary.StandingInstruction,
		}
	}
	scope, scopeErr := gitpkg.InspectCommitScope(gitpkg.CommitScopeOptions{
		CWD:               runtime.CWD,
		Env:               runtime.Env,
		StagedOnly:        parsed.Staged,
		IncludeUntracked:  !parsed.NoUntracked,
		DiffOnly:          parsed.DiffOnly,
		PreferenceSources: preferenceSources,
		Pathspecs:         parsed.Pathspecs,
	})
	if parsed.JSON {
		router := output.NewRouter(true, runtime.Stdout, runtime.Stderr)
		if scopeErr != nil {
			_ = router.WriteJSON(map[string]any{"schemaVersion": 1, "command": "cnm", "ok": false, "status": "scope_error", "scopeError": scopeErr.Error()})
			return int(output.Error)
		}
		_ = router.WriteJSON(map[string]any{"schemaVersion": 1, "command": "cnm", "ok": true, "status": "tui_preview", "scope": scope, "contextPolicy": scope.ContextPolicy, "aiExposure": scope.AIExposure})
		return int(output.Success)
	}
	if configErr != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", configErr)
		return int(output.Error)
	}
	if scopeErr != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", scopeErr)
		return int(output.Error)
	}
	modelInput := tui.ModelInput{
		Scope:        scope,
		PlanCommits:  interactiveCommitPlanner(runtime, parsed, resolvedConfig.Values, scope),
		DiffProvider: workingTreeDiffProvider(runtime, scope),
		FileReader:   repairFileReader(runtime),
	}
	if runtime.IsTTY {
		tuiRunner := runtime.TUIRunner
		if tuiRunner == nil {
			tuiRunner = tui.Run
		}
		result, err := tuiRunner(modelInput, tui.Runtime{Input: runtime.Stdin, Output: runtime.Stdout})
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		if !result.Accepted || result.Cancelled {
			fmt.Fprintln(runtime.Stderr, "Cancelled. No commit was created.")
			return int(output.UserCancel)
		}
		executionScope := filterScopeByPaths(scope, result.ScopeFiles)
		commitPlan, err := autoCommitPlanFromTUIPlan(result.CommitPlan, executionScope)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		execution, err := executeCommitPlan(runtime, executionScope, autoArgs{NoVerify: parsed.NoVerify}, commitPlan)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		commitResult := execution.lastResult()
		if commitResult.Git.ExitCode != 0 {
			fmt.Fprintln(runtime.Stderr, firstNonEmptyString(commitResult.Git.Stderr, commitResult.Git.Stdout, "git commit failed"))
			return int(output.Error)
		}
		fmt.Fprintf(runtime.Stdout, "Committed %s %s\n", commitResult.Hash, commitResult.Message)
		return int(output.Success)
	}
	modelInput.NoColor = true
	fmt.Fprint(runtime.Stdout, tui.NewModel(modelInput).View())
	return int(output.Success)
}

func interactiveCommitPlanner(runtime Runtime, parsed rootArgs, resolvedConfig config.EffectiveConfig, initialScope gitpkg.CommitScope) tui.PlanCommitsFunc {
	return func(input tui.PlanCommitsInput) (tui.CommitPlanView, error) {
		executionScope := filterScopeByPaths(input.Scope, input.ScopeFiles)
		executionScope.StagedOnly = initialScope.StagedOnly
		executionScope.IncludesUntracked = initialScope.IncludesUntracked
		execution, err := executeAutoCommit(runtime, executionScope, autoArgs{DryRun: true, Staged: parsed.Staged, NoUntracked: parsed.NoUntracked, DiffOnly: parsed.DiffOnly, NoVerify: parsed.NoVerify, Pathspecs: parsed.Pathspecs}, resolvedConfig)
		if err != nil {
			return tui.CommitPlanView{}, err
		}
		return tuiCommitPlanFromAutoPlan(execution.Plan), nil
	}
}

type rootArgs struct {
	JSON                bool
	Provider            string
	Model               string
	BaseURL             string
	PromptStyle         string
	MessageLanguage     string
	StandingInstruction string
	Staged              bool
	NoUntracked         bool
	DiffOnly            bool
	NoVerify            bool
	Pathspecs           []string
}

func parseRootArgs(args []string) (rootArgs, error) {
	parsed := rootArgs{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			parsed.Pathspecs = append(parsed.Pathspecs, args[index+1:]...)
			return parsed, nil
		}
		switch arg {
		case "--json":
			parsed.JSON = true
		case "--staged":
			parsed.Staged = true
		case "--no-untracked":
			parsed.NoUntracked = true
		case "--diff-only":
			parsed.DiffOnly = true
		case "--no-verify":
			parsed.NoVerify = true
		case "--provider":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.Provider = value
			index = next
		case "--model":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.Model = value
			index = next
		case "--base-url":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.BaseURL = value
			index = next
		case "--prompt-style":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.PromptStyle = value
			index = next
		case "--message-language":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.MessageLanguage = value
			index = next
		case "--standing-instruction":
			value, next, err := requireNextArg(args, index)
			if err != nil {
				return parsed, err
			}
			parsed.StandingInstruction = value
			index = next
		default:
			if strings.HasPrefix(arg, "-") {
				return parsed, fmt.Errorf("unknown option %q", arg)
			}
			return parsed, fmt.Errorf("unknown command '%s'", arg)
		}
	}
	return parsed, nil
}

func requireNextArg(args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("missing value for %s", args[index])
	}
	return args[index+1], index + 1, nil
}

func rootFlagOverrides(args rootArgs) (config.ConfigValues, error) {
	overrides := config.ConfigValues{}
	if strings.TrimSpace(args.Provider) != "" {
		value, err := config.ParseKeyValue("provider", args.Provider)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	if strings.TrimSpace(args.Model) != "" {
		value, err := config.ParseKeyValue("model", args.Model)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	if strings.TrimSpace(args.BaseURL) != "" {
		value, err := config.ParseKeyValue("baseURL", args.BaseURL)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	if strings.TrimSpace(args.PromptStyle) != "" {
		value, err := config.ParseKeyValue("promptStyle", args.PromptStyle)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	if strings.TrimSpace(args.MessageLanguage) != "" {
		value, err := config.ParseKeyValue("messageLanguage", args.MessageLanguage)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	if strings.TrimSpace(args.StandingInstruction) != "" {
		value, err := config.ParseKeyValue("standingInstruction", args.StandingInstruction)
		if err != nil {
			return overrides, err
		}
		overrides = mergeConfigOverrides(overrides, value)
	}
	return overrides, nil
}

func needsOnboarding(values config.EffectiveConfig) bool {
	if values.APIKey == nil || strings.TrimSpace(*values.APIKey) == "" {
		return true
	}
	if strings.TrimSpace(values.Model) == "" {
		return true
	}
	return values.Provider == config.ProviderOpenAICompatible && (values.BaseURL == nil || strings.TrimSpace(*values.BaseURL) == "")
}

func runFirstRunOnboarding(runtime Runtime, current config.EffectiveConfig) error {
	if runtime.IsTTY && runtime.OnboardingRunner != nil {
		return runFirstRunOnboardingTUI(runtime, current)
	}
	prompter := interactive.NewPrompter(runtime.Stdin, runtime.Stdout)
	if _, err := fmt.Fprintln(runtime.Stdout, "Onboarding"); err != nil {
		return err
	}
	provider, err := askOnboardingProvider(prompter)
	if err != nil {
		return err
	}
	model, err := askOnboardingText(prompter, "Model", config.GetDefaultModel(provider), true)
	if err != nil {
		return err
	}
	var baseURL *string
	if provider == config.ProviderOpenAICompatible {
		value, err := askOnboardingText(prompter, "Base URL", "", true)
		if err != nil {
			return err
		}
		baseURL = &value
	} else {
		baseURL = current.BaseURL
	}
	promptStyle, err := askOnboardingPromptStyle(prompter)
	if err != nil {
		return err
	}
	messageLanguage, err := askOnboardingMessageLanguage(prompter)
	if err != nil {
		return err
	}
	standingInstruction, err := askOnboardingText(prompter, "Standing Instruction", stringPointerValue(current.StandingInstruction), false)
	if err != nil {
		return err
	}
	apiKey, err := askOnboardingText(prompter, "API key", "", true)
	if err != nil {
		return err
	}

	save, err := config.SaveAPIKeyToSecretStore(provider, apiKey, runtime.SecretStore)
	if err != nil {
		return fmt.Errorf("%v Use `cnm init --plaintext-api-key` to store this key in user config instead.", err)
	}
	if !save.Stored {
		message := "Secret Store is not available; use `cnm init --plaintext-api-key` to store this key in user config instead."
		if save.Warning != nil {
			message = *save.Warning
		}
		return fmt.Errorf("%s", message)
	}
	patch := config.ConfigValues{
		Provider:            &provider,
		Model:               &model,
		BaseURL:             baseURL,
		PromptStyle:         &promptStyle,
		MessageLanguage:     &messageLanguage,
		StandingInstruction: optionalStringPointer(standingInstruction),
	}
	if _, err := config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(runtime.Stdout, "Onboarding complete.")
	return err
}

func runFirstRunOnboardingTUI(runtime Runtime, current config.EffectiveConfig) error {
	input := tui.OnboardingInput{
		CurrentProvider: current.Provider,
		CurrentModel:    current.Model,
		CurrentStyle:    current.PromptStyle,
		CurrentLanguage: current.MessageLanguage,
	}
	if current.BaseURL != nil {
		input.CurrentBaseURL = *current.BaseURL
	}
	if current.StandingInstruction != nil {
		input.CurrentStanding = *current.StandingInstruction
	}
	result, err := runtime.OnboardingRunner(input, tui.Runtime{Input: runtime.Stdin, Output: runtime.Stdout})
	if err != nil {
		return err
	}
	if result.Cancelled {
		return fmt.Errorf("onboarding cancelled")
	}
	provider := result.Provider
	if provider == "" {
		provider = config.DefaultProvider
	}
	model := strings.TrimSpace(result.Model)
	if model == "" {
		model = config.GetDefaultModel(provider)
	}
	apiKey := strings.TrimSpace(result.APIKey)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	save, err := config.SaveAPIKeyToSecretStore(provider, apiKey, runtime.SecretStore)
	if err != nil {
		return fmt.Errorf("%v Use `cnm init --plaintext-api-key` to store this key in user config instead.", err)
	}
	if !save.Stored {
		message := "Secret Store is not available; use `cnm init --plaintext-api-key` to store this key in user config instead."
		if save.Warning != nil {
			message = *save.Warning
		}
		return fmt.Errorf("%s", message)
	}
	var baseURL *string
	if provider == config.ProviderOpenAICompatible || strings.TrimSpace(result.BaseURL) != "" {
		baseURL = optionalStringPointer(result.BaseURL)
	}
	style := result.PromptStyle
	if style == "" {
		style = config.DefaultPromptStyle
	}
	language := result.MessageLanguage
	if language == "" {
		language = config.DefaultMessageLanguage
	}
	patch := config.ConfigValues{
		Provider:            &provider,
		Model:               &model,
		BaseURL:             baseURL,
		PromptStyle:         &style,
		MessageLanguage:     &language,
		StandingInstruction: optionalStringPointer(result.StandingInstruction),
	}
	if _, err := config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(runtime.Stdout, "Onboarding complete.")
	return err
}

func askOnboardingProvider(prompter *interactive.Prompter) (config.ProviderType, error) {
	choice, err := prompter.AskChoice("Provider (openai-responses, openai-compatible, anthropic-messages, google-gemini)", providerChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.ProviderType(choice), nil
}

func askOnboardingPromptStyle(prompter *interactive.Prompter) (config.PromptStyle, error) {
	choice, err := prompter.AskChoice("Prompt style (auto, conventional, angular, google, atom, plain, custom)", promptStyleChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.PromptStyle(choice), nil
}

func askOnboardingMessageLanguage(prompter *interactive.Prompter) (config.MessageLanguage, error) {
	choice, err := prompter.AskChoice("Message language (auto, en, zh-CN, zh-TW)", messageLanguageChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.MessageLanguage(choice), nil
}

func askOnboardingText(prompter *interactive.Prompter, label string, defaultValue string, required bool) (string, error) {
	for {
		value, err := prompter.AskText(label, defaultValue)
		if err != nil {
			return "", err
		}
		trimmed := strings.TrimSpace(value)
		if trimmed != "" || !required {
			return trimmed, nil
		}
	}
}

func providerChoiceStrings() []string {
	values := make([]string, 0, len(config.ProviderTypes))
	for _, value := range config.ProviderTypes {
		values = append(values, string(value))
	}
	return values
}

func promptStyleChoiceStrings() []string {
	values := make([]string, 0, len(config.PromptStyles))
	for _, value := range config.PromptStyles {
		values = append(values, string(value))
	}
	return values
}

func messageLanguageChoiceStrings() []string {
	values := make([]string, 0, len(config.MessageLanguages))
	for _, value := range config.MessageLanguages {
		values = append(values, string(value))
	}
	return values
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mergeConfigOverrides(base config.ConfigValues, patch config.ConfigValues) config.ConfigValues {
	result := base
	if patch.Provider != nil {
		result.Provider = patch.Provider
	}
	if patch.Model != nil {
		result.Model = patch.Model
	}
	if patch.BaseURL != nil {
		result.BaseURL = patch.BaseURL
	}
	if patch.PromptStyle != nil {
		result.PromptStyle = patch.PromptStyle
	}
	if patch.MessageLanguage != nil {
		result.MessageLanguage = patch.MessageLanguage
	}
	if patch.StandingInstruction != nil {
		result.StandingInstruction = patch.StandingInstruction
	}
	if patch.APIKey != nil {
		result.APIKey = patch.APIKey
	}
	return result
}

func resolveCommand(args []string) (string, int) {
	for index := 0; index < len(args); index++ {
		token := args[index]
		if token == "--" {
			return "", -1
		}
		if isRootFlagWithValue(token) {
			index++
			continue
		}
		if strings.HasPrefix(token, "-") {
			continue
		}
		if token == "init" || token == "config" || token == "doctor" || token == "split" || token == "help" {
			return token, index
		}
		return token, index
	}
	return "", -1
}

func isRootFlagWithValue(token string) bool {
	switch token {
	case "--provider", "--model", "--base-url", "--prompt-style", "--message-language", "--standing-instruction":
		return true
	default:
		return false
	}
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func printUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: cnm [command] [flags]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  cnm          Start Interactive Commit in the Full-screen TUI")
	fmt.Fprintln(stdout, "  cnm auto     Run Autonomous Commit without step-by-step confirmation")
	fmt.Fprintln(stdout, "  cnm init     Run Onboarding and configure preferences")
	fmt.Fprintln(stdout, "  cnm config   Inspect and edit preferences")
	fmt.Fprintln(stdout, "  cnm doctor   Diagnose local setup and provider capability")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Flags:")
	fmt.Fprintln(stdout, "  --json                         Emit a TUI preview Machine Output Contract")
	fmt.Fprintln(stdout, "  --staged                       Restrict Commit Scope to staged changes")
	fmt.Fprintln(stdout, "  --no-untracked                 Exclude untracked files")
	fmt.Fprintln(stdout, "  --diff-only                    Use diff-only Context Policy")
	fmt.Fprintln(stdout, "  --no-verify                    Pass --no-verify to git commit after confirmation")
	fmt.Fprintln(stdout, "  --provider <provider>          Override provider for this run")
	fmt.Fprintln(stdout, "  --model <model>                Override model for this run")
	fmt.Fprintln(stdout, "  --base-url <url>               Override provider base URL for this run")
	fmt.Fprintln(stdout, "  --prompt-style <style>         Override commit message style for this run")
	fmt.Fprintln(stdout, "  --message-language <language>  Override generated message language")
	fmt.Fprintln(stdout, "  --standing-instruction <text>  Add standing instruction for this run")
	fmt.Fprintln(stdout, "  -- <pathspec...>               Limit the Commit Scope with Git pathspecs")
}

func printAutoUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage: cnm auto [flags] [-- <pathspec...>]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Autonomous Commit creates local commits from the selected Commit Scope without step-by-step confirmation.")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Flags:")
	fmt.Fprintln(stdout, "  --dry-run       Show a Commit Plan Preview without creating commits")
	fmt.Fprintln(stdout, "  --json          Emit the Machine Output Contract")
	fmt.Fprintln(stdout, "  --tui, -i       Hand off to the Full-screen TUI when developer judgment is needed")
	fmt.Fprintln(stdout, "  --staged        Restrict Commit Scope to staged changes")
	fmt.Fprintln(stdout, "  --no-untracked  Exclude untracked files")
	fmt.Fprintln(stdout, "  --diff-only     Use diff-only Context Policy")
	fmt.Fprintln(stdout, "  --no-verify     Pass --no-verify to git commit")
	fmt.Fprintln(stdout, "  --instruction <text>, -m <text>  Per-run hint to the AI (e.g. context for unusual changes)")
	fmt.Fprintln(stdout, "  --verbose, -v   Show detailed run output")
}

func normalizeRuntime(runtime Runtime) Runtime {
	if strings.TrimSpace(runtime.CWD) == "" {
		cwd, err := os.Getwd()
		if err == nil {
			runtime.CWD = cwd
		} else {
			runtime.CWD = "."
		}
	}
	if runtime.Stdin == nil {
		runtime.Stdin = os.Stdin
	}
	if runtime.Stdout == nil {
		runtime.Stdout = os.Stdout
	}
	if runtime.Stderr == nil {
		runtime.Stderr = os.Stderr
	}
	if runtime.Env == nil {
		runtime.Env = map[string]string{}
	}
	return runtime
}

func isTTYDefault() bool {
	stdinInfo, stdinErr := os.Stdin.Stat()
	stdoutInfo, stdoutErr := os.Stdout.Stat()
	if stdinErr != nil || stdoutErr != nil {
		return false
	}
	return (stdinInfo.Mode()&os.ModeCharDevice) != 0 && (stdoutInfo.Mode()&os.ModeCharDevice) != 0
}
