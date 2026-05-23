package git

import "github.com/ByteTrue/commit-now-myfriend/internal/security"

type IssueSeverity string

const (
	IssueSeverityWarning  IssueSeverity = "warning"
	IssueSeverityBlocking IssueSeverity = "blocking"
)

type ChangeKind string

const (
	ChangeAdded      ChangeKind = "added"
	ChangeCopied     ChangeKind = "copied"
	ChangeDeleted    ChangeKind = "deleted"
	ChangeModified   ChangeKind = "modified"
	ChangeRenamed    ChangeKind = "renamed"
	ChangeTypechange ChangeKind = "typechange"
	ChangeUnmerged   ChangeKind = "unmerged"
	ChangeUnknown    ChangeKind = "unknown"
)

type Issue struct {
	Code     string        `json:"code"`
	Message  string        `json:"message"`
	Severity IssueSeverity `json:"severity"`
}

type FileStatus struct {
	Path         string      `json:"path"`
	OriginalPath *string     `json:"originalPath,omitempty"`
	Staged       *ChangeKind `json:"staged,omitempty"`
	Unstaged     *ChangeKind `json:"unstaged,omitempty"`
	Untracked    bool        `json:"untracked"`
	Binary       bool        `json:"binary"`
}

type DiffMetadata struct {
	Bytes         int  `json:"bytes"`
	OriginalBytes int  `json:"originalBytes"`
	Truncated     bool `json:"truncated"`
	OmittedBytes  int  `json:"omittedBytes"`
	MaxBytes      int  `json:"maxBytes"`
}

type ContextPolicyMode string

const (
	ContextPolicyModeBounded  ContextPolicyMode = "bounded"
	ContextPolicyModeDiffOnly ContextPolicyMode = "diff_only"
)

type ContextPolicy struct {
	Mode             ContextPolicyMode `json:"mode"`
	FileReadsAllowed bool              `json:"fileReadsAllowed"`
}

type BudgetUsage struct {
	MaxBytes      int  `json:"maxBytes"`
	UsedBytes     int  `json:"usedBytes"`
	OriginalBytes int  `json:"originalBytes,omitempty"`
	Truncated     bool `json:"truncated"`
	OmittedBytes  int  `json:"omittedBytes,omitempty"`
}

type OpaqueChangeSummary struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type ProviderVisibleFile struct {
	Path   string `json:"path"`
	Source string `json:"source"`
	Opaque bool   `json:"opaque"`
}

type PreferenceExposure struct {
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	APIKey              string `json:"apiKey"`
	PromptStyle         string `json:"promptStyle"`
	MessageLanguage     string `json:"messageLanguage"`
	StandingInstruction string `json:"standingInstruction"`
}

type AIExposureSummary struct {
	SelectedFileCount    int                   `json:"selectedFileCount"`
	OpaqueChangeCount    int                   `json:"opaqueChangeCount"`
	SecretBlockerCount   int                   `json:"secretBlockerCount"`
	DiffBudget           BudgetUsage           `json:"diffBudget"`
	ReadBudget           BudgetUsage           `json:"readBudget"`
	ProviderVisibleFiles []ProviderVisibleFile `json:"providerVisibleFiles"`
	PreferenceSources    PreferenceExposure    `json:"preferenceSources"`
	OpaqueChanges        []OpaqueChangeSummary `json:"opaqueChanges,omitempty"`
}

type RepositoryState struct {
	IsRepository         bool    `json:"isRepository"`
	RootPath             *string `json:"rootPath,omitempty"`
	GitDirPath           *string `json:"gitDirPath,omitempty"`
	IsBare               bool    `json:"isBare"`
	IsInitialCommit      bool    `json:"isInitialCommit"`
	IsDetachedHead       bool    `json:"isDetachedHead"`
	BranchName           *string `json:"branchName,omitempty"`
	MergeInProgress      bool    `json:"mergeInProgress"`
	RebaseInProgress     bool    `json:"rebaseInProgress"`
	CherryPickInProgress bool    `json:"cherryPickInProgress"`
	HasGitIdentity       bool    `json:"hasGitIdentity"`
	GitIdentity          struct {
		Name  *string `json:"name,omitempty"`
		Email *string `json:"email,omitempty"`
	} `json:"gitIdentity"`
}

type Inspection struct {
	Repository         RepositoryState           `json:"repository"`
	Files              []FileStatus              `json:"files"`
	StagedFiles        []FileStatus              `json:"stagedFiles"`
	UnstagedFiles      []FileStatus              `json:"unstagedFiles"`
	UntrackedFiles     []FileStatus              `json:"untrackedFiles"`
	StagedDiff         string                    `json:"stagedDiff"`
	Diff               DiffMetadata              `json:"diff"`
	SecretScan         security.SecretScanResult `json:"secretScan"`
	Issues             []Issue                   `json:"issues"`
	Warnings           []Issue                   `json:"warnings"`
	BlockingIssues     []Issue                   `json:"blockingIssues"`
	HasStagedChanges   bool                      `json:"hasStagedChanges"`
	HasUnstagedChanges bool                      `json:"hasUnstagedChanges"`
	HasUntrackedFiles  bool                      `json:"hasUntrackedFiles"`
}

type CommitScopeOptions struct {
	CWD               string
	Env               map[string]string
	GitRunner         CommandRunner
	StagedOnly        bool
	IncludeUntracked  bool
	DiffOnly          bool
	MaxDiffBytes      int
	MaxReadBytes      int
	PreferenceSources PreferenceExposure
	Pathspecs         []string
}

type CommitScope struct {
	Files              []FileStatus          `json:"files"`
	Pathspecs          []string              `json:"pathspecs,omitempty"`
	StagedOnly         bool                  `json:"stagedOnly"`
	IncludesUntracked  bool                  `json:"includesUntracked"`
	HasSelectedChanges bool                  `json:"hasSelectedChanges"`
	IndexSnapshot      *IndexSnapshot        `json:"indexSnapshot,omitempty"`
	SecretBlockers     []ScopedSecretFinding `json:"secretBlockers,omitempty"`
	ContextPolicy      ContextPolicy         `json:"contextPolicy"`
	AIExposure         AIExposureSummary     `json:"aiExposure"`
}

type IndexSnapshot struct {
	Head        *string      `json:"head,omitempty"`
	StagedFiles []FileStatus `json:"stagedFiles"`
}

type ScopedSecretFinding struct {
	Path        string                  `json:"path"`
	Code        string                  `json:"code"`
	Description string                  `json:"description"`
	Line        int                     `json:"line"`
	Excerpt     string                  `json:"excerpt"`
	Severity    security.SecretSeverity `json:"severity"`
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type CommandRunner func(cwd string, args []string, env map[string]string) (CommandResult, error)

type InspectOptions struct {
	CWD          string
	MaxDiffBytes int
	Env          map[string]string
	GitRunner    CommandRunner
}

type StageAllChangesOptions struct {
	CWD          string
	Confirmed    bool
	IsTTY        bool
	MaxDiffBytes int
	Env          map[string]string
	GitRunner    CommandRunner
}

type StageAllChangesResult struct {
	Staged     bool       `json:"staged"`
	Reason     string     `json:"reason"`
	Inspection Inspection `json:"inspection"`
}

type CommitScopeCommitOptions struct {
	CWD       string
	Env       map[string]string
	GitRunner CommandRunner
	Scope     CommitScope
	Message   string
	NoVerify  bool
}

type CommitScopeCommitResult struct {
	Hash    string        `json:"hash"`
	Message string        `json:"message"`
	Git     CommandResult `json:"-"`
}

type CommitTransactionSnapshot struct {
	Head   string `json:"head"`
	Status string `json:"-"`
}

type CommitTransactionRollbackResult struct {
	RolledBack bool   `json:"rolledBack"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}
