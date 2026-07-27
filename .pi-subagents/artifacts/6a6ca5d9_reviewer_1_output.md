# GitHub #34 穿刺设计与硬件独立审查

## Review

### 结论

**当前设计应先修正后再开始数据制作或训练。** 基础模型选择和 700 MB 数字本身没有被证据否定，但现有评测流程存在一个会造成虚假通过的 blocker：已反复看过的 26 例既被继续当最终门，又允许在第一配置失败后诊断并调整第二配置。换 bundle hash 只改变封装，不会让语义用例重新变成未见测试集。

### Correct

- **基础模型事实核对正确。** 官方固定 revision `ea3f2471cf1b1f0db85067f1ef93848e38e88c25` 的 `config.json` 确认为 `Qwen2ForCausalLM`、24 层、hidden size 896、151,936 vocab、494,032,768 参数、BF16、32,768 context、tied embeddings；Apache-2.0 与官方模型卡一致。来源：`Qwen/Qwen2.5-Coder-0.5B-Instruct@ea3f.../config.json` 和 `README.md`。
- **责任边界方向正确。** `.cs/epics/001-o-offline-commit-flow/spec.md:55-67` 要求模型输出语义、代码只处理机械格式，并明确禁止 renderer 掩盖/补造语义；GitHub #34 body:35-53 又要求最终 rendered message 接受语义评分。这比让 0.5B 同时学习七套表面语法更合理。
- **质量门没有因模型变小而显式降低。** GitHub #34 body:128-149 要求 BF16 与最终 Q5 各自满足无 0 分、至少 24/26 满分、Guidance 5/5、body 4/4；且失败即停止 Git rewrite。它针对 `/tmp/cnm-train31/final-report.md:35-53` 暴露的语义、Guidance、body 和量化退化问题。
- **700 MB 并非已证明不可实现。** 以 issue 给出的 Q5 规划值 522,186,624 bytes 计算，尚余 **177,813,376 bytes** 给 runtime、launcher、renderer、license/notice 和其他安装文件。对于裁剪后的单一 llama.cpp runtime，这个余量紧但现实；正确做法是提前测完整 skeleton，而不是现在宣判失败。
- **完整多文件、repo-isolated split、来源 revision/license/hash、外部处理前 secret gate 的要求方向正确。** GitHub #34 body:55-72 明确拒绝把 #31 的单文件 slices 当完整 commit；这修正了 `/tmp/cnm-train31/build_dataset.py:122-135` 的既有局限。

## Blocker

### B1 — 已污染的 26 例仍被当最终门，且第二配置可根据第一配置的测试失败适配，会虚假通过

**证据**

- GitHub #34 body:120-123 明确复用 #31–#33 的相同 26 个语义场景和真实 diff，只因 prompt/parser/renderer 改变而换 bundle hash。
- 这些用例的逐项失败模式已被检查并公开：`/tmp/cnm-train31/final-report.md:35-53` 已列出 Guidance、body、格式及具体语义失败；Epic 也在 `.cs/epics/001-o-offline-commit-flow/spec.md:152-164` 汇总了三轮结果。训练/数据设计者不可能再“未见”这些用例。
- GitHub #34 body:102-107 允许第二配置解决“diagnosed optimization failure or underfit”，而 body:128-145 会先评第一配置的 26 例再决定是否继续。即使 checkpoint 不按 26 例选择，第二配置仍可按测试失败类别调整，测试集事实上成为 dev set。
- GitHub #34 body:153-157 直到 Q5 通过后才运行 50–100 historical review；该集合也没有要求在训练前冻结，因此仍有事后选样空间。

**影响**

一个针对已知 26 例和已知失败模式特化的模型可以过门，却不能证明对新 diff、未见 Guidance 或真实历史提交泛化。新的 evaluator hash 不能消除语义污染。

**最小修正**

1. 把现有 26 例降级为 **公开 regression set**，不得作为 GO 的最终独立证据。
2. 在训练数据选择之前，由不参与训练的人冻结一个不可见 shadow gate：使用基础模型发布之后的 commit（至少晚于 2024-11-18）或本地新制真实多文件 diff，避免训练语料与基础模型预训练记忆；只向训练方公开 hash、覆盖矩阵和 rubric，不公开 raw cases。
3. 两个训练配置都必须在任何 shadow output 生成前完整预声明。配置选择只用 train/validation；最终选定一个 BF16 checkpoint 后只运行 shadow gate 一次。若用 shadow 失败信息调整第二配置，则该 shadow gate 作废，必须换一组新的未见 gate。
4. 50–100 historical commits 的 commit IDs、canonical repo groups、抽样算法和 hash 也必须训练前冻结；旧于模型训练 cutoff 的公共提交只能作 regression，不能单独证明泛化。

在此修正前，不应执行 full training。

## High

### H1 — renderer 的权限边界和自检不足，机械层仍可掩盖模型不服从 Guidance/type/scope/auto 的失败

**位置**：GitHub #34 body:35-53、120-143；`.cs/epics/001-o-offline-commit-flow/spec.md:55-67`

Issue 只要求 malformed record 和 renderer violation 自检，并规定只给最终消息打门分；却没有限制 renderer 的输入，也让 renderer “uses recent history for auto”。若 renderer 能看到 diff、Guidance、history 或 expected metadata，它可以替模型补 exact prefix、body shape、type/scope 或 auto style，使结构/Guidance 检查通过。`type`/`scope` 又允许 null，而 Conventional/Angular 是否对 null 必须失败没有写死。另一个风险是“机械 casing”直接改写 `GitHub`、标识符或中文/英文混排，既可能损伤语义，也会把模型的 casing 失败静默修好。

**最小修正**

- 固定接口为 `render(resolvedBuiltinStyle, semanticRecord)`；renderer 不得读取 diff、Guidance、history、gold output 或 case ID。`auto` 的历史风格解析应是独立、冻结、可单测的 deterministic resolver，renderer 只接收其结果。
- Conventional/Angular 的缺失或非法 `type` 必须 reject；不得 default。scope 可选时只能省略，不能推断。subject/body 不得由 renderer 添加、删减或改写；对错误 casing 应 reject 或保留，不做猜测式修复。
- 除最终消息评分外，增加 parseable-but-wrong mutation tests：交换 subject、删除 body、错误 type/scope、复制 history、忽略 Guidance。它们必须保持语义错误并使 gate 失败。
- Guidance 合规必须由模型 record 内容体现；renderer 只可施加固定 builtin-style 语法。分别报告 raw record semantic/guidance 分和 final rendered 分，不能只报告后者。

### H2 — 五个已知 Guidance case 不能证明 Epic 要求的“任意 Custom Guidance”

**位置**：GitHub #34 body:11-17、68-69、136-143；`.cs/epics/001-o-offline-commit-flow/spec.md:117-125,198`

门只要求已知 Guidance 5/5。#31 已经围绕 Chinese、prefix、bullet、Why、SECURITY 等固定模板构造训练数据（`/tmp/cnm-train31/build_dataset.py:320-377`），这些意图和措辞又已公开。一个模型可以记住这几类模板而无法组合、改写或处理冲突指令。后续 historical 50–100 门也未要求携带 style/guidance，无法补这个缺口。

**最小修正**：保留现有 5 例作 regression；shadow gate 对每个枚举能力至少放入未见 diff + 未见措辞，并加入组合用例（例如 Chinese+Conventional、prefix+required body、exact bullets+issue reference、subject-only 与原消息 body 冲突）。按 guidance intent family 隔离 train/validation/shadow，而不只是按字符串模板隔离；所有 Guidance case 必须语义分 `2` 且精确约束全满足才算 pass。

### H3 — 数据门没有客观的 gold-label 与去重通过标准，#31 的已知薄弱检查可能被原样继承

**位置**：GitHub #34 body:55-72、112-115；`/tmp/cnm-train31/build_dataset.py:99-107,387-419`；`/tmp/cnm-train31/audit_dataset.py:29-74`

Issue 要求“strict filters”“audited random sample”“normalized near-duplicate”，但没有冻结：最小 family/repo/长度覆盖、semantic-record label 如何从原 commit message 产生、人工审查比例、near-duplicate 算法/阈值、fork/mirror 如何归组。#31 的实现只对 changed-line set 做 Jaccard 并以 `0.5` 为阈值，split audit 则只看 repo/family/exact diff hash；改路径、拆分 commit、fork/mirror 或轻微改名都可能穿透。更重要的是，`/tmp/cnm-train31/final-report.md:9-15` 记录自动审计后仍由人工发现坏 source labels 和一条 credential-bearing diff，证明旧 regex/机械审计不足。

**最小修正**

- 先冻结数据 acceptance manifest：最小 canonical repositories/families、完整多文件比例、token-length bins、语言/变更类型、body 与每类 Guidance 覆盖，再选择数据。
- 每个 gold semantic record 必须在本地对完整 diff 做人工核对；source message 只能是候选标签，不能自动视为真值。审查采用固定 seed、预声明样本量和 0 critical-label-error 门。
- 按 canonical upstream/fork group 切 split；同时检查 commit ID、规范化 patch token/shingle、路径无关 changed-line overlap 和 message/target overlap。training 对 validation/test/shadow/historical 全部做同一检查。
- secret scan 覆盖 diff、message、history、generated labels 和发布 manifest，组合成熟 scanner/高熵检查与人工审查；任何外部翻译/标注前先在本机完成。不得复用 `/tmp/cnm-train31/build_dataset.py:99-107` 的五条 regex 作为唯一门。

### H4 — “fixed seed + fresh process”只证明一次推理重复，不证明 0.5B 训练可复现；decode 甚至尚未冻结

**位置**：GitHub #34 body:100-108、151-172、187-195；`/tmp/cnm-train31/final-report.md:59-61`

官方 `generation_config.json` 默认 `do_sample=true`、temperature `0.7`、top-p `0.8`、top-k `20`、repetition penalty `1.05`；#34 没有冻结 temperature/sampler/max output/stop/context/threads，因此 decode 参数可成为不计入“两配置”的隐形调参空间。固定 seed 也不能保证 MLX/Metal 或 CUDA kernel 的跨运行位级确定性。Issue 只要求重跑推理，没有要求从干净环境复训 selected config。#31 又只把完整证据留在 `/tmp`，`/tmp/cnm-train31/final-report.md:59-61` 的路径不是持久可重建的发布证据。

**最小修正**

- 在 base baseline 前冻结完整 prompt、官方 chat template hash、grammar、parser、renderer、temperature/greedy policy、top-k/top-p、repetition penalty、max tokens、EOS/stop、context、batch/thread/GPU settings；BF16 与 Q5 除 runtime 必要差异外相同。建议 gate 用 greedy/temperature 0，避免把抽样运气当质量。
- 固定并保存 exact commands、lockfile/container、Python/MLX 或 PyTorch、OS/driver、data order、RNG states、checkpoint selection code、conversion/llama.cpp revision及所有输出 hash。
- 对最终 selected config 至少从干净目录完整复训一次；预先定义“可复现”为相同 checkpoint selection 和门结果/指标容差，而非不现实地宣称跨 GPU bitwise identical。
- scripts/manifests/evaluator 必须进入持久版本库；adapter/model/log 可放内容寻址的持久 artifact store，并记录可获取位置与 hash，不能只留 `/tmp`。

### H5 — M5 Pro 48 GB 的 40 GB/2 GB swap 门不可比、余量过小，且“practical”没有阈值

**位置**：GitHub #34 body:74-98；`.cs/epics/001-o-offline-commit-flow/spec.md:164,186-190`

- `peak process/unified-memory pressure` 混合了不同指标；#31 的 19.971 GB “reported memory”没有说明是 MLX Metal active/peak allocation、RSS 还是系统 footprint，不能直接与 40 GB 比。
- 允许 40 GB 分配再允许 2 GB swap，在 48 GB 统一内存机器上只给 OS、窗口系统、文件缓存和测量误差留下约 6 GB；这可能已进入 compression/yellow pressure，烟测通过却导致长跑抖动或系统不稳定。
- macOS swap 文件不会在进程退出时可靠回落，用“swap growth attributable”但不定义采样方法会误判。`367 GiB free` 是磁盘余量，对统一内存安全门没有帮助。
- 单个最长样本的短跑没有保证执行与真实 config 相同的 microbatch、grad accumulation、target layers、gradient checkpoint、optimizer step、validation/checkpoint save；两配置也可能有不同峰值。
- “projected run time is practical”没有数值，无法审计。RTX 5080 handoff 也必须先证明同一配置适合其实际 16 GB VRAM/WSL2 栈，不能把 Mac 失败自动当作可运行 handoff。

**最小修正**

1. 每个预声明 config 用完全相同 dtype、sequence cap、microbatch、grad accumulation、target layers、checkpointing、optimizer 做多步 forward/backward/update，并至少跑一次 validation/checkpoint save。
2. 同时报 `/usr/bin/time -l` max RSS、MLX Metal peak/active allocation、`memory_pressure` 状态、`vm_stat` pageout/compression 和 smoke 前后 swap；门改为“始终 green、无 run-attributable pageout/swap、保留预声明的 OS 余量”，而不是 40 GB+2 GB 的硬拼接。若必须给数字，应以空闲机 baseline 和 `recommendedMaxWorkingSetSize` 推导，不应先验写 40 GB。
3. 在看到 smoke 前冻结单配置最大 wall time/step time；超过即 handoff。
4. WSL2 handoff 固定 driver/CUDA/PyTorch/Flash-Attention（如使用）、VRAM/host-RAM offload policy，并先跑同等 smoke；训练框架变化要进入 config identity 和复现报告。

### H6 — 既有硬件证据会泄漏设备唯一标识，#34 deliverable 未要求清洗

**位置**：`/tmp/cnm-train31/manifest/hardware-start.json:3-5`；GitHub #34 body:187-195

#31 的 hardware manifest 包含主机名、机器序列号、Hardware UUID、Provisioning UDID 和 Activation Lock 状态。若 #34 沿用 `system_profiler` 全量输出并把 smoke evidence 附到 issue/artifact，会造成不必要的设备身份泄漏；这些字段对训练复现没有价值。

**最小修正**：硬件 manifest 改用字段白名单，只留 chip/model identifier、core/GPU count、RAM、OS/build、training framework/driver；在 hash/上传前自动拒绝 serial、UUID、UDID、hostname、用户名和绝对 home path。不要把现有未清洗 manifest 发布到外部。

## Medium

### M1 — 700 MB 门应在训练前做一次完整 skeleton 测量；“official Q5”来源也需纠正

**位置**：GitHub #34 body:19-31、159-166；`.cs/epics/001-o-offline-commit-flow/spec.md:123-126`

Qwen 官方固定 revision 的文件清单只有 safetensors/tokenizer/config/license，没有官方 GGUF；issue 中 522,186,624-byte Q5 应标明实际 community conversion 或本地 conversion 的 repo/revision/hash，不能只称“official unfine-tuned Q5”。完整体积被推迟到所有质量门之后，会在最贵的训练之后才发现 runtime/package 布局超预算。

**最小修正**：现在就用该 Q5 reference、拟用的 stripped pinned llama.cpp、launcher/parser/renderer stub、所有动态库/license/notice，按真实 npm/native 安装布局做一次 skeleton。对每个平台以解包后的 logical file bytes（包括 wrapper/platform package 的实际重复文件）计数，预留少量增长 margin；若已超过 700,000,000，训练前 STOP。最终仍须对 fused Q5 重新测量。

### M2 — 32,768 理论 context 没有转成产品可验证的输入上限

**位置**：官方 `config.json:max_position_embeddings=32768`；GitHub #34 body:25、89、163-166；`.cs/epics/001-o-offline-commit-flow/spec.md:47-53`

训练“longest allowed sequence”不等于产品 reliable context。预算必须同时容纳 chat template、style、Guidance、history、完整 diff、grammar output 和生成余量；当前 gate 没有 near-limit 语义用例，也没有 over-limit reject 用例。历史上长 diff 已是失败集中区（`.cs/epics/001-o-offline-commit-flow/spec.md:160-162`）。

**最小修正**：训练前冻结 tokenizer 计算的总 token 公式与最大 Commit Input；保留输出余量，禁止 runtime silent truncation。shadow/Q5/offline gate 各包含一个接近上限的完整多文件 diff和一个超限必须明确拒绝的 case。

### M3 — historical“90% usable”和性能“record”不是可执行 GO 标准

**位置**：GitHub #34 body:151-166；`.cs/epics/001-o-offline-commit-flow/spec.md:200-204`

`usable` 未映射到 0/1/2；若把分数 1（明确“不完整或不精确”）计为 usable，模型可大量部分正确仍过 90%。历史门也未要求七 style/Guidance 分层。性能只要求记录 cold/warm/RSS，没有最大延迟或 RSS；Epic 又承认阈值未确认，因此技术报告可能写 GO，但产品“快”尚未被接受。

**最小修正**：预先定义 historical usable（建议 `score=2`；任何 `0`/critical failure 直接失败），按 repo/长度/style/body/Guidance 分层并随机盲化。性能数据出来后必须由维护者冻结 cold/warm/RSS 上限并明确接受，之前结论只能写“quality/size viable, performance pending”，不得启动 rewrite。

### M4 — BF16 MLX 到 fused GGUF 缺少非量化转换等价性检查

**位置**：GitHub #34 body:27-29、147-149

若 MLX BF16+adapter 通过而 Q5 失败，当前流程会把 fuse、HF→GGUF、chat-template/tokenizer 差异和 Q5 量化合并成一个失败，无法确认是模型量化退化还是转换错误。

**最小修正**：在 Q5 前用同一 pinned llama.cpp 跑 fused F16/BF16 GGUF 的小型 parity set，核对 tokenizer、prompt tokens、grammar 和输出；parity 通过后再量化。该 artifact 不计入可发布候选，只用于定位转换正确性。

## 建议的最短修正顺序

1. 先封存 unseen shadow/historical gate，并一次性预声明两个 config 与 decode/evaluator。
2. 冻结 renderer 最小接口、mutation self-check 和 Guidance composition gate。
3. 冻结数据 acceptance/label audit/去重/secret 方法；清洗 hardware manifest。
4. 用真实 config 做 Mac smoke，同时做 700 MB skeleton。
5. 只在以上均通过后训练；最终 selected config 做一次 clean retrain，再运行一次 unseen gate。

## Residual risks

- 即使修正，0.5B 是否能达到 24/26 full、无 0、Guidance/body 全过仍完全未知；#31 的失败不能推出 Qwen2.5-Coder 0.5B 会成功。
- 700 MB 对 Q5 有约 177.8 MB 余量，macOS 单平台较可信；Windows/Linux、universal binary、签名/安装器仍未证明，不能从单平台穿刺外推。
- 基础模型预训练语料不可完全审计；使用模型发布后的 shadow commits只能降低、不能数学消除记忆风险。
- 人工 0/1/2 评分仍有主观性；应盲化 candidate/config 标签、固定 rubric/anchors，并记录分歧与裁决。
- `progress.md` 在要求路径不存在；本次按只读审查继续，不将其视为仓库问题。