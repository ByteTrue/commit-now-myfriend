package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	configFileName        = "config.json"
	projectConfigFileName = ".cnmrc.json"
)

type ConfigEnvironment struct {
	CWD string
	Env map[string]string
}

type ResolveConfigOptions struct {
	CWD           string
	Env           map[string]string
	FlagOverrides ConfigValues
	SecretStore   SecretStore
}

type SecretStore interface {
	GetAPIKey(provider ProviderType) (*string, error)
}

type WritableSecretStore interface {
	SecretStore
	SetAPIKey(provider ProviderType, apiKey string) error
}

type SaveAPIKeyResult struct {
	Provider ProviderType `json:"provider"`
	Source   APIKeySource `json:"source"`
	Stored   bool         `json:"stored"`
	Warning  *string      `json:"warning,omitempty"`
}

type ConfigPaths struct {
	ProjectConfigPath string
	UserConfigHome    string
	UserConfigPath    string
}

type ResolvedConfig struct {
	Paths         ConfigPaths
	UserConfig    ConfigValues
	ProjectConfig ConfigValues
	EnvConfig     ConfigValues
	FlagOverrides ConfigValues
	Warnings      []string
	Values        EffectiveConfig
}

type WriteUserConfigResult struct {
	Path     string
	Stored   ConfigValues
	Warnings []string
}

type JSONConfigView struct {
	APIKey              *string `json:"apiKey"`
	APIKeySource        string  `json:"apiKeySource"`
	BaseURL             *string `json:"baseURL"`
	MessageLanguage     string  `json:"messageLanguage"`
	Model               string  `json:"model"`
	PromptStyle         string  `json:"promptStyle"`
	Provider            string  `json:"provider"`
	RecommendedModel    *string `json:"recommendedModel"`
	RecommendedProvider *string `json:"recommendedProvider"`
	StandingInstruction *string `json:"standingInstruction"`
}

type ConfigSourceSummary struct {
	Provider            ConfigValueSource `json:"provider"`
	Model               ConfigValueSource `json:"model"`
	APIKey              ConfigValueSource `json:"apiKey"`
	PromptStyle         ConfigValueSource `json:"promptStyle"`
	MessageLanguage     ConfigValueSource `json:"messageLanguage"`
	StandingInstruction string            `json:"standingInstruction"`
}

var ConfigFileNames = struct {
	Project string
	User    string
}{
	Project: projectConfigFileName,
	User:    configFileName,
}

func GetUserConfigHome(env map[string]string) string {
	if configured := strings.TrimSpace(getEnv(env, "CNM_HOME")); configured != "" {
		return resolvePath(configured)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return resolvePath(".cnm")
	}

	return filepath.Join(home, ".cnm")
}

func GetConfigPaths(options ConfigEnvironment) ConfigPaths {
	cwd := runtimeCWD(options.CWD)
	userConfigHome := GetUserConfigHome(options.Env)

	return ConfigPaths{
		ProjectConfigPath: filepath.Join(cwd, projectConfigFileName),
		UserConfigHome:    userConfigHome,
		UserConfigPath:    filepath.Join(userConfigHome, configFileName),
	}
}

func LoadUserConfig(options ConfigEnvironment) (ConfigValues, error) {
	paths := GetConfigPaths(options)
	config, err := readOptionalConfigFile(paths.UserConfigPath, "User config")
	if err != nil {
		return ConfigValues{}, err
	}
	if config == nil {
		return ConfigValues{}, nil
	}
	return *config, nil
}

func LoadProjectConfig(options ConfigEnvironment) (ConfigValues, []string, error) {
	paths := GetConfigPaths(options)
	projectConfig, err := readOptionalConfigFile(paths.ProjectConfigPath, "Project config")
	if err != nil {
		return ConfigValues{}, nil, err
	}
	if projectConfig == nil {
		return ConfigValues{}, nil, nil
	}

	sanitized := *projectConfig
	warnings := []string{}
	if sanitized.Provider != nil {
		if sanitized.RecommendedProvider == nil {
			sanitized.RecommendedProvider = sanitized.Provider
		}
		warnings = append(warnings, fmt.Sprintf("Project config at %s contains private `provider`; use `recommendedProvider` for team suggestions.", paths.ProjectConfigPath))
		sanitized.Provider = nil
	}
	if sanitized.Model != nil {
		if sanitized.RecommendedModel == nil {
			sanitized.RecommendedModel = sanitized.Model
		}
		warnings = append(warnings, fmt.Sprintf("Project config at %s contains private `model`; use `recommendedModel` for team suggestions.", paths.ProjectConfigPath))
		sanitized.Model = nil
	}
	if sanitized.BaseURL != nil {
		warnings = append(warnings, fmt.Sprintf("Project config at %s contains private `baseURL`; project-level provider endpoints are ignored.", paths.ProjectConfigPath))
		sanitized.BaseURL = nil
	}
	if sanitized.APIKey != nil {
		warnings = append(warnings, fmt.Sprintf("Project config at %s contains `apiKey`; project-level secrets are ignored.", paths.ProjectConfigPath))
		sanitized.APIKey = nil
	}

	return sanitized, warnings, nil
}

func GetEnvConfig(env map[string]string) (ConfigValues, error) {
	config := ConfigValues{}

	if provider := strings.TrimSpace(getEnv(env, "CNM_PROVIDER")); provider != "" {
		if !IsProviderType(provider) {
			return ConfigValues{}, newError("Environment variable `CNM_PROVIDER` has unsupported value `%s`.", provider)
		}
		providerType := ProviderType(provider)
		config.Provider = &providerType
	}

	if model := strings.TrimSpace(getEnv(env, "CNM_MODEL")); model != "" {
		config.Model = stringPtr(model)
	}

	if baseURL := strings.TrimSpace(getEnv(env, "CNM_BASE_URL")); baseURL != "" {
		config.BaseURL = stringPtr(baseURL)
	}

	if promptStyle := strings.TrimSpace(getEnv(env, "CNM_PROMPT_STYLE")); promptStyle != "" {
		if !IsPromptStyle(promptStyle) {
			return ConfigValues{}, newError("Environment variable `CNM_PROMPT_STYLE` has unsupported value `%s`.", promptStyle)
		}
		style := PromptStyle(promptStyle)
		config.PromptStyle = &style
	}

	if messageLanguage := strings.TrimSpace(getEnv(env, "CNM_MESSAGE_LANGUAGE")); messageLanguage != "" {
		if !IsMessageLanguage(messageLanguage) {
			return ConfigValues{}, newError("Environment variable `CNM_MESSAGE_LANGUAGE` has unsupported value `%s`.", messageLanguage)
		}
		language := MessageLanguage(messageLanguage)
		config.MessageLanguage = &language
	}

	if standingInstruction := strings.TrimSpace(getEnv(env, "CNM_STANDING_INSTRUCTION")); standingInstruction != "" {
		config.StandingInstruction = stringPtr(standingInstruction)
	}

	if apiKey := strings.TrimSpace(getEnv(env, "CNM_API_KEY")); apiKey != "" {
		config.APIKey = stringPtr(apiKey)
	}

	return config, nil
}

func ResolveEffectiveConfig(options ResolveConfigOptions) (ResolvedConfig, error) {
	paths := GetConfigPaths(ConfigEnvironment{CWD: options.CWD, Env: options.Env})
	userConfig, err := LoadUserConfig(ConfigEnvironment{CWD: options.CWD, Env: options.Env})
	if err != nil {
		return ResolvedConfig{}, err
	}
	projectConfig, warnings, err := LoadProjectConfig(ConfigEnvironment{CWD: options.CWD, Env: options.Env})
	if err != nil {
		return ResolvedConfig{}, err
	}
	envConfig, err := GetEnvConfig(options.Env)
	if err != nil {
		return ResolvedConfig{}, err
	}
	flagOverrides := normalizeConfig(options.FlagOverrides)
	merged := mergeConfigs(userConfig, projectConfig, envConfig, flagOverrides)
	provider := effectiveProvider(merged)
	promptStyle := effectivePromptStyle(merged)
	messageLanguage := effectiveMessageLanguage(merged)
	model := GetDefaultModel(provider)
	if merged.Model != nil {
		model = *merged.Model
	}
	apiKey, apiKeySource, err := resolveAPIKey(provider, userConfig, envConfig, flagOverrides, options.SecretStore)
	if err != nil {
		return ResolvedConfig{}, err
	}
	standingInstruction := combineStandingInstructions(projectConfig.StandingInstruction, userConfig.StandingInstruction, envConfig.StandingInstruction, flagOverrides.StandingInstruction)

	return ResolvedConfig{
		Paths:         paths,
		UserConfig:    userConfig,
		ProjectConfig: projectConfig,
		EnvConfig:     envConfig,
		FlagOverrides: flagOverrides,
		Warnings:      warnings,
		Values: EffectiveConfig{
			APIKey:              apiKey,
			APIKeySource:        apiKeySource,
			BaseURL:             merged.BaseURL,
			MessageLanguage:     messageLanguage,
			Model:               model,
			PromptStyle:         promptStyle,
			Provider:            provider,
			RecommendedModel:    merged.RecommendedModel,
			RecommendedProvider: merged.RecommendedProvider,
			StandingInstruction: standingInstruction,
		},
	}, nil
}

func ParseKeyValue(key string, rawValue string) (ConfigValues, error) {
	if !IsConfigKey(key) {
		return ConfigValues{}, newError("Unsupported config key `%s`.", key)
	}

	value, err := ensureNonEmptyString(rawValue, ConfigKey(key), "CLI input")
	if err != nil {
		return ConfigValues{}, err
	}

	switch ConfigKey(key) {
	case ConfigKeyProvider:
		if !IsProviderType(value) {
			return ConfigValues{}, newError("Unsupported provider `%s`.", value)
		}
		provider := ProviderType(value)
		return ConfigValues{Provider: &provider}, nil
	case ConfigKeyPromptStyle:
		if !IsPromptStyle(value) {
			return ConfigValues{}, newError("Unsupported prompt style `%s`.", value)
		}
		promptStyle := PromptStyle(value)
		return ConfigValues{PromptStyle: &promptStyle}, nil
	case ConfigKeyMessageLanguage:
		if !IsMessageLanguage(value) {
			return ConfigValues{}, newError("Unsupported message language `%s`.", value)
		}
		messageLanguage := MessageLanguage(value)
		return ConfigValues{MessageLanguage: &messageLanguage}, nil
	case ConfigKeyStandingInstruction:
		return ConfigValues{StandingInstruction: &value}, nil
	case ConfigKeyRecommendedProvider:
		if !IsProviderType(value) {
			return ConfigValues{}, newError("Unsupported provider recommendation `%s`.", value)
		}
		provider := ProviderType(value)
		return ConfigValues{RecommendedProvider: &provider}, nil
	case ConfigKeyRecommendedModel:
		return ConfigValues{RecommendedModel: &value}, nil
	case ConfigKeyAPIKey:
		return ConfigValues{APIKey: &value}, nil
	case ConfigKeyBaseURL:
		return ConfigValues{BaseURL: &value}, nil
	case ConfigKeyModel:
		return ConfigValues{Model: &value}, nil
	default:
		return ConfigValues{}, newError("Unsupported config key `%s`.", key)
	}
}

func WriteUserConfig(config ConfigValues, options ConfigEnvironment) (WriteUserConfigResult, error) {
	paths := GetConfigPaths(options)
	if err := os.MkdirAll(paths.UserConfigHome, 0o755); err != nil {
		return WriteUserConfigResult{}, newError("Unable to create user config directory at %s.", paths.UserConfigHome)
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return WriteUserConfigResult{}, err
	}
	content = append(content, '\n')

	if err := os.WriteFile(paths.UserConfigPath, content, 0o600); err != nil {
		return WriteUserConfigResult{}, newError("Unable to write user config at %s.", paths.UserConfigPath)
	}

	warnings := []string{}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(paths.UserConfigPath, 0o600); err != nil {
			warnings = append(warnings, fmt.Sprintf("Unable to set 0600 permissions on user config at %s.", paths.UserConfigPath))
		}
	}

	return WriteUserConfigResult{Path: paths.UserConfigPath, Stored: config, Warnings: warnings}, nil
}

func WriteUserConfigPatch(patch ConfigValues, options ConfigEnvironment) (WriteUserConfigResult, error) {
	current, err := LoadUserConfig(options)
	if err != nil {
		return WriteUserConfigResult{}, err
	}
	return WriteUserConfig(mergeConfigs(current, patch), options)
}

func SaveAPIKeyToSecretStore(provider ProviderType, apiKey string, store WritableSecretStore) (SaveAPIKeyResult, error) {
	if store == nil {
		warning := "Secret Store is not available; use --plaintext-api-key to store this key in user config."
		return SaveAPIKeyResult{Provider: provider, Source: APIKeySourceMissing, Stored: false, Warning: &warning}, nil
	}
	if err := store.SetAPIKey(provider, apiKey); err != nil {
		return SaveAPIKeyResult{}, newError("Unable to write API key to Secret Store for %s.", provider)
	}
	return SaveAPIKeyResult{Provider: provider, Source: APIKeySourceSecretStore, Stored: true}, nil
}

func UnsetUserConfigKey(key ConfigKey, options ConfigEnvironment) (WriteUserConfigResult, error) {
	current, err := LoadUserConfig(options)
	if err != nil {
		return WriteUserConfigResult{}, err
	}

	next := current
	switch key {
	case ConfigKeyProvider:
		next.Provider = nil
	case ConfigKeyModel:
		next.Model = nil
	case ConfigKeyBaseURL:
		next.BaseURL = nil
	case ConfigKeyPromptStyle:
		next.PromptStyle = nil
	case ConfigKeyMessageLanguage:
		next.MessageLanguage = nil
	case ConfigKeyStandingInstruction:
		next.StandingInstruction = nil
	case ConfigKeyRecommendedProvider:
		next.RecommendedProvider = nil
	case ConfigKeyRecommendedModel:
		next.RecommendedModel = nil
	case ConfigKeyAPIKey:
		next.APIKey = nil
	default:
		return WriteUserConfigResult{}, newError("Unsupported config key `%s`.", key)
	}

	return WriteUserConfig(next, options)
}

func RedactSecret(secret *string) *string {
	if secret == nil || strings.TrimSpace(*secret) == "" {
		return nil
	}
	return stringPtr("[redacted]")
}

func RedactConfigValues(config ConfigValues) ConfigValues {
	redacted := config
	redacted.APIKey = RedactSecret(config.APIKey)
	return redacted
}

func RedactEffectiveConfig(config EffectiveConfig) EffectiveConfig {
	redacted := config
	redacted.APIKey = RedactSecret(config.APIKey)
	return redacted
}

func ToJSONConfigView(config EffectiveConfig) JSONConfigView {
	var recommendedProvider *string
	if config.RecommendedProvider != nil {
		recommendedProvider = stringPtr(string(*config.RecommendedProvider))
	}
	return JSONConfigView{
		APIKey:              RedactSecret(config.APIKey),
		APIKeySource:        string(config.APIKeySource),
		BaseURL:             config.BaseURL,
		MessageLanguage:     string(config.MessageLanguage),
		Model:               config.Model,
		PromptStyle:         string(config.PromptStyle),
		Provider:            string(config.Provider),
		RecommendedModel:    config.RecommendedModel,
		RecommendedProvider: recommendedProvider,
		StandingInstruction: config.StandingInstruction,
	}
}

func SummarizeConfigSources(resolved ResolvedConfig) ConfigSourceSummary {
	return ConfigSourceSummary{
		Provider:            sourceForSingleValue(resolved.FlagOverrides.Provider, resolved.EnvConfig.Provider, resolved.ProjectConfig.Provider, resolved.UserConfig.Provider),
		Model:               sourceForSingleValue(resolved.FlagOverrides.Model, resolved.EnvConfig.Model, resolved.ProjectConfig.Model, resolved.UserConfig.Model),
		APIKey:              sourceForAPIKey(resolved.Values.APIKeySource),
		PromptStyle:         sourceForSingleValue(resolved.FlagOverrides.PromptStyle, resolved.EnvConfig.PromptStyle, resolved.ProjectConfig.PromptStyle, resolved.UserConfig.PromptStyle),
		MessageLanguage:     sourceForSingleValue(resolved.FlagOverrides.MessageLanguage, resolved.EnvConfig.MessageLanguage, resolved.ProjectConfig.MessageLanguage, resolved.UserConfig.MessageLanguage),
		StandingInstruction: sourceListForStandingInstruction(resolved.ProjectConfig.StandingInstruction, resolved.UserConfig.StandingInstruction, resolved.EnvConfig.StandingInstruction, resolved.FlagOverrides.StandingInstruction),
	}
}

func ToHumanConfigLines(config EffectiveConfig) []string {
	view := ToJSONConfigView(config)
	return []string{
		renderConfigLine(string(ConfigKeyProvider), stringPtr(view.Provider)),
		renderConfigLine(string(ConfigKeyModel), stringPtr(view.Model)),
		renderConfigLine(string(ConfigKeyBaseURL), view.BaseURL),
		renderConfigLine(string(ConfigKeyPromptStyle), stringPtr(view.PromptStyle)),
		renderConfigLine(string(ConfigKeyMessageLanguage), stringPtr(view.MessageLanguage)),
		renderConfigLine(string(ConfigKeyStandingInstruction), view.StandingInstruction),
		renderConfigLine(string(ConfigKeyRecommendedProvider), view.RecommendedProvider),
		renderConfigLine(string(ConfigKeyRecommendedModel), view.RecommendedModel),
		renderConfigLine("apiKeySource", stringPtr(view.APIKeySource)),
		renderConfigLine(string(ConfigKeyAPIKey), view.APIKey),
	}
}

func GetConfigValue(config EffectiveConfig, key ConfigKey) *string {
	switch key {
	case ConfigKeyProvider:
		return stringPtr(string(config.Provider))
	case ConfigKeyModel:
		return stringPtr(config.Model)
	case ConfigKeyBaseURL:
		return config.BaseURL
	case ConfigKeyPromptStyle:
		return stringPtr(string(config.PromptStyle))
	case ConfigKeyMessageLanguage:
		return stringPtr(string(config.MessageLanguage))
	case ConfigKeyStandingInstruction:
		return config.StandingInstruction
	case ConfigKeyRecommendedProvider:
		if config.RecommendedProvider == nil {
			return nil
		}
		return stringPtr(string(*config.RecommendedProvider))
	case ConfigKeyRecommendedModel:
		return config.RecommendedModel
	case ConfigKeyAPIKey:
		return RedactSecret(config.APIKey)
	default:
		return nil
	}
}

func AssertConfigKey(key string) (ConfigKey, error) {
	if !IsConfigKey(key) {
		return "", newError("Unsupported config key `%s`.", key)
	}
	return ConfigKey(key), nil
}

func readOptionalConfigFile(path string, sourceLabel string) (*ConfigValues, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, newError("Unable to read %s at %s.", sourceLabel, path)
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, newError("%s at %s is not valid JSON.", sourceLabel, path)
	}

	parsed, err := parseConfigObject(payload, fmt.Sprintf("%s at %s", sourceLabel, path))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseConfigObject(raw map[string]any, sourceLabel string) (ConfigValues, error) {
	parsed := ConfigValues{}
	for key := range raw {
		if !IsConfigKey(key) {
			return ConfigValues{}, newError("Unsupported config key `%s` in %s.", key, sourceLabel)
		}
	}

	if rawProvider, ok := raw[string(ConfigKeyProvider)]; ok {
		provider, err := ensureNonEmptyString(rawProvider, ConfigKeyProvider, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		if !IsProviderType(provider) {
			return ConfigValues{}, newError("%s has unsupported `provider` value `%s`.", sourceLabel, provider)
		}
		providerType := ProviderType(provider)
		parsed.Provider = &providerType
	}

	if rawModel, ok := raw[string(ConfigKeyModel)]; ok {
		model, err := ensureNonEmptyString(rawModel, ConfigKeyModel, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		parsed.Model = &model
	}

	if rawBaseURL, ok := raw[string(ConfigKeyBaseURL)]; ok {
		baseURL, err := ensureNonEmptyString(rawBaseURL, ConfigKeyBaseURL, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		parsed.BaseURL = &baseURL
	}

	if rawPromptStyle, ok := raw[string(ConfigKeyPromptStyle)]; ok {
		promptStyle, err := ensureNonEmptyString(rawPromptStyle, ConfigKeyPromptStyle, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		if !IsPromptStyle(promptStyle) {
			return ConfigValues{}, newError("%s has unsupported `promptStyle` value `%s`.", sourceLabel, promptStyle)
		}
		style := PromptStyle(promptStyle)
		parsed.PromptStyle = &style
	}

	if rawMessageLanguage, ok := raw[string(ConfigKeyMessageLanguage)]; ok {
		messageLanguage, err := ensureNonEmptyString(rawMessageLanguage, ConfigKeyMessageLanguage, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		if !IsMessageLanguage(messageLanguage) {
			return ConfigValues{}, newError("%s has unsupported `messageLanguage` value `%s`.", sourceLabel, messageLanguage)
		}
		language := MessageLanguage(messageLanguage)
		parsed.MessageLanguage = &language
	}

	if rawStandingInstruction, ok := raw[string(ConfigKeyStandingInstruction)]; ok {
		standingInstruction, err := ensureNonEmptyString(rawStandingInstruction, ConfigKeyStandingInstruction, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		parsed.StandingInstruction = &standingInstruction
	}

	if rawRecommendedProvider, ok := raw[string(ConfigKeyRecommendedProvider)]; ok {
		recommendedProvider, err := ensureNonEmptyString(rawRecommendedProvider, ConfigKeyRecommendedProvider, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		if !IsProviderType(recommendedProvider) {
			return ConfigValues{}, newError("%s has unsupported `recommendedProvider` value `%s`.", sourceLabel, recommendedProvider)
		}
		provider := ProviderType(recommendedProvider)
		parsed.RecommendedProvider = &provider
	}

	if rawRecommendedModel, ok := raw[string(ConfigKeyRecommendedModel)]; ok {
		recommendedModel, err := ensureNonEmptyString(rawRecommendedModel, ConfigKeyRecommendedModel, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		parsed.RecommendedModel = &recommendedModel
	}

	if rawAPIKey, ok := raw[string(ConfigKeyAPIKey)]; ok {
		apiKey, err := ensureNonEmptyString(rawAPIKey, ConfigKeyAPIKey, sourceLabel)
		if err != nil {
			return ConfigValues{}, err
		}
		parsed.APIKey = &apiKey
	}

	return parsed, nil
}

func ensureNonEmptyString(value any, key ConfigKey, sourceLabel string) (string, error) {
	stringValue, ok := value.(string)
	if !ok {
		return "", newError("%s has an invalid `%s` value; expected a string.", sourceLabel, key)
	}
	trimmed := strings.TrimSpace(stringValue)
	if trimmed == "" {
		return "", newError("%s has an empty `%s` value.", sourceLabel, key)
	}
	return trimmed, nil
}

func normalizeConfig(config ConfigValues) ConfigValues {
	result := ConfigValues{}
	if config.Provider != nil {
		result.Provider = config.Provider
	}
	if config.Model != nil {
		result.Model = config.Model
	}
	if config.BaseURL != nil {
		result.BaseURL = config.BaseURL
	}
	if config.PromptStyle != nil {
		result.PromptStyle = config.PromptStyle
	}
	if config.MessageLanguage != nil {
		result.MessageLanguage = config.MessageLanguage
	}
	if config.StandingInstruction != nil {
		result.StandingInstruction = config.StandingInstruction
	}
	if config.RecommendedProvider != nil {
		result.RecommendedProvider = config.RecommendedProvider
	}
	if config.RecommendedModel != nil {
		result.RecommendedModel = config.RecommendedModel
	}
	if config.APIKey != nil {
		result.APIKey = config.APIKey
	}
	return result
}

func mergeConfigs(configs ...ConfigValues) ConfigValues {
	result := ConfigValues{}
	for _, config := range configs {
		if config.Provider != nil {
			result.Provider = config.Provider
		}
		if config.Model != nil {
			result.Model = config.Model
		}
		if config.BaseURL != nil {
			result.BaseURL = config.BaseURL
		}
		if config.PromptStyle != nil {
			result.PromptStyle = config.PromptStyle
		}
		if config.MessageLanguage != nil {
			result.MessageLanguage = config.MessageLanguage
		}
		if config.StandingInstruction != nil {
			result.StandingInstruction = config.StandingInstruction
		}
		if config.RecommendedProvider != nil {
			result.RecommendedProvider = config.RecommendedProvider
		}
		if config.RecommendedModel != nil {
			result.RecommendedModel = config.RecommendedModel
		}
		if config.APIKey != nil {
			result.APIKey = config.APIKey
		}
	}
	return result
}

func effectiveProvider(config ConfigValues) ProviderType {
	if config.Provider != nil {
		return *config.Provider
	}
	return DefaultProvider
}

func effectivePromptStyle(config ConfigValues) PromptStyle {
	if config.PromptStyle != nil {
		return *config.PromptStyle
	}
	return DefaultPromptStyle
}

func effectiveMessageLanguage(config ConfigValues) MessageLanguage {
	if config.MessageLanguage != nil {
		return *config.MessageLanguage
	}
	return DefaultMessageLanguage
}

func resolveAPIKey(provider ProviderType, userConfig ConfigValues, envConfig ConfigValues, flagOverrides ConfigValues, store SecretStore) (*string, APIKeySource, error) {
	if flagOverrides.APIKey != nil {
		return flagOverrides.APIKey, APIKeySourcePlaintextConfig, nil
	}
	if envConfig.APIKey != nil {
		return envConfig.APIKey, APIKeySourceEnv, nil
	}
	if store != nil {
		apiKey, err := store.GetAPIKey(provider)
		if err != nil {
			return nil, APIKeySourceMissing, newError("Unable to read API key from Secret Store for %s.", provider)
		}
		if apiKey != nil && strings.TrimSpace(*apiKey) != "" {
			return apiKey, APIKeySourceSecretStore, nil
		}
	}
	if userConfig.APIKey != nil {
		return userConfig.APIKey, APIKeySourcePlaintextConfig, nil
	}
	return nil, APIKeySourceMissing, nil
}

func sourceForSingleValue(flag any, env any, project any, user any) ConfigValueSource {
	if !isNilValue(flag) {
		return ConfigValueSourceFlag
	}
	if !isNilValue(env) {
		return ConfigValueSourceEnv
	}
	if !isNilValue(project) {
		return ConfigValueSourceProject
	}
	if !isNilValue(user) {
		return ConfigValueSourceUser
	}
	return ConfigValueSourceDefault
}

func sourceForAPIKey(source APIKeySource) ConfigValueSource {
	switch source {
	case APIKeySourceEnv:
		return ConfigValueSourceEnv
	case APIKeySourceSecretStore:
		return ConfigValueSourceSecretStore
	case APIKeySourcePlaintextConfig:
		return ConfigValueSourceUser
	default:
		return ConfigValueSourceMissing
	}
}

func sourceListForStandingInstruction(project *string, user *string, env *string, flag *string) string {
	sources := make([]string, 0, 4)
	if project != nil {
		sources = append(sources, string(ConfigValueSourceProject))
	}
	if user != nil {
		sources = append(sources, string(ConfigValueSourceUser))
	}
	if env != nil {
		sources = append(sources, string(ConfigValueSourceEnv))
	}
	if flag != nil {
		sources = append(sources, string(ConfigValueSourceFlag))
	}
	if len(sources) == 0 {
		return string(ConfigValueSourceDefault)
	}
	return strings.Join(sources, ",")
}

func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case *ProviderType:
		return typed == nil
	case *PromptStyle:
		return typed == nil
	case *MessageLanguage:
		return typed == nil
	case *string:
		return typed == nil
	default:
		return false
	}
}

func combineStandingInstructions(values ...*string) *string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return stringPtr(strings.Join(parts, "\n\n"))
}

func getEnv(env map[string]string, key string) string {
	if env != nil {
		if value, ok := env[key]; ok {
			return value
		}
	}
	return os.Getenv(key)
}

func runtimeCWD(cwd string) string {
	if strings.TrimSpace(cwd) != "" {
		return cwd
	}
	resolved, err := os.Getwd()
	if err != nil {
		return "."
	}
	return resolved
}

func resolvePath(candidate string) string {
	if filepath.IsAbs(candidate) {
		return candidate
	}
	resolved, err := filepath.Abs(candidate)
	if err != nil {
		return candidate
	}
	return resolved
}

func renderConfigLine(key string, value *string) string {
	if value == nil {
		return fmt.Sprintf("%s=(unset)", key)
	}
	return fmt.Sprintf("%s=%s", key, *value)
}

func stringPtr(value string) *string {
	v := value
	return &v
}
