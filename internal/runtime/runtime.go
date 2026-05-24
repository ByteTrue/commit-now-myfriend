package runtime

import (
	"fmt"
	"strings"
	"time"

	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
)

type ToolCallRuntime struct {
	provider       ToolCallProvider
	tools          DomainToolSet
	limits         LoopLimits
	contextPolicy  gitpkg.ContextPolicy
	repairPolicy   RepairPolicy
	readFiles      map[string]bool
	inspectedScope bool
}

func NewToolCallRuntime(options ToolCallRuntimeOptions) *ToolCallRuntime {
	limits := options.Limits
	if limits.MaxToolCalls <= 0 {
		limits.MaxToolCalls = 32
	}
	if limits.MaxProviderRetries < 0 {
		limits.MaxProviderRetries = 0
	}
	contextPolicy := options.ContextPolicy
	if contextPolicy.Mode == "" {
		contextPolicy = gitpkg.ContextPolicy{Mode: gitpkg.ContextPolicyModeBounded, FileReadsAllowed: true}
	}
	return &ToolCallRuntime{
		provider:      options.Provider,
		tools:         options.Tools,
		limits:        limits,
		contextPolicy: contextPolicy,
		repairPolicy:  options.RepairPolicy,
		readFiles:     map[string]bool{},
	}
}

func (r *ToolCallRuntime) Run() (RunResult, error) {
	if r.provider == nil {
		return RunResult{}, fmt.Errorf("tool call runtime requires a provider")
	}

	var allResults []ToolCallResult
	var pendingResults []ToolCallResult
	toolCallCount := 0
	providerRetries := 0
	noToolStrikes := 0
	startedAt := time.Now()
	for {
		turn, err := r.provider.NextToolCalls(pendingResults)
		if err != nil {
			if providerRetries >= r.limits.MaxProviderRetries {
				return RunResult{Status: RunStatusLimited, Message: err.Error(), LimitReason: "max_provider_retries", Calls: allResults}, nil
			}
			providerRetries++
			continue
		}
		providerRetries = 0
		pendingResults = nil
		if len(turn.ToolCalls) == 0 {
			if noToolStrikes >= 4 {
				return RunResult{Status: RunStatusAborted, Message: "provider returned no tool calls", Calls: allResults}, nil
			}
			noToolStrikes++
			pendingResults = []ToolCallResult{{
				CallID: fmt.Sprintf("__reminder_%d", noToolStrikes),
				Name:   ToolName("__reminder__"),
				OK:     false,
				Error: &ToolCallError{
					Code:    "no_tool_call",
					Message: "Your last response had no tool call. You MUST respond by calling exactly one tool from: inspect_commit_scope, get_diff, read_file, preview_commit, create_commits, repair_file, finish, abort. Do not add commentary. Call create_commits now if you have enough information.",
				},
			}}
			continue
		}
		noToolStrikes = 0
		for _, call := range turn.ToolCalls {
			if r.durationExceeded(startedAt) {
				return RunResult{Status: RunStatusLimited, Message: "tool call duration limit exceeded", LimitReason: "max_duration", Calls: allResults}, nil
			}
			toolCallCount++
			if toolCallCount > r.limits.MaxToolCalls {
				return RunResult{Status: RunStatusLimited, Message: "tool call limit exceeded", LimitReason: "max_tool_calls", Calls: allResults}, nil
			}
			if call.Name == ToolFinish {
				return RunResult{Status: RunStatusCompleted, Message: stringArgument(call.Arguments, "message"), Calls: allResults}, nil
			}
			if call.Name == ToolAbort {
				return RunResult{Status: RunStatusAborted, Message: stringArgument(call.Arguments, "message"), Calls: allResults}, nil
			}
			result := r.executeToolCall(call)
			pendingResults = append(pendingResults, result)
			allResults = append(allResults, result)
			if r.durationExceeded(startedAt) {
				return RunResult{Status: RunStatusLimited, Message: "tool call duration limit exceeded", LimitReason: "max_duration", Calls: allResults}, nil
			}
		}
	}
}

func (r *ToolCallRuntime) durationExceeded(startedAt time.Time) bool {
	return r.limits.MaxDuration > 0 && time.Since(startedAt) > r.limits.MaxDuration
}

func (r *ToolCallRuntime) executeToolCall(call ToolCallRequest) ToolCallResult {
	if strings.TrimSpace(call.ID) == "" {
		return invalidToolCall(call, "missing_call_id", "Tool call id is required.")
	}
	switch call.Name {
	case ToolInspectCommitScope:
		return r.executeInspectCommitScope(call)
	case ToolGetDiff:
		return r.executeGetDiff(call)
	case ToolReadFile:
		return r.executeReadFile(call)
	case ToolPreviewCommit:
		return r.executePreviewCommit(call)
	case ToolCreateCommits:
		return r.executeCreateCommits(call)
	case ToolRepairFile:
		return r.executeRepairFile(call)
	default:
		return invalidToolCall(call, "unknown_tool", fmt.Sprintf("Tool %q is not exposed by the runtime.", call.Name))
	}
}

func (r *ToolCallRuntime) executeInspectCommitScope(call ToolCallRequest) ToolCallResult {
	if r.tools.inspectCommitScope == nil {
		return invalidToolCall(call, "tool_unavailable", "inspect_commit_scope is not available.")
	}
	scope, err := r.tools.inspectCommitScope()
	if err != nil {
		return toolFailed(call, err)
	}
	r.inspectedScope = true
	return toolSucceeded(call, scope)
}

func (r *ToolCallRuntime) executeGetDiff(call ToolCallRequest) ToolCallResult {
	if r.tools.getDiff == nil {
		return invalidToolCall(call, "tool_unavailable", "get_diff is not available.")
	}
	diff, err := r.tools.getDiff()
	if err != nil {
		return toolFailed(call, err)
	}
	return toolSucceeded(call, diff)
}

func (r *ToolCallRuntime) executeReadFile(call ToolCallRequest) ToolCallResult {
	if !r.contextPolicy.FileReadsAllowed {
		return invalidToolCall(call, "context_policy_denied", "read_file is disabled by the active Context Policy.")
	}
	if r.tools.readFile == nil {
		return invalidToolCall(call, "tool_unavailable", "read_file is not available.")
	}
	path := stringArgument(call.Arguments, "path")
	if strings.TrimSpace(path) == "" {
		return invalidToolCall(call, "invalid_arguments", "read_file requires a non-empty path.")
	}
	file, err := r.tools.readFile(path)
	if err != nil {
		return toolFailed(call, err)
	}
	r.readFiles[path] = true
	return toolSucceeded(call, file)
}

func (r *ToolCallRuntime) executePreviewCommit(call ToolCallRequest) ToolCallResult {
	if r.tools.previewCommit == nil {
		return invalidToolCall(call, "tool_unavailable", "preview_commit is not available.")
	}
	message := stringArgument(call.Arguments, "message")
	if strings.TrimSpace(message) == "" {
		return invalidToolCall(call, "invalid_arguments", "preview_commit requires a non-empty message.")
	}
	preview, err := r.tools.previewCommit(CommitPreviewInput{Message: message})
	if err != nil {
		return toolFailed(call, err)
	}
	return toolSucceeded(call, preview)
}

func (r *ToolCallRuntime) executeCreateCommits(call ToolCallRequest) ToolCallResult {
	if r.tools.createCommits == nil {
		return invalidToolCall(call, "tool_unavailable", "create_commits is not available.")
	}
	if !r.inspectedScope {
		return invalidToolCall(call, "inspect_before_create_required", "create_commits requires a successful inspect_commit_scope call first.")
	}
	input, err := createCommitsInputFromArguments(call.Arguments)
	if err != nil {
		return invalidToolCall(call, "invalid_arguments", err.Error())
	}
	created, err := r.tools.createCommits(input)
	if err != nil {
		return toolFailed(call, err)
	}
	return toolSucceeded(call, created)
}

func (r *ToolCallRuntime) executeRepairFile(call ToolCallRequest) ToolCallResult {
	if r.tools.repairFile == nil {
		return invalidToolCall(call, "tool_unavailable", "repair_file is not available.")
	}
	path := stringArgument(call.Arguments, "path")
	if strings.TrimSpace(path) == "" {
		return invalidToolCall(call, "invalid_arguments", "repair_file requires a non-empty path.")
	}
	content := stringArgument(call.Arguments, "content")
	if strings.TrimSpace(content) == "" {
		return invalidToolCall(call, "invalid_arguments", "repair_file requires non-empty content.")
	}
	if !r.readFiles[path] {
		return invalidToolCall(call, "read_before_write_required", "repair_file requires a successful read_file call for the same path first.")
	}
	if !r.repairAllowed(path) {
		return invalidToolCall(call, "repair_path_not_allowed", "repair_file can only write eligible conflicted files in Interactive Repair.")
	}
	confirmed, err := r.confirmRepairWrite(RepairFileInput{Path: path, Content: content})
	if err != nil {
		return toolFailed(call, err)
	}
	if !confirmed {
		return invalidToolCall(call, "repair_confirmation_required", "repair_file requires developer confirmation before applying writes.")
	}
	repair, err := r.tools.repairFile(RepairFileInput{Path: path, Content: content})
	if err != nil {
		return toolFailed(call, err)
	}
	return toolSucceeded(call, repair)
}

func (r *ToolCallRuntime) confirmRepairWrite(input RepairFileInput) (bool, error) {
	if r.repairPolicy.ConfirmWrite == nil {
		return false, nil
	}
	return r.repairPolicy.ConfirmWrite(input)
}

func (r *ToolCallRuntime) repairAllowed(path string) bool {
	if len(r.repairPolicy.AllowedPaths) == 0 {
		return true
	}
	for _, allowed := range r.repairPolicy.AllowedPaths {
		if allowed == path {
			return true
		}
	}
	return false
}

func toolSucceeded(call ToolCallRequest, result any) ToolCallResult {
	return ToolCallResult{CallID: call.ID, Name: call.Name, OK: true, Result: result}
}

func toolFailed(call ToolCallRequest, err error) ToolCallResult {
	return ToolCallResult{CallID: call.ID, Name: call.Name, OK: false, Error: &ToolCallError{Code: "tool_failed", Message: err.Error()}}
}

func invalidToolCall(call ToolCallRequest, code string, message string) ToolCallResult {
	return ToolCallResult{
		CallID: call.ID,
		Name:   call.Name,
		OK:     false,
		Error:  &ToolCallError{Code: code, Message: message},
	}
}

func stringArgument(arguments map[string]any, key string) string {
	if arguments == nil {
		return ""
	}
	value, ok := arguments[key].(string)
	if !ok {
		return ""
	}
	return value
}

func createCommitsInputFromArguments(arguments map[string]any) (CreateCommitsInput, error) {
	if arguments == nil {
		return CreateCommitsInput{}, fmt.Errorf("create_commits requires commits")
	}
	commitsValue, ok := arguments["commits"]
	if !ok {
		return CreateCommitsInput{}, fmt.Errorf("create_commits requires commits")
	}
	commitValues, ok := commitsValue.([]any)
	if !ok {
		return CreateCommitsInput{}, fmt.Errorf("create_commits commits must be an array")
	}
	if len(commitValues) == 0 {
		return CreateCommitsInput{}, fmt.Errorf("create_commits requires at least one commit")
	}
	input := CreateCommitsInput{Kind: stringArgument(arguments, "kind")}
	splitLimitations, err := createSplitLimitationsFromArguments(arguments)
	if err != nil {
		return CreateCommitsInput{}, err
	}
	input.SplitLimitations = splitLimitations
	seen := map[string]bool{}
	for _, rawCommit := range commitValues {
		commitMap, ok := rawCommit.(map[string]any)
		if !ok {
			return CreateCommitsInput{}, fmt.Errorf("each commit must be an object")
		}
		message, _ := commitMap["message"].(string)
		if strings.TrimSpace(message) == "" {
			return CreateCommitsInput{}, fmt.Errorf("each commit requires a non-empty message")
		}
		files, err := stringArrayArgument(commitMap, "files")
		if err != nil {
			return CreateCommitsInput{}, err
		}
		if len(files) == 0 {
			return CreateCommitsInput{}, fmt.Errorf("each commit requires at least one file")
		}
		for _, file := range files {
			if strings.TrimSpace(file) == "" {
				return CreateCommitsInput{}, fmt.Errorf("commit files must be non-empty paths")
			}
			if seen[file] {
				return CreateCommitsInput{}, fmt.Errorf("commit plan assigns file more than once: %s", file)
			}
			seen[file] = true
		}
		input.Commits = append(input.Commits, CreateCommitInput{Message: message, Files: files})
	}
	return input, nil
}

func createSplitLimitationsFromArguments(arguments map[string]any) ([]CreateSplitLimitation, error) {
	value, ok := arguments["splitLimitations"]
	if !ok {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("splitLimitations must be an array")
	}
	limitations := make([]CreateSplitLimitation, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each split limitation must be an object")
		}
		limitations = append(limitations, CreateSplitLimitation{Code: stringMapArgument(object, "code"), Message: stringMapArgument(object, "message"), Fallback: stringMapArgument(object, "fallback")})
	}
	return limitations, nil
}

func stringMapArgument(arguments map[string]any, key string) string {
	value, ok := arguments[key].(string)
	if !ok {
		return ""
	}
	return value
}

func stringArrayArgument(arguments map[string]any, key string) ([]string, error) {
	value, ok := arguments[key]
	if !ok {
		return nil, fmt.Errorf("each commit requires files")
	}
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("commit files must be strings")
			}
			result = append(result, text)
		}
		return result, nil
	case []string:
		return append([]string{}, typed...), nil
	default:
		return nil, fmt.Errorf("commit files must be an array")
	}
}
