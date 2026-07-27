## Review

- Correct: `evaluate.py` 现在只输出机械检查结果并明确标记等待独立语义评分（`.cs/epics/001-o-offline-commit-flow/artifacts/034/scripts/evaluate.py:150-161`）；`score_gate.py` 已引入独立 `raw_score`/`rendered_score`，且错误语义分数 mutation 会阻止通过（`score_gate.py:39-44`，`test_score_gate.py:44-53`）。
- Correct: hidden gate 已使用 normalized patch、changed shingles、target hash/shingles；候选集也在切分前全局去重并执行交叉 split 断言（`build_dataset.py:267-300,465-504`）。
- Correct: `run_lora.py` 输出结构化 MLX active/cache/peak telemetry；monitor 同时读取 `/usr/bin/time -l` RSS、pressure、pageout、swapout、有限 train/val loss 和 checkpoint（`run_lora.py:9-16`，`monitor_smoke.py:125-188`）。
- Correct: `augment_guidance.py` 中没有伪造 Why 转换；全部生成的 guidance variants 均经过 schema、renderer、constraint、secret/PII 验证，manifest 保持 `awaiting_blind_audit`（`augment_guidance.py:73-127,153-175,191-215`）。

- Blocker: **质量门仍接受类型错误的独立审核字段，可能产生假 GO。** `score_gate.py:35-36,48-49` 对 `disposition_pass`、`guidance_pass`、`body_pass` 直接调用 `bool()`；例如 JSON 中的 `"guidance_pass":"false"` 和 `"disposition_pass":"false"` 都会被当作真值。脚本必须要求这些字段为真正的 JSON boolean，并拒绝缺失或错误类型；当前 mutation test 没覆盖该路径。
- Blocker: **public regression 的 target-message overlap 仍未检查。** `build_dataset.py:243-254` 对 public corpus 只加入 diff、normalized patch 和 changed-line shingles，并把 target 传成空字符串；没有从对应 commit message 建立 target exact/shingle exclusions。`freeze_public_regression.py:58-67` 生成的 public rows 同样不携带 target。因而训练集仍可能包含公开 26-case 的目标消息或近重复目标，违反 issue 要求的 train/valid/test/public 全域 target overlap 隔离。
- Blocker: **repository split 不是要求的 canonical upstream/fork group。** `build_dataset.py:78-80` 仅按 repository basename 分组，并明确依赖“fork 通常保留 basename”的假设；`assign_components()` 在 `build_dataset.py:294-300` 直接把该启发式值作为 split component。重命名 fork 无法被归入同一组，因此当前交叉 split 断言不能证明 upstream/fork 隔离。
- Blocker: **smoke 仍未严格证明执行了要求数量的真实训练步。** `monitor_smoke.py:130-135` 将日志中最大的 `Iter N` 当作 `actual_steps`，`monitor_smoke.py:176` 只检查该最大值是否达到 `smoke_iters`。单条伪造的 `Iter 20: Train loss ...` 配合一个文件名为 `*.safetensors` 的文件和一条伪造 telemetry 即可满足这些证据；checkpoint 也只做文件扩展名与哈希检查（`monitor_smoke.py:141-142`）。现有测试故意省略 checkpoint 和 telemetry（`test_monitor_smoke.py:29-36`），没有测试这一完整绕过路径。

结论：上一轮第 4 项已关闭；第 1 项的职责分离已完成但 score schema 仍有 blocker；第 2、3 项仍未完全关闭。