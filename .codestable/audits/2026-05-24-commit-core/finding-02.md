---
doc_type: audit-finding
audit: 2026-05-24-commit-core
finding_id: "bug-02"
nature: bug
severity: P1
confidence: medium
suggested_action: cs-issue
status: open
---

# Finding 02：rollback 在 reset 前没有先检查并发工作区变化

## 速答

`RollbackCommitTransaction` 先执行 `git reset --mixed snapshot.Head`，再检查当前 status 是否等于 snapshot；如果用户在事务开始后、rollback 前改了工作区，这个 reset 已经先改动了 index/HEAD，才报告 status change。

## 关键证据

- `internal/git/service.go:394` — snapshot 记录开始时 HEAD 和 porcelain status。
- `internal/git/service.go:402` — snapshot status 来自 `git status --porcelain=v1 -z --untracked-files=all`。
- `internal/git/service.go:409` — rollback 入口没有先读取当前 status。
- `internal/git/service.go:416` — 直接执行 `git reset --mixed snapshot.Head`。
- `internal/git/service.go:420` — reset 之后才再次读取 status。
- `internal/git/service.go:424` — reset 后发现 status 不同才返回 `rolled_back_with_status_change`。
- `internal/git/service_test.go:357` — 现有测试覆盖“并发文件变化后 rollback 返回 status_change”，但它断言的是 reset 后检测，不证明 reset 前保护。

## 影响

如果 Autonomous Commit 创建了部分提交后失败，同时用户或外部进程改动了 index/worktree，rollback 可能先移动 HEAD/index，再报告 status change。因为 `--mixed` 通常不会覆盖工作树内容，实际破坏面不一定大，所以置信度为 medium；但这与“detect concurrent repository changes before rollback”的产品语义不完全一致。

## 修复方向

rollback 前先读取当前 porcelain status，与 snapshot status 比较；如果不同，返回 unsafe / concurrent-change 状态，不执行 reset。需要补测试证明 concurrent change 时 `git reset` 没被调用或 HEAD 不变。

## 建议动作

`cs-issue`，因为这是事务回滚行为与安全语义不一致的具体 bug 风险。
