# GitHub #31 最终高风险语义评审

## Review

- **Blocker — 唯一结论：STOP。** 风格、附加 guidance 和 body 控制仍明显不可靠，直接触发 #31 的停止门。BF16 在 26 例中仅 **7 pass / 3 partial / 16 fail**；Q4 为 **0 pass / 1 partial / 25 fail**。
- **Correct — 评审输入完整。** 已逐例对照 `/tmp/cnm-train31/eval/high-risk.jsonl` 中的完整 system、user 和 diff，以及 `/tmp/cnm-train31/eval/results-bf16-adapter/results.json`、`/tmp/cnm-train31/eval/results-q4/results.json` 的输出；两份 `summary.json` 只作为结构检查参考，没有替代人工语义判断。
- **Blocker — guidance/body 全面失守。** 5 个带附加 guidance 的 case，BF16 和 Q4 都是 **0/5 pass**。其中 4 个明确要求 body 的 case，两候选也都是 **0/4 完全满足**。
- **Blocker — Q4 发生严重量化退化。** 严格 pass 从 BF16 的 7/26（26.9%）降至 0/26（0%，**下降 26.9 个百分点 / pass 数下降 100%**）；pass+partial 从 10/26（38.5%）降至 1/26（3.8%，**下降 34.6 个百分点 / 可用输出减少 90%**）；fail 从 16 增至 25（61.5% → 96.2%，增加 34.6 个百分点）。
- **Note — 自动结构检查显著高估质量。** `/tmp/cnm-train31/eval/results-bf16-adapter/summary.json` 报 20/26 automatic pass，但人工严格语义 pass 仅 7/26；Q4 summary 报 15/26 automatic pass，但人工严格 pass 为 0/26。结构合法不能证明内容忠实。

## 判定口径

- **pass**：忠实表达 diff 的核心变化，同时满足指定 style、guidance 和 body 要求；无实质幻觉或有害重复。
- **partial**：内容有明确 diff 依据，但过度笼统、只覆盖次要子集，或措辞存在明显不准确；不能视为可直接交付。
- **fail**：核心语义错误/幻觉、严重重复，或违反明确 style、guidance、body 契约。即使语义部分正确，违反硬性格式要求仍判 fail。

## 26 例逐候选语义核对

| Case | BF16 | Q4 | 关键依据 |
|---|---|---|---|
| `auto-small-workflow` | partial | fail | BF16 的“make release workflow more robust”有方向但未说明将 release ref 直接交给 checkout；Q4 幻觉出 diff 中不存在的 `refresh` tag。 |
| `auto-ci` | pass | fail | BF16 准确概括新增专用 GitHub Actions release workflow；Q4 把它说成模糊的 release check error，并逐字重复 subject/body。 |
| `auto-installer` | fail | fail | BF16 错称“add release workflow”；实际核心是并发安装的临时目录/文件隔离。Q4 虽提 checksum 文件，却虚构“找不到 checksums file”的故障，未覆盖隔离根因。 |
| `auto-windows-smoke` | pass | fail | BF16 的 Windows installer tests 抓住 release smoke coverage 主线；Q4 的“Windows tool string error”和 checklist body 不对应核心变化。 |
| `auto-rollback-security` | fail | fail | BF16 的“more complex API”无依据且 subject/body同义重复；Q4 的“common error in API tests”同样未描述回退安全/校验变化。 |
| `auto-audit-fixes` | partial | partial | BF16 只覆盖 tool-call 子集，遗漏 rollback safety 与 provider 相关修复；Q4 的 failure-check 描述可能对应 tool failure 子集，但表述含混且未覆盖整组审计修复。 |
| `conventional-small-workflow` | fail | fail | BF16 错称修复 `fetch-depth`（该值并未由本 diff 改成 0），漏掉 checkout ref；Q4 不符合 Conventional Commits 且虚构 Node version。 |
| `conventional-windows-smoke` | fail | fail | BF16 核心语义可接受，但完全缺少 Conventional Commit 前缀，违反指定 style；Q4 是 Windows driver 漏洞/安装提示幻觉且格式错误。 |
| `conventional-rollback-security` | fail | fail | BF16 将变化错误缩成 concurrent untracked-file test，并加入“placeholder”幻觉；Q4 既不符合 conventional 格式，也把变化说成 URL validation error。 |
| `angular-small-workflow` | fail | fail | BF16 虽有 Angular 外形，但“set up Go with release tag”不是 diff 的 checkout-ref 变化，type 也不合理；Q4 错称设置 Node tag。 |
| `angular-windows-smoke` | pass | fail | BF16 使用合法 Angular 格式，并忠实抓到 Windows installer 定位解压后 `cnm.exe` 的核心行为；Q4 为重复的 Windows driver/安装删除语句。 |
| `angular-rollback-security` | fail | fail | BF16 完整重复同一行且未覆盖 rollback/provider 变化；Q4 虽有 Angular 外形，却幻觉为 API documentation error。 |
| `google-small-workflow` | fail | fail | BF16 的“release specifier to fetch docs”对象错误且含 docs 幻觉；Q4 错称从 repo URL 删除 release tag。 |
| `google-windows-smoke` | pass | fail | BF16 是简洁、命令式且忠实的 Windows installer test 概括；Q4 幻觉 Windows 8 标准库并重复。 |
| `google-rollback-security` | fail | fail | BF16 虚构 `concurrent.txt` 并重复同一测试语义；Q4 的 API response error 与 diff 不符且重复。 |
| `atom-small-workflow` | partial | fail | BF16 提到 release ref，但“check to GitHub”不够具体，未准确表达 checkout ref；Q4 错称 repo URL/tag。 |
| `atom-windows-smoke` | pass | fail | BF16 符合 Atom 的简洁命令式风格且覆盖 smoke-test 主线；Q4 幻觉 Windows update/driver validation。 |
| `atom-rollback-security` | fail | fail | BF16 虚构 `concurrent.txt` 并重复；Q4 包含元话语“In the previous answers”及重复 API 幻觉。 |
| `plain-small-workflow` | pass | fail | BF16 的“Fix release ref check”简洁、自然且覆盖核心；Q4 错称 repo URL。 |
| `plain-windows-smoke` | pass | fail | BF16 是忠实的 plain-style 核心概括；Q4 幻觉 Windows driver 漏洞与“可信基层”测试。 |
| `plain-rollback-security` | fail | fail | BF16 再次虚构 `concurrent.txt` 并重复；Q4 是互相矛盾的 string/value 返回描述。 |
| `guided-conventional-issue-body` | fail | fail | BF16 虽包含 `#123`，但 body 没解释 reliability reason，且 subject 只说测试；Q4 连 `#123`、reliability reason 和 conventional style 都未满足。 |
| `guided-google-why-body` | fail | fail | BF16 无 body，完全遗漏“why checksum verification matters”；Q4 同样无合格 body，也未解释 checksum 原因。 |
| `custom-chinese` | fail | fail | BF16 满足中文/无 body 外形，但错误声称为开发团队添加 fetch-depth 分支；Q4 直接违反简体中文要求且语义错误。 |
| `custom-bullets` | fail | fail | BF16 完全没有要求的两条 bullet；Q4 只有一条 checklist 形式内容，并幻觉 macOS/Linux/UI tests，不是恰好两条忠实 bullet。 |
| `custom-security-prefix` | fail | fail | BF16 有 `SECURITY:` 前缀但没有要求的解释性 body，且 subject 只描述测试；Q4 既没有前缀，也没有忠实原因说明。 |

## 风格、提示词与 body 汇总

### BF16

- 总计：**7 pass / 3 partial / 16 fail**。
- Conventional 三个基础 case：**0/3 pass**；`conventional-windows-smoke` 的语义尚可但硬性格式失败，其余还存在语义错误。
- Angular 三个基础 case：**1/3 pass**；另外两例分别错义、重复。
- Google 三个基础 case：**1/3 pass**；Atom：**1/3 pass + 1 partial**；Plain：**2/3 pass**。
- 5 个 guidance case：**0/5 pass**。这单项已经足以触发 STOP。
- 4 个明确 body case：**0/4 pass**；只有 `guided-conventional-issue-body` 放入了 `#123`，但仍遗漏要求的可靠性原因。
- 关键语义失败：`auto-installer` 完全跑题；所有 rollback-security 变体基本都未抓住实际变化，并多次虚构 `concurrent.txt`/API；多个输出用 subject/body 重述同一句。

### Q4

- 总计：**0 pass / 1 partial / 25 fail**。
- 唯一 partial 是 `auto-audit-fixes`，也只是可能覆盖 tool failure handling 的一个子集。
- 5 个 guidance case：**0/5 pass**；4 个明确 body case：**0/4 pass**。
- 所有明确 style 的基础 case均无严格 pass；常见问题包括错误前缀、中文乱码式语义、无依据的 Windows driver/Windows 8/repo URL/API response 叙述，以及 subject/body 重复。
- `summary.json` 的 15 个 automatic pass 中仍包含大量明显幻觉，证明量化后不仅是格式退化，而是核心语义能力崩塌。

## 最终门结论

**STOP。** BF16 本身的严格语义 pass 率只有 26.9%，附加 guidance 与 body 均为 0% 完整成功；Q4 又将严格 pass 降为 0%，可用（pass+partial）输出减少 90%。因此不能进入 Git workflow rewrite，也不能把 Q4 作为可接受部署候选。

## 验证与残余风险

- 本次为只读语义评审；未修改仓库文件、未添加或更新测试。
- 已运行 JSON 加载/逐 case 输出对照，并读取两份 summary；未运行项目测试，因为评审对象是静态生成结果，不是代码改动。
- `git status --short && git diff --cached --name-only` 显示工作树原有 `.github/workflows/release.yml` 修改及 `.cs/`、`.pi-subagents/` 未跟踪项，但 **staging area 为空**；本评审未触碰这些内容。
- 残余风险：pass/partial 边界不可避免含人工判断；本报告采用偏严格口径，且明确 guidance/style/body 的硬性违约不受该主观边界影响，因此不会改变 STOP 结论。