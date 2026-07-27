## Review

- **Correct:** GitHub #31 keeps the experiment bounded to one frozen base revision, at most two training configurations, and forbids product/Git workflow work before model quality passes. This matches `.cs/epics/001-o-offline-commit-flow/spec.md:154-164`.
- **Correct:** The required behaviors cover the seven styles, additive/custom-primary guidance and multi-line messages, consistent with `.cs/epics/001-o-offline-commit-flow/spec.md:54-67`.
- **Blocker:** GitHub #31 says to use the “frozen #30 prompt envelope and corpus” and a frozen 0–2 semantic rubric, but #30 only records the corpus and prompt envelope descriptively; its published rubric used four booleans, not the 0–2 rubric. The full 26-case matrix was never run, and exact prompt bytes/check definitions are not attached. Training must not start until a machine-readable evaluation manifest, prompt hashes and rubric are frozen.
- **Blocker:** GitHub #31 requires exclusion of near-duplicate patches but gives no definition or audit procedure. A commit-hash denylist alone is insufficient for cherry-picks, squashes, renamed files or workflow-template copies.
- **Blocker:** Google, Atom and plain are not structurally distinguishable under the current specification. Google and Atom both permit concise imperative subjects and optional bodies; plain can legally produce the same result. Therefore “100% structural/style checks” cannot honestly distinguish them from `.cs/epics/001-o-offline-commit-flow/spec.md:62-64`. Freeze maintainer-approved examples and use a human pairwise rubric; do not invent regex differences after outputs are observed.
- **Note:** `progress.md` does not exist. This did not block the review because GitHub #30/#31 and the Epic contain the relevant execution state.
- **Note:** `plan.md` resolves to the old product completion plan, while the Epic explicitly says old PLAN/CONTEXT material is not the target truth (`.cs/epics/001-o-offline-commit-flow/spec.md:82-88`). It should not inform training labels.

## 最小训练数据方案

### 1. 隔离原则

最简单且最可靠的规则：

1. **整个 `ByteTrue/commit-now-myfriend` 仓库禁止进入 train/validation。**  
   #30 的七个固定输入、相邻 commits、auto history 和潜在修订版本都来自该仓库。整仓隔离比维护不断增长的 commit denylist 更安全。
2. 训练数据只从少量、明确允许再分发数据的外部仓库取得；记录仓库、许可证、commit、parent 和抓取日期。
3. 先按原始 patch family 分割，再生成风格、指导和 body 变体。同一 patch 的任何改写不得跨 split。
4. 任何 teacher 或人工标注界面都不得加载 #30 的 diff、历史消息、参考消息或评测目标。

### 2. 最小规模

建议把数据预算固定为：

| Split | Patch families | 每 family 记录 | 总记录 |
|---|---:|---:|---:|
| Train | 210 | 4 | 840 |
| Validation | 35 | 4 | 140 |
| Blind final | 8 | 共 15 个冻结 cases | 15 |

Train 与 validation 应按**源仓库分组隔离**，而不只是随机按 commit 分割。

每个训练 family 生成四条经人工确认的目标：

1. 一个均衡分配的基础 style；
2. 另一个内置 style；
3. 内置 style 加 additive guidance；
4. `custom`，其 guidance 是主要约束。

全量分布要求：

- 七种 style 各不少于 60 条训练记录；
- subject-only 与 subject/body 各约 50%；
- guidance 类别 `中文 / exact prefix / bullets / Why: / concise` 均不少于 40 条；
- 中文输出不少于 10%；
- small/medium patch 均有覆盖；
- rollback/security、CI、测试、文档、重构和错误处理至少各有若干 family。

这些数字只是最小穿刺预算，不应通过复制同一 patch 或轻微改名来凑数。

### 3. 标签制作

先为每个 patch 编写一份只包含 diff 可证实事实的 canonical fact sheet，再据此写四个目标。所有目标至少由一名标注者编写、另一名复核：

- subject 不得包含 diff 无法支持的动机；
- `Why:` 只用于原因可由测试名、注释、错误路径或安全修复关系推导的 patch；
- bullets 必须表达不同的 material changes，不能把一句话拆成两条；
- 中文允许保留代码标识符，但不能退化为英文消息加少量中文；
- body 必须提供 subject 没有表达的补充信息。

每条记录保存：

```text
source_repo + source_revision
source_license
commit + parent
raw_diff_sha256
normalized_patch_sha256
patch_family_id
split
style
history
guidance
target
target_author/reviewer
```

## 防泄漏检查

在构造任何风格变体前执行：

1. 对 #30 七个 commits 的原始 binary diff 保存 SHA-256 和 `git patch-id --stable`。
2. 对候选训练 patch 同时比较：
   - 原始 diff SHA-256；
   - stable patch-id；
   - 仅由增加/删除内容构成的规范化 5-token shingles。
3. 满足下列任意条件即删除整个 family：
   - SHA 或 patch-id 相同；
   - shingle Jaccard ≥ 0.50；
   - 任一方向的内容 containment ≥ 0.80。
4. 对每个 #30 patch 人工复核相似度最高的 20 个候选，即使低于阈值。
5. 同样扫描 target 和 commit message，排除复制 #30 参考消息或显著短语的记录。
6. 输出 train/validation family 列表、split hash 和 leakage report；报告必须为零命中才能训练。

该方法不能证明 Gemma 基础模型从未见过公开仓库，只能严格证明本次微调没有加入这些评测数据；这是残余风险，不能表述为绝对无污染。

## 当前项目历史不能自动提供的目标

| 目标 | 为什么不能自动构造 | 最小补足 |
|---|---|---|
| Google/Atom/plain 区分 | Epic 定义重叠，历史消息没有风格标签 | 维护者冻结每种 3 个正例、3 个反例，并进行同 diff 的人工对照标注 |
| Custom Guidance | Git history 不记录生成消息时的用户提示 | 人工创建 guidance/target pair |
| 中文指导服从 | 中文 commit 不证明其由中文 guidance 产生 | 人工标注中文 guidance 与中文输出 |
| exact prefix | 历史前缀可能是作者习惯而非指令响应 | 冻结如 `SECURITY:` 的 exact-match fixture |
| bullets | 历史 body 即使有 bullets，也没有对应要求 | 人工创建恰好两条 bullets 的目标 |
| `Why:` body | 很多 diff 不包含可支持的原因 | 使用动机可从 diff 推导的专用 fixture，人工审核 |
| useful multi-line body | 历史 body 的存在不代表必要或有用 | 人工判断 body 是否增加 material information |
| 多种 auto history | 当前近期历史主要是 Conventional | 从隔离的外部仓库选纯净历史窗口；缺失风格用人工冻结的 synthetic history |
| 任意 Custom Guidance | 有限语料无法证明“任意”自然语言指令 | 明确只证明冻结的代表性类别，不宣称普遍服从 |

## 冻结评测

### A. #30 已知回归集

在训练前生成机器可执行 manifest：

- 6 个可推理真实 diffs；
- 1 个 oversize refusal diff；
- #30 原定 26 个 primary cases：
  - auto：6；
  - conventional/angular/google/atom/plain：各 3，共 15；
  - built-in + guidance：2；
  - custom：3；
- 9 个 high-risk cases 明确列出 case ID；
- 4 个 deterministic replay cases；
- 每个 case 保存完整 prompt bytes、history、diff hash、token limit、结构谓词和人工 rubric。

必须保留 #30 当时实际使用的 prefix 文本；目前评论中存在 `[security]` 与 `SECURITY:` 表述差异，不能事后任选一个。

### B. 未见 blind final set

在训练前封存 15 个 cases，仅在配置选定后打开：

1. auto + Conventional history；
2. auto + plain/中文 history；
3. conventional；
4. conventional + `Why:` body；
5. angular；
6. angular + additive guidance；
7–9. 同一 diff 的 google/atom/plain 对照；
10. google + useful Why body；
11. atom + exactly two bullets；
12. plain + 中文指导；
13. custom + 中文；
14. custom + exact `SECURITY:` prefix 和 body；
15. custom + subject、空行、两条 bullets。

至少四例要求多行 body；至少一例是 medium diff 并检查自然 EOS、无重复、未达到 token ceiling。

Blind manifest 的输入、谓词和参考 fact sheet 在 checkpoint hash 确定前不得交给训练者。两名不知道配置身份的评审分别按 0–2 评分：

- `0`：material hallucination 或漏掉主要变化；
- `1`：主要意图正确，但遗漏次要 material change 或需小改；
- `2`：完整、相关、可直接提交。

分歧交由第三人裁定。

## 自动检查

- 非空，无 fence、外围引号或模型解释；
- subject 非空；
- Conventional/Angular 使用训练前冻结的精确 grammar；
- 多行格式必须是 `subject + 一个空行 + body`；
- prefix 完全匹配；
- bullets 数量和前缀完全匹配；
- `Why:` 存在且后面有非空内容；
- 中文 case 的自然语言主体必须包含足量汉字，标识符例外；
- EOS 在 160-token ceiling 前自然结束；
- 无明显重复 n-gram；
- 相同输入、量化模型、seed 和 runtime 输出逐字一致；
- 65537-byte/oversize 输入在推理前拒绝，不能截断。

## 最多两个训练配置

先冻结可训练的 `google/gemma-3-270m-it` HF revision；#30 的 bartowski GGUF revision 不能代替训练基础 revision。

- **配置 A:** 单一 LoRA SFT，rank 16；固定模块、学习率、epoch、seed 和数据顺序。
- **配置 B:** 与 A 完全相同，只把 rank 调到 32，并按相同比例调整 alpha。只有 A 已通过全部 instruction/structure checks、但语义完整度不足时才运行。

两者之间禁止：

- 修改训练或 validation 样本；
- 修改 prompt；
- 增加 epoch；
- 根据 #30 输出补标签；
- 更换量化、runtime 或 decoding 参数。

如果 A 的主要失败是 guidance、格式、hallucination 或重复，直接停止；更高 rank 没有针对性，不应浪费第二次训练。

最终质量必须在**合并并量化后的发行候选**上评估，不能用未量化 adapter 的结果代替。

## 明确停止条件

立即停止并判定该穿刺失败，如果出现任一项：

1. 无法获得清晰的数据来源/许可证或 leakage report 非零；
2. 无法在训练前冻结 exact prompt manifest 和 rubric；
3. 配置 A 的 high-risk gate 出现 hallucination、语义 0 分、guidance/格式失败或 token-ceiling 重复；
4. 配置 B 若被允许运行，仍未通过全部 9 个 high-risk cases；
5. #30 full set 或 blind set中：
   - structural/style < 100%；
   - custom guidance < 100%；
   - requested useful body < 100%；
   - 任一语义评分为 0；
   - 2 分比例 < 90%；
6. deterministic replay 不逐字相同；
7. 超限输入被截断或送入模型；
8. 量化后质量跌破上述任一门槛；
9. 完整发行 payload > 300,000,000 bytes；
10. 离线/禁止出网环境不能完成推理。

触发后不得运行第三个配置、调整冻结评测、降低门槛或开始 Git 工作流重写；只报告“该模型/训练方案不足”。