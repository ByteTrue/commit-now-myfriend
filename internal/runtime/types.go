package runtime

import (
	"time"

	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
)

type ToolName string

const (
	ToolInspectCommitScope ToolName = "inspect_commit_scope"
	ToolGetDiff            ToolName = "get_diff"
	ToolReadFile           ToolName = "read_file"
	ToolPreviewCommit      ToolName = "preview_commit"
	ToolCreateCommits      ToolName = "create_commits"
	ToolRepairFile         ToolName = "repair_file"
	ToolFinish             ToolName = "finish"
	ToolAbort              ToolName = "abort"
)

type RunStatus string

const (
	RunStatusCompleted RunStatus = "completed"
	RunStatusAborted   RunStatus = "aborted"
	RunStatusLimited   RunStatus = "limited"
)

type ToolCallRequest struct {
	ID        string         `json:"id"`
	Name      ToolName       `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ToolCallResult struct {
	CallID string         `json:"callId"`
	Name   ToolName       `json:"name"`
	OK     bool           `json:"ok"`
	Result any            `json:"result,omitempty"`
	Error  *ToolCallError `json:"error,omitempty"`
}

type ToolCallError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ProviderTurn struct {
	ToolCalls []ToolCallRequest
}

type ToolCallProvider interface {
	NextToolCalls(results []ToolCallResult) (ProviderTurn, error)
}

type ToolCallRuntimeOptions struct {
	Provider      ToolCallProvider
	Tools         DomainToolSet
	Limits        LoopLimits
	ContextPolicy gitpkg.ContextPolicy
	RepairPolicy  RepairPolicy
}

type LoopLimits struct {
	MaxToolCalls       int
	MaxProviderRetries int
	MaxDuration        time.Duration
}

type RepairPolicy struct {
	AllowedPaths []string
	ConfirmWrite func(input RepairFileInput) (bool, error)
}

type RunResult struct {
	Status      RunStatus
	Message     string
	LimitReason string
	Calls       []ToolCallResult
}

type DomainToolSetOptions struct {
	InspectCommitScope func() (gitpkg.CommitScope, error)
	GetDiff            func() (DiffResult, error)
	ReadFile           func(path string) (FileReadResult, error)
	PreviewCommit      func(input CommitPreviewInput) (CommitPreviewResult, error)
	CreateCommits      func(input CreateCommitsInput) (CreateCommitsResult, error)
	RepairFile         func(input RepairFileInput) (RepairFileResult, error)
}

type DomainToolSet struct {
	inspectCommitScope func() (gitpkg.CommitScope, error)
	getDiff            func() (DiffResult, error)
	readFile           func(path string) (FileReadResult, error)
	previewCommit      func(input CommitPreviewInput) (CommitPreviewResult, error)
	createCommits      func(input CreateCommitsInput) (CreateCommitsResult, error)
	repairFile         func(input RepairFileInput) (RepairFileResult, error)
}

func NewDomainToolSet(options DomainToolSetOptions) DomainToolSet {
	return DomainToolSet{
		inspectCommitScope: options.InspectCommitScope,
		getDiff:            options.GetDiff,
		readFile:           options.ReadFile,
		previewCommit:      options.PreviewCommit,
		createCommits:      options.CreateCommits,
		repairFile:         options.RepairFile,
	}
}

type DiffResult struct {
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

type FileReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

type CommitPreviewInput struct {
	Message string `json:"message"`
}

type CommitPreviewResult struct {
	Message   string `json:"message"`
	FileCount int    `json:"fileCount"`
}

type CreateCommitsInput struct {
	Kind             string                  `json:"kind,omitempty"`
	Commits          []CreateCommitInput     `json:"commits"`
	SplitLimitations []CreateSplitLimitation `json:"splitLimitations,omitempty"`
}

type CreateCommitInput struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

type CreateSplitLimitation struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Fallback string `json:"fallback"`
}

type CreateCommitsResult struct {
	DryRun       bool                                   `json:"dryRun"`
	Status       string                                 `json:"status"`
	Error        string                                 `json:"error,omitempty"`
	Plan         CreateCommitPlanResult                 `json:"plan"`
	Commits      []CreateCommitResultSummary            `json:"commits,omitempty"`
	Results      []gitpkg.CommitScopeCommitResult       `json:"results,omitempty"`
	Transaction  gitpkg.CommitTransactionRollbackResult `json:"transaction,omitempty"`
	RetryAttempt bool                                   `json:"retryAttempt"`
	RetryCount   int                                    `json:"retryCount"`
}

type CreateCommitResultSummary struct {
	Hash     string `json:"hash,omitempty"`
	Message  string `json:"message"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

type CreateCommitPlanResult struct {
	Kind             string                   `json:"kind"`
	Commits          []CreateCommitPlanCommit `json:"commits"`
	SplitLimitations []CreateSplitLimitation  `json:"splitLimitations,omitempty"`
}

type CreateCommitPlanCommit struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

type RepairFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RepairFileResult struct {
	Path    string `json:"path"`
	Applied bool   `json:"applied"`
}
