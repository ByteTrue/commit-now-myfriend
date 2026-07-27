## Inherited decisions

- 产品硬约束已冻结：**完全离线、完整安装包 ≤300,000,000 bytes、七种风格、任意 Custom Guidance、多行 commit message**。
- 维护者已于 2026-07-25 接受 Gemma Terms；项目记忆中“#31 尚待条款确认”的状态已经过期，不再是 blocker。
- #30 的两个候选均为 **0/9**：
  - 专用 commit LoRA：能概括部分小 diff，但继承了 English-only、single-line、约 2k context、3,000 字符截断、Python 偏置，并丢失指令服从。
  - 官方 270M IT：基础指令服从正常，但面对真实 diff 会复述或返回空结果。
- #31 最多允许两个训练配置；通过前不得建设 Git 写入流程、改写 CLI/TUI/provider 或降低质量门槛。
- 冻结的 #30 prompt、语料和高风险门必须原样保留；新数据不得包含其 commit 或近重复 patch。

## Diagnosis

### 基础模型选择

**应从官方、固定 revision 的 `google/gemma-3-270m-it` 重新做 adapter-SFT，而不是继续训练现有 commit checkpoint。**

原因：

1. 现有 commit checkpoint 本身就是从官方 IT 做的 LoRA；两条路线不是两个独立基础模型。
2. 它的 0/9 不是轻微格式偏差，而是训练契约与产品契约直接冲突：English-only、single-line、短 diff、首行后处理。
3. 它在中型和 security diff 上也没有证明可靠的 diff comprehension，因此不存在值得保护的稳定领域能力。
4. 官方 IT 已证明 chat template 和基础 instruction following 正常。为它增加 diff 能力，比从一个已灾难性遗忘的 checkpoint 恢复通用控制能力风险更小。
5. Gemma 270M 中约 170M 是词嵌入、真正 transformer block 约 100M；容量余量很小，更不应先继承一轮错误适配。

这里的核心变量不是“LoRA 还是 SFT”——LoRA 本身就是参数高效 SFT——而是：**正确的混合训练数据能否在有限 transformer 容量内形成 diff semantics 与指令控制的共同解。**

### 数据设计

使用一个冻结的数据版本和 split，包含两类数据：

1. **约 70% domain batches：diff → message**
   - 真实、多语言、多文件类型 Git diff，覆盖添加、删除、重命名、二进制 metadata、配置、脚本和代码。
   - 同一 patch 生成不同风格/指导的反事实样本；不要让某个 repository 或语言与某种 style 固定相关。
   - 覆盖七种风格、built-in + additive guidance、custom-primary、中文、前缀、bullet、`Why:`、subject-only 和 subject/body。
   - `auto` 必须带历史，并用同一 diff 配不同历史，迫使模型真正参考历史风格。
   - 加入“指导要求 diff 不支持的信息”的样本：服从格式，但不得编造 issue、原因或安全结论。
   - 加入 diff 内含类似 prompt/instruction 文本的样本，训练模型把 fenced diff 当数据而非高优先级指令。
   - 不允许静默截断；训练长度分布应覆盖实际短/中/长 prompt，超预算输入由产品拒绝。

2. **约 30% licensed instruction-replay batches**
   - 使用许可清晰、与评估无泄漏的多语言 instruction 数据。
   - 覆盖精确格式、抽取、改写、语言切换、多约束组合和“只输出指定内容”等能力。
   - 保持官方 chat template；只对 assistant token 计算 loss。
   - 单独冻结 instruction-retention 验证集，不能只看 commit 任务 loss。

关键 split 规则：

- 先按 repository 和 patch 相似度分组，再做 train/dev/test。
- 同一 patch 的所有 style/guidance 变体必须留在同一个 split。
- 整个 `ByteTrue/commit-now-myfriend` 历史应从训练和开发集排除。
- 对 CommitBench 等来源做 normalized patch/近重复检查。
- 历史 commit message 只能作为初始标签；语义不完整或与 diff 不符的目标必须剔除或人工重写。
- 合成标签必须经过 diff-grounded 审核，不能把教师幻觉蒸馏进去。

有限语料无法证明数学意义上的“任意 guidance”；可接受证据应是**组合、措辞和语言均未在训练中出现的 held-out guidance**，而不是穷举模板后宣称泛化。

## Drift / contradiction check

- **已解决的 Gemma 条款不能继续被当作 #31 blocker。** 当前 issue comment 和 Epic 已记录明确接受；旧 project-memory 条目需要视为过期前提。
- 继续训练现有 commit checkpoint，会与“保留 instruction behavior”及 32k/长 diff 方向冲突。
- 仅增加七种固定 style token 或七套模板不满足“任意 Custom Guidance”；那只是分类器，不是 instruction-following。
- 仅在 BF16/MLX checkpoint 上通过不够：发行判定对象必须是**合并后、直接从 BF16 转出的最终 GGUF 量化物**。
- 不能在看到 #30 输出后修改 frozen prompt/corpus 来帮助模型通过；可以增加独立 supplemental tests，但不能替换或放宽原门槛。
- 不应通过正则重写、截首行或 retry 隐藏模型失败；只保留已经声明的非空、无 Markdown、格式校验护栏。

## Recommendation

### 配置 A：唯一默认配置

- 固定官方 Gemma 3 270M IT BF16 revision。
- 对 transformer projection 层做保守 LoRA adapter-SFT。
- 使用上述冻结的 70/30 混合数据、完整 chat template、assistant-only loss。
- checkpoint 选择依据为联合 dev score：
  1. diff grounding/completeness；
  2. style/custom/body compliance；
  3. instruction-retention。
- 合并 adapter 后直接由 BF16 转 GGUF，再量化到已经证明有体积余量的 **Q5_K_M**；禁止从旧 Q8 二次 requantize。
- 先运行 frozen high-risk gate；全部通过后才运行完整矩阵。

### 配置 B：仅在明确的 adapter-underfit 证据下运行

- 同一个官方 base、同一数据、split、seed、prompt、sequence budget 和评估。
- 冻结 embedding/词表相关权重，对全部 transformer blocks 做低学习率 SFT，从而只改变“适配容量”这个变量。
- 仅当配置 A：
  - 保留了 instruction/custom control；
  - 无空输出、重复、幻觉；
  - 但在多个 held-out diff 上稳定漏掉 material changes；
  
  才有理由运行配置 B。

若 A 的失败是指令遗忘、训练/评估泄漏、重复、空输出或幻觉，则 B 的更大更新自由度没有合理成功假设，应停止而不是消费第二次机会。

**不建议把“继续训练 hks checkpoint”作为配置 B**：它同时改变起点、旧数据影响和优化历史，诊断价值低，并继承已知错误契约。

## Risks

应立即停止 270M 路线的证据：

1. **配置 B 仍有任一 frozen high-risk case 失败**；#31 已明确要求全部通过。
2. 任一最终候选在 security/rollback diff 上出现 material hallucination、遗漏两个核心变化之一、空输出、重复到 token 上限或把 diff 内容当指令。
3. checkpoint 轨迹显示稳定的 Pareto 冲突：能理解 diff 的 checkpoint 丢失 guidance/control，而保留 instruction 的 checkpoint 无法理解 diff；增加训练只是在两种失败间移动。
4. BF16/merged 模型通过，但所有能把完整 bundle 控制在 300,000,000 bytes 内的量化版本均失败。Q5 已接近可交付上界；更低量化若也不通过，就没有发行路径。
5. 实际冻结 prompt 加 65,536-byte 合法输入无法落入验证过的上下文预算，或者只能靠静默截断才能运行。
6. 无法建立许可清晰、与 #30 评估及其近重复隔离的数据集；这种情况下训练结果不可作为产品证据。
7. 两个配置均只能记忆训练 repository/style 模板，在 repository-held-out 与 unseen-guidance 上显著崩溃。

当前最大剩余不确定性不是训练工具链，而是仅约 100M transformer 参数是否能同时承载真实 diff comprehension 和开放式控制泛化。不得因训练成本低而进行第三轮配置或降低门槛。

## Need from main agent

无待决策项。Gemma 人工 gate 已解决；可以按上述配置 A 开始，配置 B 保留为严格条件化的最后一次实验。

## Suggested execution prompt

本次任务是只读决策审查，**不需要 executor handoff**。后续若实际执行 #31，应直接以 issue 中冻结的 manifest、数据隔离和停止门为执行合同，而不是先修改产品仓库。