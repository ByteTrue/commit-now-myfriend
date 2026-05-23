package providers

import (
	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

type ProviderCapability struct {
	Provider          config.ProviderType `json:"provider"`
	Protocol          string              `json:"protocol"`
	NativeToolCalls   bool                `json:"nativeToolCalls"`
	StreamingProgress bool                `json:"streamingProgress"`
	InteractiveRepair bool                `json:"interactiveRepair"`
}

type ToolProtocolAdapter interface {
	Provider() config.ProviderType
	Capability() ProviderCapability
	BuildInitialRequest(tools []runtimex.ToolName) ([]byte, error)
	ParseTurn(payload []byte) (runtimex.ProviderTurn, error)
	BuildToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error)
}

type ProviderConfig struct {
	Provider        config.ProviderType
	APIKey          *string
	BaseURL         *string
	Model           string
	MaxOutputTokens *int
	HTTPClient      HTTPDoer
	UserAgent       string
}
