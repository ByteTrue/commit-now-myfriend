## Review

### Blocker（Critical）— evaluator 会把语义错误的模型输出记为 PASS

- **位置：** `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/evaluate.py:114-160`；`.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/pipeline.py:125-170`；`.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/test_pipeline.py:67-72`
- **证据：** `automatic_pass` 只取决于 JSON 解析、renderer 和少量机械 Guidance 检查。它不比较当前 diff 与 `subject/body/type/scope`，也不接收独立 0/1/2 评分；最终进程仍在全部 `automatic_pass` 时返回 0。任何格式合法但描述错误改动的 record 都能 PASS。现有 “swapped subject” 测试也只证明 renderer 不改写错误内容，没有证明错误内容会失败。
- **影响：** 直接违反 #34 的“raw semantic record 和 rendered message 都要评分，renderer syntax success 不能把语义错误变成 pass”以及 mutation self-test 要求；可产生质量门 false PASS。
- **最小修复：** 将当前结果明确限定为 `mechanical_pass`，不得据此返回质量门成功；增加必需的独立语义评分输入/阶段，并仅在每例 0/1/2 分、Guidance、body、raw record、rendered message 的冻结阈值全部满足后生成 gate PASS。增加一个交换 gold subject 后评分必须失败的 mutation test。

### Blocker（Critical）— 数据隔离只检查 raw exact hash，未实现冻结要求的泄漏检查

- **位置：** `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py:173-211, 315-365`
- **证据：** `assign_components()` 只用 `repo_group` 和完全相同的 `diff_sha256` 连边；`message_sha256` 也只是规范空白后的整条消息精确哈希。最终检查同样仅覆盖 `diff_sha256`、`message_sha256`、`family`、`component`。代码没有生成/比较：exact normalized patch、去路径 changed-line shingles、target-message near overlap；也没有加载 historical set 的 commit/diff/message/shingle 排除。shadow/public 排除仅是 canonical repo 的精确字符串和 commit/diff exact hash；改名 fork 或相同改动换路径可穿透。
- **影响：** train/validation/test/shadow/historical 之间可发生要求明确禁止的数据泄漏，之后的验证结果可能 false PASS。
- **最小修复：** 在选 split 前，为每个 family 生成 normalized-patch hash、path-independent changed-line shingle/near-duplicate signature 和 normalized target overlap signature；将所有冲突 family union 到同一 component，且对 shadow/historical manifest 做同样的排除。canonical upstream/fork group 应来自冻结映射，而不是仅凭 basename/精确 repo 字符串。任何跨集合命中必须 fail closed，并补对应测试。

### Blocker（Critical）— hardware gate 可被不完整甚至伪造的 smoke 日志判为 PASS

- **位置：** `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/monitor_smoke.py:78-129`
- **证据：** gate 只从任意子命令日志中正则提取 `Peak mem ... GB` 和 `It/sec ...`，不验证实际执行了指定数量的 train steps、forward/backward/update、validation、checkpoint save，也不扫描 NaN/Inf loss。`projected` 用总 wall time除以命令行传入的 `--smoke-iters`，而不是实际观测 step 数。内存方面只 gate `Pageouts`；未 gate `Swapouts`，`compressor_pages_delta` 仅记录不判定，`memory_pressure -Q` 只以 `<8% free` 为异常，不能证明全过程 pressure 为 normal。RSS 是每秒 `ps` 采样父进程，不是要求的 `/usr/bin/time -l` maximum RSS，也可能漏掉短峰值和子进程。
- **影响：** 一个 exit 0、打印两段匹配文本的命令即可满足核心布尔条件；真实 swapout、非有限 loss、漏 validation/checkpoint、错误 step 数或漏测 RSS 峰值也可能 PASS。这是硬件门的直接 false PASS。
- **最小修复：** 用结构化 trainer telemetry 验证配置 hash、实际 step 数、有限 loss、validation 完成和 checkpoint 文件/hash；按实际 step timestamps 计算 median/projected time。用 `/usr/bin/time -l` 包裹完整进程组并解析 maximum RSS；记录并 gate `Swapouts`、Pageouts、pressure 非 normal 状态及采样失败；MLX active/cache/peak 应从 MLX API 的结构化输出读取并要求字段齐全，缺失即失败。

### Blocker（High）— Guidance 变体可生成错误标签，且绕过数据审计

- **位置：** `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/augment_guidance.py:79-139, 171-192`；`.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/build_dataset.py:374-393`
- **证据：** `why` 对任意非空 source body 直接前置 `Why:`，没有验证原 body 真的是原因；因此事实/操作描述会被标成“reason”标签。`issue` 直接注入 source diff/message 中不存在的随机 issue reference。基础 200-row audit 在 augmentation 之前生成，guidance manifest 随后直接标记 `deterministic_guidance_ready`，没有对生成变体重新做 groundedness、label correctness、secret/PII 或人工 audit。
- **影响：** 训练/验证可包含不满足 Guidance 语义的 gold label；验证可能奖励模型复现错误变换，并且不满足 #34 对 generated transformations 和每行检查的要求。
- **最小修复：** 只从能证明 body 为 rationale 的已审核样本生成 `why`；把 Guidance 注入内容与来源/变换规则显式记录。对全部 augmentation 重新运行 secrets/PII、grounding/label checks，并把 Guidance 变体纳入冻结的盲审；审计通过前 manifest 状态不得为 ready。

### Note — 测试状态与残余风险

- `.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/test_pipeline.py` 的 13 个可加载测试通过；`test_dataset.py` 因当前环境缺少 `pyarrow` 无法加载，因此完整测试集未通过。
- 五个目标脚本均通过 `py_compile`。
- 没有针对 `evaluate.py`、`monitor_smoke.py`、split/shingle leakage 或 augmentation label validity 的端到端负向测试；上述 blocker 修复后应各留一个最小 false-PASS 回归测试。
- 本次只读审查未修改仓库文件；仅写入要求的审查报告。