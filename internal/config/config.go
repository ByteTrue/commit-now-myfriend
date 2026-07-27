package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ByteTrue/commit-now-myfriend/internal/api"
	"github.com/ByteTrue/commit-now-myfriend/internal/style"
)

type Config struct {
	Provider  api.Provider `json:"provider"`
	APIKey    string       `json:"api_key"`
	APIURL    string       `json:"api_url"`
	Model     string       `json:"model"`
	Style     style.Style  `json:"style"`
	Prompt    string       `json:"prompt,omitempty"`
	MaxTokens int          `json:"max_tokens,omitempty"`
}

func Path() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "cnm", "config.json")
}

func Load() (*Config, error) {
	c := &Config{
		Provider:  api.OpenAIChat,
		APIURL:    "https://api.openai.com",
		Model:     "gpt-4o-mini",
		Style:     style.Auto,
		MaxTokens: 200,
	}

	// Try config file
	data, err := os.ReadFile(Path())
	if err == nil {
		if err := json.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}
	}

	// Env vars override file
	if v := os.Getenv("CNM_PROVIDER"); v != "" {
		c.Provider = api.Provider(v)
	}
	if v := os.Getenv("CNM_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("CNM_API_URL"); v != "" {
		c.APIURL = v
	}
	if v := os.Getenv("CNM_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("CNM_STYLE"); v != "" {
		c.Style = style.Style(v)
	}
	if v := os.Getenv("CNM_PROMPT"); v != "" {
		c.Prompt = v
	}

	// Common API key env vars
	if c.APIKey == "" {
		c.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if c.APIKey == "" {
		c.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return c, nil
}

func Save(c *Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (c *Config) OverrideFromCLI(st, prompt, provider, model, apiKey, apiURL string) {
	if st != "" {
		c.Style = style.Style(st)
	}
	if prompt != "" {
		c.Prompt = prompt
	}
	if provider != "" {
		c.Provider = api.Provider(provider)
	}
	if model != "" {
		c.Model = model
	}
	if apiKey != "" {
		c.APIKey = apiKey
	}
	if apiURL != "" {
		c.APIURL = apiURL
	}
}