package doctor

import (
	"os"
	"runtime"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
	"github.com/ByteTrue/commit-now-myfriend/internal/providers"
)

type RunOptions struct {
	CWD           string
	Env           map[string]string
	SecretStore   config.SecretStore
	ProbeProvider bool
	Probe         func(ProbeInput) ProbeResult
}

const fixedProviderProbeContent = "cnm provider capability probe: fixed non-repository content."

func Run(options RunOptions) (Report, error) {
	resolvedConfig, err := config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: options.CWD, Env: options.Env, SecretStore: options.SecretStore})
	if err != nil {
		return Report{}, err
	}

	inspection, err := gitpkg.InspectRepository(gitpkg.InspectOptions{CWD: options.CWD, Env: options.Env})
	if err != nil {
		return Report{}, err
	}

	report := Report{
		Command:  "cnm doctor",
		OK:       true,
		ReadOnly: true,
		Status:   "ok",
		Issues:   []Issue{},
	}
	report.Paths.ProjectConfigPath = resolvedConfig.Paths.ProjectConfigPath
	report.Paths.UserConfigHome = resolvedConfig.Paths.UserConfigHome
	report.Paths.UserConfigPath = resolvedConfig.Paths.UserConfigPath

	gitVersion, gitAvailable := detectGitVersion(options.CWD, options.Env)
	report.Checks.Git.Available = gitAvailable
	report.Checks.Git.Version = gitVersion
	if gitAvailable {
		report.Checks.Git.Status = CheckStatusPass
		report.Checks.Git.Message = "Git is available."
	} else {
		report.Checks.Git.Status = CheckStatusError
		report.Checks.Git.Message = "Git is not available on PATH."
		report.Issues = append(report.Issues, Issue{Code: "git_unavailable", Check: "git", Message: report.Checks.Git.Message, Severity: SeverityError})
	}

	report.Checks.Repository.IsRepository = inspection.Repository.IsRepository
	report.Checks.Repository.RootPath = inspection.Repository.RootPath
	if inspection.Repository.IsRepository {
		report.Checks.Repository.Status = CheckStatusPass
		report.Checks.Repository.Message = "Repository inspection completed."
	} else {
		report.Checks.Repository.Status = CheckStatusWarning
		report.Checks.Repository.Message = "Current directory is not inside a git repository."
		report.Issues = append(report.Issues, Issue{Code: "not_git_repository", Check: "repository", Message: report.Checks.Repository.Message, Severity: SeverityWarning})
	}

	cfgView := config.ToJSONConfigView(resolvedConfig.Values)
	report.Checks.EffectiveConfig.Config = cfgView
	report.Checks.EffectiveConfig.Sources = ConfigSources{
		Provider: sourceLabel(resolvedConfig.UserConfig.Provider != nil, resolvedConfig.ProjectConfig.Provider != nil, resolvedConfig.EnvConfig.Provider != nil, true),
		Model:    sourceLabel(resolvedConfig.UserConfig.Model != nil, resolvedConfig.ProjectConfig.Model != nil, resolvedConfig.EnvConfig.Model != nil, true),
		BaseURL:  sourceLabel(resolvedConfig.UserConfig.BaseURL != nil, resolvedConfig.ProjectConfig.BaseURL != nil, resolvedConfig.EnvConfig.BaseURL != nil, resolvedConfig.Values.BaseURL != nil),
		APIKey:   sourceLabel(resolvedConfig.UserConfig.APIKey != nil, false, resolvedConfig.EnvConfig.APIKey != nil, resolvedConfig.Values.APIKey != nil),
	}
	configIssues := collectConfigIssues(resolvedConfig, inspection)
	report.Issues = append(report.Issues, configIssues...)
	if len(configIssues) == 0 {
		report.Checks.EffectiveConfig.Status = CheckStatusPass
		report.Checks.EffectiveConfig.Message = "Effective provider configuration looks usable."
	} else {
		report.Checks.EffectiveConfig.Status = CheckStatusError
		report.Checks.EffectiveConfig.Message = "Provider configuration needs attention."
	}

	for _, warning := range resolvedConfig.Warnings {
		report.Issues = append(report.Issues, Issue{Code: "project_api_key_ignored", Check: "effectiveConfig", Message: warning, Severity: SeverityWarning})
	}
	if hasInsecureUserConfigPermissions(report.Paths.UserConfigPath) {
		report.Issues = append(report.Issues, Issue{Code: "user_config_permissions_insecure", Check: "effectiveConfig", Message: "User config permissions are broader than 0600.", Severity: SeverityWarning})
	}
	for _, gitIssue := range append(inspection.BlockingIssues, inspection.Warnings...) {
		severity := SeverityWarning
		if gitIssue.Severity == gitpkg.IssueSeverityBlocking && gitIssue.Code != "not_git_repository" {
			severity = SeverityError
		}
		report.Issues = append(report.Issues, Issue{Code: gitIssue.Code, Check: "repository", Message: gitIssue.Message, Severity: severity})
	}

	if capability, ok := providers.CapabilityForProvider(resolvedConfig.Values.Provider); ok {
		report.Checks.ProviderCapability.Status = CheckStatusPass
		report.Checks.ProviderCapability.Capability = capability
		report.Checks.ProviderCapability.Message = "Provider protocol supports native Domain Tool calls."
	} else {
		report.Checks.ProviderCapability.Status = CheckStatusError
		report.Checks.ProviderCapability.Message = "Provider protocol capability metadata is unavailable."
		report.Issues = append(report.Issues, Issue{Code: "provider_capability_missing", Check: "providerCapability", Message: report.Checks.ProviderCapability.Message, Severity: SeverityError})
	}

	if options.ProbeProvider {
		probe := options.Probe
		if probe == nil {
			probe = defaultProbeProvider
		}
		result := probe(ProbeInput{Provider: resolvedConfig.Values.Provider, Model: resolvedConfig.Values.Model, Content: fixedProviderProbeContent})
		report.Probe = &result
		if result.Status != "ok" && result.Status != "skipped" {
			report.Issues = append(report.Issues, Issue{Code: "provider_probe_failed", Check: "providerProbe", Message: firstNonEmpty(result.Error, result.Message, "Provider probe failed."), Severity: SeverityError})
		}
	}

	for _, issue := range report.Issues {
		if issue.Severity == SeverityError {
			report.Summary.Errors++
		} else if issue.Severity == SeverityWarning {
			report.Summary.Warnings++
		}
	}
	if report.Summary.Errors > 0 {
		report.OK = false
		report.Status = "issues_found"
	}

	return report, nil
}

func defaultProbeProvider(input ProbeInput) ProbeResult {
	return ProbeResult{Status: "skipped", Message: "Provider probe transport is not configured in this build."}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func detectGitVersion(cwd string, env map[string]string) (*string, bool) {
	result, err := gitpkg.DefaultCommandRunner(cwd, []string{"--version"}, env)
	if err != nil || result.ExitCode != 0 {
		return nil, false
	}
	version := strings.TrimSpace(result.Stdout)
	if version == "" {
		return nil, false
	}
	return &version, true
}

func collectConfigIssues(resolved config.ResolvedConfig, inspection gitpkg.Inspection) []Issue {
	issues := make([]Issue, 0)
	defaultProviderOnly := resolved.UserConfig.Provider == nil && resolved.ProjectConfig.Provider == nil && resolved.EnvConfig.Provider == nil
	defaultModelOnly := resolved.UserConfig.Model == nil && resolved.ProjectConfig.Model == nil && resolved.EnvConfig.Model == nil
	if defaultProviderOnly && defaultModelOnly {
		issues = append(issues, Issue{Code: "provider_config_missing", Check: "effectiveConfig", Message: "Provider configuration is still using built-in defaults; run `cnm init` or configure cnm before generating commits.", Severity: SeverityError})
	}
	if resolved.Values.APIKey == nil || strings.TrimSpace(*resolved.Values.APIKey) == "" {
		issues = append(issues, Issue{Code: "api_key_missing", Check: "effectiveConfig", Message: "No API key is configured for " + string(resolved.Values.Provider) + ".", Severity: SeverityError})
	}
	if resolved.Values.Provider == config.ProviderOpenAICompatible && (resolved.Values.BaseURL == nil || strings.TrimSpace(*resolved.Values.BaseURL) == "") {
		issues = append(issues, Issue{Code: "provider_config_missing", Check: "effectiveConfig", Message: "The openai-compatible provider requires `baseURL` to be configured.", Severity: SeverityError})
	}
	if !inspection.Repository.HasGitIdentity && inspection.Repository.IsRepository {
		issues = append(issues, Issue{Code: "git_identity_missing", Check: "repository", Message: "Git user.name and user.email must be configured before committing.", Severity: SeverityError})
	}
	return issues
}

func sourceLabel(user bool, project bool, env bool, hasDefault bool) string {
	if env {
		return "env"
	}
	if project {
		return "project"
	}
	if user {
		return "user"
	}
	if hasDefault {
		return "default"
	}
	return "missing"
}

func hasInsecureUserConfigPermissions(path string) bool {
	if runtime.GOOS == "windows" || strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o077 != 0
}
