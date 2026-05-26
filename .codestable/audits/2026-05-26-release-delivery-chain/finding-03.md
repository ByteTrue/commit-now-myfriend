---
doc_type: audit-finding
audit: 2026-05-26-release-delivery-chain
finding_id: "maintainability-03"
nature: maintainability
severity: P2
confidence: high
suggested_action: cs-refactor
status: resolved
---

# Finding 03：CI / release workflow 已重复执行 Windows smoke，验证入口开始分叉

## 速答

当前 `npm test` 已经会跑 `test:node`，其中包含 `scripts/windows-smoke.test.js`；但 CI 和 release workflow 又额外单独执行了一次 `npm run test:windows-smoke`。这会让同一类验证存在两套入口，后续增加/裁剪 smoke 场景时容易出现一处改了另一处忘了跟。

## 关键证据

- `package.json:24-27` — `npm test` → `test:go && test:node`，而 `test:node` 会跑 `scripts/*.test.js`，其中已包含 `windows-smoke.test.js`。
- `.github/workflows/ci.yml:24-28` — 先跑 `npm test`，再单独跑 `npm run test:windows-smoke`。
- `.github/workflows/release.yml:52-56` — release workflow 同样先跑 `npm test`，再单独跑 `npm run test:windows-smoke`。
- `.github/workflows/release.yml:68-69` — release workflow 还额外有一个命名不同但实际命令相同的 `Explicit Windows smoke evidence` step。

## 影响

这不是功能错误，但已经形成验证入口分叉：一旦 `windows-smoke.test.js` 被重命名、拆分、或挪到别的脚本，维护者需要同时记住 package scripts、CI、release workflow 三处耦合点。短期只是重复执行，长期容易变成行为漂移。

## 修复方向

收敛成单一来源：
- 要么 `npm test` 不再隐式包含 Windows smoke，workflow 单独跑；
- 要么 `npm test` 负责全量，workflow 只保留更语义化的单步而不重复调用。

## 建议动作

`cs-refactor`，因为这是行为不变的验证结构整理问题。

## 处理结果

已修复。`npm test` 现在只跑 Go tests + installer core Node tests，Windows smoke 保留为独立的 `test:windows-smoke` 入口；CI 和 release workflow 也改成显式调用单一 Windows smoke step，不再出现 `npm test` + 两次额外重复调用的结构分叉。