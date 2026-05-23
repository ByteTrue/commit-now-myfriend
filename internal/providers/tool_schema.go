package providers

import (
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

func toolDescription(name runtimex.ToolName) string {
	switch name {
	case runtimex.ToolInspectCommitScope:
		return "Inspect the current Commit Scope: changed files, secret blockers, AI exposure summary. Call this first."
	case runtimex.ToolGetDiff:
		return "Return the unified diff for the selected Commit Scope. Use to understand what changed."
	case runtimex.ToolReadFile:
		return "Read a single file from the working tree within the Read Budget. Required before repair_file."
	case runtimex.ToolPreviewCommit:
		return "Preview a single commit message without staging or creating it."
	case runtimex.ToolCreateCommits:
		return "Create one or more local commits from the selected Commit Scope. Each selected file must belong to exactly one commit. Use conservative file-level splits."
	case runtimex.ToolRepairFile:
		return "Write resolved content to an eligible conflicted file. Only allowed inside the Full-screen TUI Interactive Repair flow."
	case runtimex.ToolFinish:
		return "Finish the workflow after commits have been created (or the dry-run preview is acceptable)."
	case runtimex.ToolAbort:
		return "Abort the workflow with a human-readable reason. Use when the task cannot be completed safely."
	}
	return "cnm domain tool"
}

func toolSchema(name runtimex.ToolName) map[string]any {
	switch name {
	case runtimex.ToolInspectCommitScope, runtimex.ToolGetDiff:
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	case runtimex.ToolReadFile:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string", "description": "Repository-relative path of the file to read."},
			},
			"required": []string{"path"},
		}
	case runtimex.ToolPreviewCommit:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "Proposed commit message to preview."},
			},
			"required": []string{"message"},
		}
	case runtimex.ToolCreateCommits:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{
					"type":        "string",
					"description": "single or file_split. Default single.",
					"enum":        []string{"single", "file_split"},
				},
				"commits": map[string]any{
					"type":        "array",
					"description": "Ordered list of commits to create.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"message": map[string]any{"type": "string", "description": "Full commit message (subject only, or subject + body)."},
							"files":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Repository-relative file paths included in this commit."},
						},
						"required": []string{"message", "files"},
					},
				},
				"splitLimitations": map[string]any{
					"type":        "array",
					"description": "Optional notes about why the model could not split further.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"code":     map[string]any{"type": "string"},
							"message":  map[string]any{"type": "string"},
							"fallback": map[string]any{"type": "string"},
						},
						"required": []string{"code", "message"},
					},
				},
			},
			"required": []string{"commits"},
		}
	case runtimex.ToolRepairFile:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string", "description": "Eligible conflicted file path."},
				"content": map[string]any{"type": "string", "description": "Full resolved file content (no conflict markers)."},
			},
			"required": []string{"path", "content"},
		}
	case runtimex.ToolFinish:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "Optional final message."},
			},
		}
	case runtimex.ToolAbort:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "Reason the workflow was aborted."},
			},
			"required": []string{"message"},
		}
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
