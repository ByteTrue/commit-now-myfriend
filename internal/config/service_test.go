package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fakeSecretStore struct {
	keys map[ProviderType]string
}

func (s fakeSecretStore) GetAPIKey(provider ProviderType) (*string, error) {
	if value, ok := s.keys[provider]; ok {
		return &value, nil
	}
	return nil, nil
}

type fakeWritableSecretStore struct {
	keys map[ProviderType]string
}

func (s *fakeWritableSecretStore) GetAPIKey(provider ProviderType) (*string, error) {
	if value, ok := s.keys[provider]; ok {
		return &value, nil
	}
	return nil, nil
}

func (s *fakeWritableSecretStore) SetAPIKey(provider ProviderType, apiKey string) error {
	if s.keys == nil {
		s.keys = map[ProviderType]string{}
	}
	s.keys[provider] = apiKey
	return nil
}

func TestResolveEffectiveConfigPrecedence(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	env := map[string]string{
		"CNM_HOME":                 home,
		"CNM_PROVIDER":             "google-gemini",
		"CNM_MODEL":                "env-model",
		"CNM_BASE_URL":             "https://env.example/v1",
		"CNM_PROMPT_STYLE":         "plain",
		"CNM_STANDING_INSTRUCTION": "env instruction",
		"CNM_API_KEY":              "env-key",
	}

	userProvider := ProviderOpenAIResponses
	userPrompt := PromptStyleAuto
	userModel := "user-model"
	userBaseURL := "https://user.example/v1"
	userStandingInstruction := "user instruction"
	userAPIKey := "user-key"
	if _, err := WriteUserConfig(ConfigValues{
		Provider:            &userProvider,
		Model:               &userModel,
		BaseURL:             &userBaseURL,
		PromptStyle:         &userPrompt,
		StandingInstruction: &userStandingInstruction,
		APIKey:              &userAPIKey,
	}, ConfigEnvironment{CWD: cwd, Env: env}); err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(cwd, ".cnmrc.json"), []byte("{\n  \"provider\": \"anthropic-messages\",\n  \"model\": \"project-model\",\n  \"baseURL\": \"https://project.example/v1\",\n  \"promptStyle\": \"google\",\n  \"standingInstruction\": \"project instruction\",\n  \"apiKey\": \"project-key\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	flagProvider := ProviderOpenAICompatible
	flagPromptStyle := PromptStyleCustom
	flagModel := "flag-model"
	flagBaseURL := "https://flag.example/v1"
	flagStandingInstruction := "flag instruction"
	resolved, err := ResolveEffectiveConfig(ResolveConfigOptions{
		CWD: cwd,
		Env: env,
		FlagOverrides: ConfigValues{
			Provider:            &flagProvider,
			Model:               &flagModel,
			BaseURL:             &flagBaseURL,
			PromptStyle:         &flagPromptStyle,
			StandingInstruction: &flagStandingInstruction,
		},
	})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig error: %v", err)
	}

	if resolved.Values.Provider != flagProvider || resolved.Values.Model != flagModel || resolved.Values.PromptStyle != flagPromptStyle {
		t.Fatalf("flag overrides did not win: %+v", resolved.Values)
	}
	if resolved.Values.BaseURL == nil || *resolved.Values.BaseURL != flagBaseURL {
		t.Fatalf("expected flag baseURL, got %+v", resolved.Values.BaseURL)
	}
	if resolved.Values.StandingInstruction == nil || !strings.Contains(*resolved.Values.StandingInstruction, flagStandingInstruction) {
		t.Fatalf("expected flag standing instruction, got %+v", resolved.Values.StandingInstruction)
	}
	if resolved.Values.APIKey == nil || *resolved.Values.APIKey != env["CNM_API_KEY"] {
		t.Fatalf("expected env api key to win, got %+v", resolved.Values.APIKey)
	}
	if len(resolved.Warnings) == 0 {
		t.Fatalf("expected project config warnings, got %v", resolved.Warnings)
	}
}

func TestCustomPromptIsNotAConfigKeyOrEnvironmentAlias(t *testing.T) {
	if IsConfigKey("customPrompt") {
		t.Fatal("customPrompt should not be part of the redesigned config key surface")
	}
	if _, err := AssertConfigKey("customPrompt"); err == nil {
		t.Fatal("expected customPrompt config key to be rejected")
	}
	if _, err := ParseKeyValue("customPrompt", "legacy prompt"); err == nil {
		t.Fatal("expected customPrompt config set to be rejected")
	}

	envConfig, err := GetEnvConfig(map[string]string{"CNM_CUSTOM_PROMPT": "legacy prompt"})
	if err != nil {
		t.Fatalf("GetEnvConfig error: %v", err)
	}
	if envConfig.StandingInstruction != nil {
		t.Fatalf("CNM_CUSTOM_PROMPT should not populate standingInstruction: %+v", envConfig)
	}
}

func TestConfigFilesRejectCustomPrompt(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte("{\n  \"customPrompt\": \"legacy prompt\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadUserConfig(ConfigEnvironment{CWD: cwd, Env: map[string]string{"CNM_HOME": home}})
	if err == nil || !strings.Contains(err.Error(), "Unsupported config key `customPrompt`") {
		t.Fatalf("expected customPrompt config file rejection, got %v", err)
	}
}

func TestGetEnvConfigRejectsInvalidValues(t *testing.T) {
	if _, err := GetEnvConfig(map[string]string{"CNM_PROVIDER": "nope"}); err == nil {
		t.Fatal("expected invalid provider error")
	}
	if _, err := GetEnvConfig(map[string]string{"CNM_PROMPT_STYLE": "bad-style"}); err == nil {
		t.Fatal("expected invalid prompt style error")
	}
}

func TestParseKeyValueAndAssertConfigKey(t *testing.T) {
	providerConfig, err := ParseKeyValue("provider", "openai-responses")
	if err != nil {
		t.Fatalf("ParseKeyValue provider error: %v", err)
	}
	if providerConfig.Provider == nil || *providerConfig.Provider != ProviderOpenAIResponses {
		t.Fatalf("unexpected provider config: %+v", providerConfig)
	}
	if _, err := AssertConfigKey("invalid"); err == nil {
		t.Fatal("expected invalid config key error")
	}
	if _, err := ParseKeyValue("promptStyle", ""); err == nil {
		t.Fatal("expected empty prompt style error")
	}
}

func TestWriteUnsetAndRedactUserConfig(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	env := map[string]string{"CNM_HOME": home}
	provider := ProviderOpenAIResponses
	model := "gpt-5-mini"
	apiKey := "sk_test_secret_1234567890"
	promptStyle := PromptStyleConventional
	result, err := WriteUserConfig(ConfigValues{
		Provider:    &provider,
		Model:       &model,
		APIKey:      &apiKey,
		PromptStyle: &promptStyle,
	}, ConfigEnvironment{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(result.Path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected 0600 config permissions, got %o", info.Mode().Perm())
		}
	}

	loaded, err := LoadUserConfig(ConfigEnvironment{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("LoadUserConfig error: %v", err)
	}
	if loaded.APIKey == nil || *loaded.APIKey != apiKey {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}

	unsetResult, err := UnsetUserConfigKey(ConfigKeyAPIKey, ConfigEnvironment{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("UnsetUserConfigKey error: %v", err)
	}
	loaded, err = LoadUserConfig(ConfigEnvironment{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("LoadUserConfig after unset error: %v", err)
	}
	if loaded.APIKey != nil {
		t.Fatalf("expected api key to be removed, got %+v", loaded)
	}
	if unsetResult.Path == "" {
		t.Fatal("expected unset result path")
	}

	effective := EffectiveConfig{Provider: provider, Model: model, PromptStyle: promptStyle, APIKey: &apiKey}
	view := ToJSONConfigView(effective)
	if view.APIKey == nil || *view.APIKey != "[redacted]" {
		t.Fatalf("expected redacted api key view, got %+v", view)
	}
	if got := GetConfigValue(effective, ConfigKeyAPIKey); got == nil || *got != "[redacted]" {
		t.Fatalf("expected redacted config get value, got %+v", got)
	}
}

func TestLoadProjectConfigIgnoresProjectSecrets(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ".cnmrc.json"), []byte("{\n  \"provider\": \"openai-compatible\",\n  \"apiKey\": \"project-secret\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, warnings, err := LoadProjectConfig(ConfigEnvironment{CWD: cwd, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("LoadProjectConfig error: %v", err)
	}
	if loaded.APIKey != nil {
		t.Fatalf("expected project api key to be stripped, got %+v", loaded)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected warnings for project provider and api key, got %v", warnings)
	}
}

func TestResolveEffectiveConfigUsesRecommendationsWithoutForcingPrivateProvider(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	env := map[string]string{"CNM_HOME": home}

	userProvider := ProviderOpenAIResponses
	userModel := "user-model"
	if _, err := WriteUserConfig(ConfigValues{
		Provider: &userProvider,
		Model:    &userModel,
	}, ConfigEnvironment{CWD: cwd, Env: env}); err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(cwd, ".cnmrc.json"), []byte("{\n  \"provider\": \"anthropic-messages\",\n  \"model\": \"project-model\",\n  \"baseURL\": \"https://project.example/v1\",\n  \"recommendedProvider\": \"google-gemini\",\n  \"recommendedModel\": \"gemini-team\",\n  \"promptStyle\": \"google\",\n  \"messageLanguage\": \"zh-CN\",\n  \"standingInstruction\": \"Keep repository migrations separate.\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveEffectiveConfig(ResolveConfigOptions{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig error: %v", err)
	}

	if resolved.Values.Provider != userProvider || resolved.Values.Model != userModel {
		t.Fatalf("project config forced private provider/model: %+v", resolved.Values)
	}
	if resolved.Values.BaseURL != nil {
		t.Fatalf("project baseURL should not become effective private preference: %+v", resolved.Values.BaseURL)
	}
	if resolved.Values.RecommendedProvider == nil || *resolved.Values.RecommendedProvider != ProviderGoogleGemini {
		t.Fatalf("expected provider recommendation, got %+v", resolved.Values.RecommendedProvider)
	}
	if resolved.Values.RecommendedModel == nil || *resolved.Values.RecommendedModel != "gemini-team" {
		t.Fatalf("expected model recommendation, got %+v", resolved.Values.RecommendedModel)
	}
	if resolved.Values.PromptStyle != PromptStyleGoogle || resolved.Values.MessageLanguage != MessageLanguageSimplifiedChinese {
		t.Fatalf("expected shared preferences from project config, got %+v", resolved.Values)
	}
	if resolved.Values.StandingInstruction == nil || !strings.Contains(*resolved.Values.StandingInstruction, "repository migrations") {
		t.Fatalf("expected standing instruction, got %+v", resolved.Values.StandingInstruction)
	}
	if len(resolved.Warnings) == 0 {
		t.Fatal("expected warnings for ignored private project preferences")
	}
}

func TestResolveEffectiveConfigCombinesSharedAndPrivateStandingInstructions(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	env := map[string]string{"CNM_HOME": home}
	personal := "Prefer concise Chinese subjects."
	if _, err := WriteUserConfig(ConfigValues{StandingInstruction: &personal}, ConfigEnvironment{CWD: cwd, Env: env}); err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".cnmrc.json"), []byte("{\n  \"standingInstruction\": \"Keep docs changes separate when unrelated.\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveEffectiveConfig(ResolveConfigOptions{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig error: %v", err)
	}
	if resolved.Values.StandingInstruction == nil {
		t.Fatal("expected combined standing instruction")
	}
	combined := *resolved.Values.StandingInstruction
	if !strings.Contains(combined, "docs changes") || !strings.Contains(combined, "Chinese subjects") {
		t.Fatalf("expected shared and private standing instructions, got %q", combined)
	}
}

func TestResolveEffectiveConfigReportsAPIKeySources(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	provider := ProviderOpenAIResponses
	if _, err := WriteUserConfig(ConfigValues{Provider: &provider}, ConfigEnvironment{CWD: cwd, Env: map[string]string{"CNM_HOME": home}}); err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}

	secretKey := "secret-store-key"
	resolved, err := ResolveEffectiveConfig(ResolveConfigOptions{
		CWD: cwd,
		Env: map[string]string{"CNM_HOME": home},
		SecretStore: fakeSecretStore{keys: map[ProviderType]string{
			ProviderOpenAIResponses: secretKey,
		}},
	})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig with secret store error: %v", err)
	}
	if resolved.Values.APIKey == nil || *resolved.Values.APIKey != secretKey || resolved.Values.APIKeySource != APIKeySourceSecretStore {
		t.Fatalf("expected secret store key source, got %+v", resolved.Values)
	}

	resolved, err = ResolveEffectiveConfig(ResolveConfigOptions{CWD: cwd, Env: map[string]string{"CNM_HOME": home, "CNM_API_KEY": "env-key"}})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig with env key error: %v", err)
	}
	if resolved.Values.APIKey == nil || *resolved.Values.APIKey != "env-key" || resolved.Values.APIKeySource != APIKeySourceEnv {
		t.Fatalf("expected env key source, got %+v", resolved.Values)
	}

	plainKey := "plaintext-key"
	if _, err := WriteUserConfig(ConfigValues{Provider: &provider, APIKey: &plainKey}, ConfigEnvironment{CWD: cwd, Env: map[string]string{"CNM_HOME": home}}); err != nil {
		t.Fatalf("WriteUserConfig plaintext error: %v", err)
	}
	resolved, err = ResolveEffectiveConfig(ResolveConfigOptions{CWD: cwd, Env: map[string]string{"CNM_HOME": home}})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig with plaintext key error: %v", err)
	}
	if resolved.Values.APIKey == nil || *resolved.Values.APIKey != plainKey || resolved.Values.APIKeySource != APIKeySourcePlaintextConfig {
		t.Fatalf("expected plaintext key source, got %+v", resolved.Values)
	}
}

func TestSaveAPIKeyToSecretStoreDoesNotRequirePlaintextConfig(t *testing.T) {
	store := &fakeWritableSecretStore{}
	result, err := SaveAPIKeyToSecretStore(ProviderOpenAIResponses, "secret-store-key", store)
	if err != nil {
		t.Fatalf("SaveAPIKeyToSecretStore error: %v", err)
	}
	if !result.Stored || result.Source != APIKeySourceSecretStore || result.Warning != nil {
		t.Fatalf("unexpected save result: %+v", result)
	}
	stored, err := store.GetAPIKey(ProviderOpenAIResponses)
	if err != nil || stored == nil || *stored != "secret-store-key" {
		t.Fatalf("expected key in secret store, got %v err=%v", stored, err)
	}
}

func TestSaveAPIKeyToSecretStoreRequiresExplicitPlaintextFallbackWhenUnavailable(t *testing.T) {
	result, err := SaveAPIKeyToSecretStore(ProviderOpenAIResponses, "secret-store-key", nil)
	if err != nil {
		t.Fatalf("SaveAPIKeyToSecretStore error: %v", err)
	}
	if result.Stored || result.Source != APIKeySourceMissing || result.Warning == nil || !strings.Contains(*result.Warning, "--plaintext-api-key") {
		t.Fatalf("expected unavailable store warning, got %+v", result)
	}
}

func TestSummarizeConfigSourcesReportsSourcesWithoutValues(t *testing.T) {
	cwd := t.TempDir()
	home := filepath.Join(cwd, ".cnm-home")
	env := map[string]string{
		"CNM_HOME":             home,
		"CNM_API_KEY":          "sk_" + strings.Repeat("f", 32),
		"CNM_MESSAGE_LANGUAGE": "zh-CN",
	}
	userProvider := ProviderOpenAICompatible
	userModel := "user-model"
	userInstruction := "private user instruction"
	if _, err := WriteUserConfig(ConfigValues{
		Provider:            &userProvider,
		Model:               &userModel,
		StandingInstruction: &userInstruction,
	}, ConfigEnvironment{CWD: cwd, Env: env}); err != nil {
		t.Fatalf("WriteUserConfig error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".cnmrc.json"), []byte("{\n  \"promptStyle\": \"google\",\n  \"standingInstruction\": \"shared project instruction\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveEffectiveConfig(ResolveConfigOptions{CWD: cwd, Env: env})
	if err != nil {
		t.Fatalf("ResolveEffectiveConfig error: %v", err)
	}
	summary := SummarizeConfigSources(resolved)
	if summary.Provider != ConfigValueSourceUser || summary.Model != ConfigValueSourceUser || summary.APIKey != ConfigValueSourceEnv {
		t.Fatalf("unexpected private preference sources: %+v", summary)
	}
	if summary.PromptStyle != ConfigValueSourceProject || summary.MessageLanguage != ConfigValueSourceEnv || summary.StandingInstruction != "project_config,user_config" {
		t.Fatalf("unexpected shared preference sources: %+v", summary)
	}
}
