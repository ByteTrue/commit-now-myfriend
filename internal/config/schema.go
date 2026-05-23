package config

type ProviderType string

type PromptStyle string

type MessageLanguage string

type APIKeySource string

type ConfigValueSource string

type ConfigKey string

const (
	ProviderOpenAICompatible ProviderType = "openai-compatible"
	ProviderOpenAIResponses  ProviderType = "openai-responses"
	ProviderAnthropic        ProviderType = "anthropic-messages"
	ProviderGoogleGemini     ProviderType = "google-gemini"
)

const (
	PromptStyleAuto         PromptStyle = "auto"
	PromptStyleConventional PromptStyle = "conventional"
	PromptStyleAngular      PromptStyle = "angular"
	PromptStyleGoogle       PromptStyle = "google"
	PromptStyleAtom         PromptStyle = "atom"
	PromptStylePlain        PromptStyle = "plain"
	PromptStyleCustom       PromptStyle = "custom"
)

const (
	MessageLanguageAuto               MessageLanguage = "auto"
	MessageLanguageEnglish            MessageLanguage = "en"
	MessageLanguageSimplifiedChinese  MessageLanguage = "zh-CN"
	MessageLanguageTraditionalChinese MessageLanguage = "zh-TW"
)

const (
	APIKeySourceMissing         APIKeySource = "missing"
	APIKeySourceEnv             APIKeySource = "env"
	APIKeySourceSecretStore     APIKeySource = "secret_store"
	APIKeySourcePlaintextConfig APIKeySource = "plaintext_config"
)

const (
	ConfigValueSourceDefault     ConfigValueSource = "default"
	ConfigValueSourceProject     ConfigValueSource = "project_config"
	ConfigValueSourceUser        ConfigValueSource = "user_config"
	ConfigValueSourceEnv         ConfigValueSource = "env"
	ConfigValueSourceFlag        ConfigValueSource = "flag"
	ConfigValueSourceSecretStore ConfigValueSource = "secret_store"
	ConfigValueSourceMissing     ConfigValueSource = "missing"
)

const (
	ConfigKeyProvider            ConfigKey = "provider"
	ConfigKeyModel               ConfigKey = "model"
	ConfigKeyBaseURL             ConfigKey = "baseURL"
	ConfigKeyPromptStyle         ConfigKey = "promptStyle"
	ConfigKeyMessageLanguage     ConfigKey = "messageLanguage"
	ConfigKeyStandingInstruction ConfigKey = "standingInstruction"
	ConfigKeyRecommendedProvider ConfigKey = "recommendedProvider"
	ConfigKeyRecommendedModel    ConfigKey = "recommendedModel"
	ConfigKeyAPIKey              ConfigKey = "apiKey"
)

var ProviderTypes = []ProviderType{
	ProviderOpenAICompatible,
	ProviderOpenAIResponses,
	ProviderAnthropic,
	ProviderGoogleGemini,
}

var PromptStyles = []PromptStyle{
	PromptStyleAuto,
	PromptStyleConventional,
	PromptStyleAngular,
	PromptStyleGoogle,
	PromptStyleAtom,
	PromptStylePlain,
	PromptStyleCustom,
}

var MessageLanguages = []MessageLanguage{
	MessageLanguageAuto,
	MessageLanguageEnglish,
	MessageLanguageSimplifiedChinese,
	MessageLanguageTraditionalChinese,
}

var ConfigKeys = []ConfigKey{
	ConfigKeyProvider,
	ConfigKeyModel,
	ConfigKeyBaseURL,
	ConfigKeyPromptStyle,
	ConfigKeyMessageLanguage,
	ConfigKeyStandingInstruction,
	ConfigKeyRecommendedProvider,
	ConfigKeyRecommendedModel,
	ConfigKeyAPIKey,
}

type ConfigValues struct {
	Provider            *ProviderType    `json:"provider,omitempty"`
	Model               *string          `json:"model,omitempty"`
	BaseURL             *string          `json:"baseURL,omitempty"`
	PromptStyle         *PromptStyle     `json:"promptStyle,omitempty"`
	MessageLanguage     *MessageLanguage `json:"messageLanguage,omitempty"`
	StandingInstruction *string          `json:"standingInstruction,omitempty"`
	RecommendedProvider *ProviderType    `json:"recommendedProvider,omitempty"`
	RecommendedModel    *string          `json:"recommendedModel,omitempty"`
	APIKey              *string          `json:"apiKey,omitempty"`
}

type EffectiveConfig struct {
	Provider            ProviderType    `json:"provider"`
	Model               string          `json:"model"`
	BaseURL             *string         `json:"baseURL,omitempty"`
	PromptStyle         PromptStyle     `json:"promptStyle"`
	MessageLanguage     MessageLanguage `json:"messageLanguage"`
	StandingInstruction *string         `json:"standingInstruction,omitempty"`
	RecommendedProvider *ProviderType   `json:"recommendedProvider,omitempty"`
	RecommendedModel    *string         `json:"recommendedModel,omitempty"`
	APIKey              *string         `json:"apiKey,omitempty"`
	APIKeySource        APIKeySource    `json:"apiKeySource"`
}

const (
	DefaultProvider        ProviderType    = ProviderOpenAIResponses
	DefaultPromptStyle     PromptStyle     = PromptStyleAuto
	DefaultMessageLanguage MessageLanguage = MessageLanguageAuto
)

var defaultModels = map[ProviderType]string{
	ProviderAnthropic:        "claude-sonnet-4-20250514",
	ProviderGoogleGemini:     "gemini-2.5-flash",
	ProviderOpenAICompatible: "gpt-5-mini",
	ProviderOpenAIResponses:  "gpt-5-mini",
}

func GetDefaultModel(provider ProviderType) string {
	if model, ok := defaultModels[provider]; ok {
		return model
	}

	return defaultModels[DefaultProvider]
}

func IsConfigKey(value string) bool {
	for _, key := range ConfigKeys {
		if string(key) == value {
			return true
		}
	}

	return false
}

func IsProviderType(value string) bool {
	for _, provider := range ProviderTypes {
		if string(provider) == value {
			return true
		}
	}

	return false
}

func IsPromptStyle(value string) bool {
	for _, style := range PromptStyles {
		if string(style) == value {
			return true
		}
	}

	return false
}

func IsMessageLanguage(value string) bool {
	for _, language := range MessageLanguages {
		if string(language) == value {
			return true
		}
	}

	return false
}
