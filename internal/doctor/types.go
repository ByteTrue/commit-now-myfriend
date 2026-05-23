package doctor

import (
	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/providers"
)

type Severity string

type CheckStatus string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"

	CheckStatusPass    CheckStatus = "pass"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
)

type Issue struct {
	Code     string   `json:"code"`
	Check    string   `json:"check"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

type ConfigSources struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"baseURL"`
	APIKey   string `json:"apiKey"`
}

type GitCheck struct {
	Status    CheckStatus `json:"status"`
	Available bool        `json:"available"`
	Version   *string     `json:"version,omitempty"`
	Message   string      `json:"message"`
}

type RepositoryCheck struct {
	Status       CheckStatus `json:"status"`
	IsRepository bool        `json:"isRepository"`
	RootPath     *string     `json:"rootPath,omitempty"`
	Message      string      `json:"message"`
}

type EffectiveConfigCheck struct {
	Status  CheckStatus           `json:"status"`
	Config  config.JSONConfigView `json:"config"`
	Sources ConfigSources         `json:"sources"`
	Message string                `json:"message"`
}

type ProviderCapabilityCheck struct {
	Status     CheckStatus                  `json:"status"`
	Capability providers.ProviderCapability `json:"capability"`
	Message    string                       `json:"message"`
}

type ProbeInput struct {
	Provider config.ProviderType `json:"provider"`
	Model    string              `json:"model"`
	Content  string              `json:"content"`
}

type ProbeResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Report struct {
	Command  string `json:"command"`
	OK       bool   `json:"ok"`
	Status   string `json:"status"`
	ReadOnly bool   `json:"readOnly"`
	Paths    struct {
		ProjectConfigPath string `json:"projectConfigPath"`
		UserConfigHome    string `json:"userConfigHome"`
		UserConfigPath    string `json:"userConfigPath"`
	} `json:"paths"`
	Checks struct {
		Git                GitCheck                `json:"git"`
		Repository         RepositoryCheck         `json:"repository"`
		EffectiveConfig    EffectiveConfigCheck    `json:"effectiveConfig"`
		ProviderCapability ProviderCapabilityCheck `json:"providerCapability"`
	} `json:"checks"`
	Probe   *ProbeResult `json:"probe,omitempty"`
	Issues  []Issue      `json:"issues"`
	Summary struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	} `json:"summary"`
}
