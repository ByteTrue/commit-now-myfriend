package commands

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/interactive"
	"github.com/ByteTrue/commit-now-myfriend/internal/output"
)

type InitRuntime struct {
	CWD              string
	Env              map[string]string
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	IsTTY            bool
	SecretStore      config.WritableSecretStore
	OnboardingRunner OnboardingRunnerFunc
}

type OnboardingAnswers struct {
	Provider            config.ProviderType
	Model               string
	BaseURL             string
	PromptStyle         config.PromptStyle
	MessageLanguage     config.MessageLanguage
	StandingInstruction string
	APIKey              string
	Cancelled           bool
}

type OnboardingRunnerFunc func(OnboardingPrefill) (OnboardingAnswers, error)

type OnboardingPrefill struct {
	Provider            config.ProviderType
	Model               string
	BaseURL             string
	PromptStyle         config.PromptStyle
	MessageLanguage     config.MessageLanguage
	StandingInstruction string
}

func RunInit(args []string, runtime InitRuntime) int {
	fs := flag.NewFlagSet("cnm init", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)
	jsonMode := fs.Bool("json", false, "emit JSON output")
	dryRun := fs.Bool("dry-run", false, "preview command execution without side effects")
	providerFlag := fs.String("provider", "", "default AI provider")
	modelFlag := fs.String("model", "", "default AI model")
	baseURLFlag := fs.String("base-url", "", "OpenAI-compatible base URL")
	promptStyleFlag := fs.String("prompt-style", "", "commit prompt style")
	messageLanguageFlag := fs.String("message-language", "", "commit message language")
	standingInstructionFlag := fs.String("standing-instruction", "", "standing AI instruction")
	apiKeyFlag := fs.String("api-key", "", "API key stored in the Secret Store by default")
	plaintextAPIKeyFlag := fs.Bool("plaintext-api-key", false, "store --api-key in plaintext user config instead of the Secret Store")
	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(runtime.Stderr, "error: unexpected init arguments: %v\n", fs.Args())
		return int(output.UsageError)
	}
	apiKeyInput := strings.TrimSpace(*apiKeyFlag)
	if *plaintextAPIKeyFlag && apiKeyInput == "" {
		fmt.Fprintln(runtime.Stderr, "error: --plaintext-api-key requires --api-key.")
		return int(output.UsageError)
	}
	hasExplicitConfig := initHasExplicitConfigFlags(fs)
	interactiveMode := runtime.IsTTY && !*jsonMode && !*dryRun && !hasExplicitConfig
	if interactiveMode {
		return runInteractiveInit(runtime)
	}
	if !runtime.IsTTY && !*jsonMode && !*dryRun && !hasExplicitConfig {
		fmt.Fprintln(runtime.Stderr, "error: cnm init requires interactive TTY input or explicit flags in non-interactive environments.")
		return int(output.UsageError)
	}

	patch := config.ConfigValues{}
	if value := strings.TrimSpace(*providerFlag); value != "" {
		provider := config.ProviderType(value)
		if !config.IsProviderType(value) {
			fmt.Fprintf(runtime.Stderr, "error: Unsupported provider `%s`.\n", value)
			return int(output.Error)
		}
		patch.Provider = &provider
	}
	if value := strings.TrimSpace(*promptStyleFlag); value != "" {
		style := config.PromptStyle(value)
		if !config.IsPromptStyle(value) {
			fmt.Fprintf(runtime.Stderr, "error: Unsupported prompt style `%s`.\n", value)
			return int(output.Error)
		}
		patch.PromptStyle = &style
	}
	if value := strings.TrimSpace(*messageLanguageFlag); value != "" {
		language := config.MessageLanguage(value)
		if !config.IsMessageLanguage(value) {
			fmt.Fprintf(runtime.Stderr, "error: Unsupported message language `%s`.\n", value)
			return int(output.Error)
		}
		patch.MessageLanguage = &language
	}
	if value := strings.TrimSpace(*modelFlag); value != "" {
		patch.Model = &value
	}
	if value := strings.TrimSpace(*baseURLFlag); value != "" {
		patch.BaseURL = &value
	}
	if value := strings.TrimSpace(*standingInstructionFlag); value != "" {
		patch.StandingInstruction = &value
	}
	if value := apiKeyInput; value != "" && *plaintextAPIKeyFlag {
		patch.APIKey = &value
	}

	userConfig, err := config.LoadUserConfig(config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	merged := mergeConfigValues(userConfig, patch)
	if apiKeyInput != "" && !*plaintextAPIKeyFlag {
		merged.APIKey = nil
	}
	if merged.Provider == nil {
		provider := config.DefaultProvider
		merged.Provider = &provider
	}
	if merged.Model == nil {
		model := config.GetDefaultModel(*merged.Provider)
		merged.Model = &model
	}
	if merged.PromptStyle == nil {
		style := config.DefaultPromptStyle
		merged.PromptStyle = &style
	}
	if merged.MessageLanguage == nil {
		language := config.DefaultMessageLanguage
		merged.MessageLanguage = &language
	}

	effective := config.EffectiveConfig{
		Provider:            *merged.Provider,
		Model:               *merged.Model,
		PromptStyle:         *merged.PromptStyle,
		MessageLanguage:     *merged.MessageLanguage,
		BaseURL:             merged.BaseURL,
		StandingInstruction: merged.StandingInstruction,
		APIKey:              merged.APIKey,
		APIKeySource:        config.APIKeySourceMissing,
	}
	if merged.APIKey != nil {
		effective.APIKeySource = config.APIKeySourcePlaintextConfig
	}
	var apiKeySave *config.SaveAPIKeyResult
	if apiKeyInput != "" {
		effective.APIKey = &apiKeyInput
		if *plaintextAPIKeyFlag {
			effective.APIKeySource = config.APIKeySourcePlaintextConfig
			save := config.SaveAPIKeyResult{Provider: *merged.Provider, Source: config.APIKeySourcePlaintextConfig, Stored: true}
			apiKeySave = &save
		} else {
			effective.APIKeySource = config.APIKeySourceSecretStore
			save := config.SaveAPIKeyResult{Provider: *merged.Provider, Source: config.APIKeySourceSecretStore, Stored: false}
			apiKeySave = &save
		}
	}
	router := output.NewRouter(*jsonMode, runtime.Stdout, runtime.Stderr)
	paths := config.GetConfigPaths(config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if merged.APIKey != nil {
		_ = router.WriteHuman(plaintextAPIKeyWarning, output.StderrTarget)
	}
	if *dryRun {
		payload := map[string]any{
			"command": "cnm init",
			"config":  config.ToJSONConfigView(effective),
			"dryRun":  true,
			"ok":      true,
			"path":    paths.UserConfigPath,
			"status":  "dry_run",
		}
		if apiKeySave != nil {
			payload["apiKeySave"] = apiKeySave
		}
		_ = router.WriteStructured(payload, "Dry-run: would initialize user config at "+paths.UserConfigPath+".", output.StdoutTarget)
		return int(output.Success)
	}

	if apiKeyInput != "" && !*plaintextAPIKeyFlag {
		save, err := config.SaveAPIKeyToSecretStore(*merged.Provider, apiKeyInput, runtime.SecretStore)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v Use --plaintext-api-key to store this key in user config instead.\n", err)
			return int(output.Error)
		}
		if !save.Stored {
			message := "Secret Store is not available; use --plaintext-api-key to store this key in user config."
			if save.Warning != nil {
				message = *save.Warning
			}
			fmt.Fprintf(runtime.Stderr, "error: %s\n", message)
			return int(output.Error)
		}
		apiKeySave = &save
	}

	result, err := config.WriteUserConfig(merged, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range result.Warnings {
		_ = router.WriteHuman("Warning: "+warning, output.StderrTarget)
	}
	if router.IsJSON() {
		payload := map[string]any{
			"command": "cnm init",
			"config":  config.ToJSONConfigView(effective),
			"dryRun":  false,
			"ok":      true,
			"path":    result.Path,
			"status":  "initialized",
		}
		if apiKeySave != nil {
			payload["apiKeySave"] = apiKeySave
		}
		_ = router.WriteJSON(payload)
		return int(output.Success)
	}
	fmt.Fprintf(runtime.Stdout, "Initialized user config at %s.\n", result.Path)
	if apiKeySave != nil && apiKeySave.Source == config.APIKeySourceSecretStore {
		fmt.Fprintf(runtime.Stdout, "Stored API key in Secret Store for %s.\n", apiKeySave.Provider)
	}
	for _, line := range config.ToHumanConfigLines(effective) {
		fmt.Fprintln(runtime.Stdout, line)
	}
	return int(output.Success)
}

func initHasExplicitConfigFlags(fs *flag.FlagSet) bool {
	hasExplicit := false
	fs.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "provider", "model", "base-url", "prompt-style", "message-language", "standing-instruction", "api-key", "plaintext-api-key":
			hasExplicit = true
		}
	})
	return hasExplicit
}

func runOnboardingTUI(runtime InitRuntime, current config.EffectiveConfig) int {
	prefill := OnboardingPrefill{
		Provider:        current.Provider,
		Model:           current.Model,
		PromptStyle:     current.PromptStyle,
		MessageLanguage: current.MessageLanguage,
	}
	if current.BaseURL != nil {
		prefill.BaseURL = *current.BaseURL
	}
	if current.StandingInstruction != nil {
		prefill.StandingInstruction = *current.StandingInstruction
	}
	answers, err := runtime.OnboardingRunner(prefill)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	if answers.Cancelled {
		fmt.Fprintln(runtime.Stderr, "Onboarding cancelled. No configuration changes were saved.")
		return int(output.UserCancel)
	}
	provider := answers.Provider
	if provider == "" {
		provider = config.DefaultProvider
	}
	model := strings.TrimSpace(answers.Model)
	if model == "" {
		model = config.GetDefaultModel(provider)
	}
	style := answers.PromptStyle
	if style == "" {
		style = config.DefaultPromptStyle
	}
	language := answers.MessageLanguage
	if language == "" {
		language = config.DefaultMessageLanguage
	}
	apiKey := strings.TrimSpace(answers.APIKey)
	if apiKey == "" {
		fmt.Fprintln(runtime.Stderr, "error: API key is required")
		return int(output.Error)
	}
	save, err := config.SaveAPIKeyToSecretStore(provider, apiKey, runtime.SecretStore)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v Use --plaintext-api-key to store this key in user config instead.\n", err)
		return int(output.Error)
	}
	if !save.Stored {
		message := "Secret Store is not available; use --plaintext-api-key to store this key in user config."
		if save.Warning != nil {
			message = *save.Warning
		}
		fmt.Fprintf(runtime.Stderr, "error: %s\n", message)
		return int(output.Error)
	}
	var baseURL *string
	if provider == config.ProviderOpenAICompatible || strings.TrimSpace(answers.BaseURL) != "" {
		baseURL = optionalStringPointer(answers.BaseURL)
	}
	patch := config.ConfigValues{
		Provider:            &provider,
		Model:               &model,
		BaseURL:             baseURL,
		PromptStyle:         &style,
		MessageLanguage:     &language,
		StandingInstruction: optionalStringPointer(answers.StandingInstruction),
	}
	result, err := config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(runtime.Stderr, "Warning: "+warning)
	}
	fmt.Fprintf(runtime.Stdout, "Initialized user config at %s.\n", result.Path)
	fmt.Fprintf(runtime.Stdout, "Stored API key in Secret Store for %s.\n", save.Provider)
	effective := config.EffectiveConfig{
		Provider:            provider,
		Model:               model,
		BaseURL:             baseURL,
		PromptStyle:         style,
		MessageLanguage:     language,
		APIKey:              &apiKey,
		APIKeySource:        config.APIKeySourceSecretStore,
		StandingInstruction: optionalStringPointer(answers.StandingInstruction),
	}
	for _, line := range config.ToHumanConfigLines(effective) {
		fmt.Fprintln(runtime.Stdout, line)
	}
	return int(output.Success)
}

func runInteractiveInit(runtime InitRuntime) int {
	current, err := config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: runtime.CWD, Env: runtime.Env, SecretStore: runtime.SecretStore})
	if err != nil {
		current.Values.Provider = config.DefaultProvider
		current.Values.Model = config.GetDefaultModel(config.DefaultProvider)
		current.Values.PromptStyle = config.DefaultPromptStyle
		current.Values.MessageLanguage = config.DefaultMessageLanguage
	}
	if runtime.OnboardingRunner != nil {
		return runOnboardingTUI(runtime, current.Values)
	}
	prompter := interactive.NewPrompter(runtime.Stdin, runtime.Stdout)
	if _, err := fmt.Fprintln(runtime.Stdout, "Onboarding"); err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	provider, err := askInitProvider(prompter, current.Values.Provider)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	model, err := askInitText(prompter, "Model", firstNonEmptyString(current.Values.Model, config.GetDefaultModel(provider)), true)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	var baseURL *string
	if provider == config.ProviderOpenAICompatible {
		value, err := askInitText(prompter, "Base URL", stringPointerValue(current.Values.BaseURL), true)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		baseURL = &value
	} else {
		baseURL = current.Values.BaseURL
	}
	promptStyle, err := askInitPromptStyle(prompter, current.Values.PromptStyle)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	messageLanguage, err := askInitMessageLanguage(prompter, current.Values.MessageLanguage)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	standingInstruction, err := askInitText(prompter, "Standing Instruction", stringPointerValue(current.Values.StandingInstruction), false)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	apiKey, err := askInitText(prompter, "API key", "", true)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}

	save, err := config.SaveAPIKeyToSecretStore(provider, apiKey, runtime.SecretStore)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v Use --plaintext-api-key to store this key in user config instead.\n", err)
		return int(output.Error)
	}
	if !save.Stored {
		message := "Secret Store is not available; use --plaintext-api-key to store this key in user config."
		if save.Warning != nil {
			message = *save.Warning
		}
		fmt.Fprintf(runtime.Stderr, "error: %s\n", message)
		return int(output.Error)
	}

	patch := config.ConfigValues{
		Provider:            &provider,
		Model:               &model,
		BaseURL:             baseURL,
		PromptStyle:         &promptStyle,
		MessageLanguage:     &messageLanguage,
		StandingInstruction: optionalStringPointer(standingInstruction),
	}
	result, err := config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(runtime.Stderr, "Warning: "+warning)
	}
	fmt.Fprintf(runtime.Stdout, "Initialized user config at %s.\n", result.Path)
	fmt.Fprintf(runtime.Stdout, "Stored API key in Secret Store for %s.\n", save.Provider)
	effective := config.EffectiveConfig{Provider: provider, Model: model, BaseURL: baseURL, PromptStyle: promptStyle, MessageLanguage: messageLanguage, APIKey: &apiKey, APIKeySource: config.APIKeySourceSecretStore, StandingInstruction: optionalStringPointer(standingInstruction)}
	for _, line := range config.ToHumanConfigLines(effective) {
		fmt.Fprintln(runtime.Stdout, line)
	}
	return int(output.Success)
}

func askInitProvider(prompter *interactive.Prompter, current config.ProviderType) (config.ProviderType, error) {
	choice, err := prompter.AskChoice("Provider (openai-responses, openai-compatible, anthropic-messages, google-gemini)", providerChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.ProviderType(firstNonEmptyString(choice, string(current))), nil
}

func askInitPromptStyle(prompter *interactive.Prompter, current config.PromptStyle) (config.PromptStyle, error) {
	choice, err := prompter.AskChoice("Prompt style (auto, conventional, angular, google, atom, plain, custom)", promptStyleChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.PromptStyle(firstNonEmptyString(choice, string(current))), nil
}

func askInitMessageLanguage(prompter *interactive.Prompter, current config.MessageLanguage) (config.MessageLanguage, error) {
	choice, err := prompter.AskChoice("Message language (auto, en, zh-CN, zh-TW)", messageLanguageChoiceStrings())
	if err != nil {
		return "", err
	}
	return config.MessageLanguage(firstNonEmptyString(choice, string(current))), nil
}

func askInitText(prompter *interactive.Prompter, label string, defaultValue string, required bool) (string, error) {
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func mergeConfigValues(base config.ConfigValues, patch config.ConfigValues) config.ConfigValues {
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
