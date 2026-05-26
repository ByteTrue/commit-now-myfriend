---
doc_type: audit-finding
audit: 2026-05-26-release-delivery-chain
finding_id: "bug-01"
nature: bug
severity: P1
confidence: medium
suggested_action: cs-issue
status: open
---

# Finding 01：Windows smoke 仍停留在 Linux 上的契约模拟，没有真实 `cnm.exe` 启动证据

## 速答

当前所谓的 Windows smoke coverage 主要验证命名、checksum、`cnm.exe` 定位和 wrapper 路径选择，但它们都跑在 Linux Node test 里，没有在真实 Windows runner 上执行 `Expand-Archive` 或启动 `cnm.exe`。这意味着“Windows 交付链已被自动验证”的置信度仍有限。

## 关键证据

- `.github/workflows/ci.yml:8-34` — CI 只有 `ubuntu-latest` job，没有 Windows runner。
- `.github/workflows/release.yml:18-69` — release-artifacts 也只跑在 `ubuntu-latest`。
- `scripts/windows-smoke.test.js:16-75` — Windows smoke test 验证的是 archive naming、checksum、`findBinary`、`resolveWrapperBinaryPath`；没有真实调用 `powershell.exe Expand-Archive`，也没有执行 `cnm.exe`。
- `scripts/npm-install.js:51-62` — Windows 分支真正依赖 `powershell.exe` 的 `Expand-Archive`，这条路径并未被自动执行证实。
- `scripts/cnm.js:25-34` — wrapper 真正的运行时行为是 `spawnSync(binaryPath, ...)`；Windows smoke 目前没有对 `cnm.exe --help` 的真实执行证据。

## 影响

这不是文档错误，而是验证可信度缺口：当前自动化更像“Windows contract simulation”，而不是“Windows runtime smoke”。一旦 Windows 的解压命令、路径行为或 `spawnSync` 细节出问题，Linux 上的模拟测试不会发现。因为契约模拟确实能挡住一部分错误，所以置信度为 medium，而不是 high。

## 修复方向

两条路二选一：
1. 真补 Windows runner 级 smoke（优先）；或
2. 保持 Linux 模拟测试，但把 workflow / docs 的表述明确降级为“contract smoke”，避免让维护者误以为已经有真实 Windows 执行证据。

## 建议动作

`cs-issue`，因为这是交付验证置信度与实际保障不匹配的具体缺口。