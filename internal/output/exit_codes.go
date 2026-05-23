package output

type ExitCode int

const (
	Success    ExitCode = 0
	NoChange   ExitCode = 0
	DryRun     ExitCode = 0
	UserCancel ExitCode = 130
	Error      ExitCode = 1
	UsageError ExitCode = 2
)
