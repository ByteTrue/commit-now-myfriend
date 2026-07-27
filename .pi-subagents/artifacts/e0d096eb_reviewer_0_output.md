# #31 微调数据只读审查

## Review

### 结论：**STOP**

当前数据的 split 隔离和 manifest 数量/哈希是一致的，但固定哈希抽样的 60 个独立 family 中至少发现 5 个明显坏 target（其中包含 train），另有一条疑似明文 Redis 凭据。对仅 840 条训练记录的小型穿刺集，这不是可忽略噪声；应先修正/过滤并重新生成哈希，再开始微调。

### 抽样方法与覆盖

- 固定种子：`cnm-31-review-v1`。
- 对每个 split，以 `SHA-256("cnm-31-review-v1\\0" + family)` 升序选 20 个 family；每个 family 再以 `SHA-256("cnm-31-review-v1\\0" + record_id)` 升序选 1 个 variant。
- 共人工核对 60 条、60 个不同 family：train/valid/test 各 20 条。检查了 target 对 diff 的支持程度、style/guidance、明显幻觉/错误及机械截断。
- 另外对全部 1,120 条记录做了 JSON 解析、数量、split、repo/family/diff/target 交集、style 分布、重复 target 和 manifest SHA-256 检查。

### Blocker

- **训练/验证 target 存在明显机械截断，不能直接用于当前小规模 SFT。**
  - `/tmp/cnm-train31/data/train.jsonl:307`：`test(vok-framework-v10-vo): remove unnecessary test deps from` 在介词 `from` 后结束；diff 是删除若干测试依赖，target 明显不完整。
  - `/tmp/cnm-train31/data/valid.jsonl:114`：`feat(scrabble-score): improve scrabble by removing unnecessary` 在形容词后结束，缺少宾语。
  - `/tmp/cnm-train31/data/valid.jsonl:131`：`feat(doc): make snippet run successfully again:` 以冒号结束，没有后续内容；对要求单行且简洁的 Angular target 是残句。
  - `/tmp/cnm-train31/data/test.jsonl:4`：`feat(project): change constructors to \`parse\` and \`of\` to constructors` 语义自相矛盾；从抽样 diff 可见变更是在调整/新增 value constructors，target 不能可靠描述方向。
  - `/tmp/cnm-train31/data/test.jsonl:7`：首行 `SECURITY: Clear previously-loaded models in Antimony when getting a new` 在 `a new` 后断掉，body 也不能补全缺失宾语。这看起来是把 subject 硬截到 72 字符，而不是重写成完整短句。
  - 60 条抽样中 5 条（8.3%）明显不可接受；即使只计 train/valid，也有 3/40（7.5%）。这会直接训练/选择出残句行为，且与“72 字符以内 when possible”不等同。

### High

- **抽样数据含疑似明文服务凭据。** `/tmp/cnm-train31/data/valid.jsonl:56` 的 diff 新增 `redisUri = "h:p4d9...@ec2-54-227-223-104.compute-1.amazonaws.com:60759"`。无论该历史凭据是否仍有效，都不应在未做 secret scan/脱敏和来源许可确认的可分发训练工件中保留。重新生成前应过滤该 family，并对全量 diff 做 secret scan。

- **自定义 guidance target 有明显幻觉/敷衍模板。** `/tmp/cnm-train31/data/test.jsonl:92` 的 guidance 要求“exactly two concise bullet points describing the changed files”，但 family ID/抽样 diff 仅显示 `src/main/kotlin/.../Message.kt` 一个文件；target 却写：
  - `- Update Message.kt`
  - `- Update src to test whether required class exists`
  第二条把目录 `src` 当成 changed file，且第一条没有描述实际变更。这既没有准确描述 changed files，也制造了不存在的第二个文件级变化。

### Medium

- **有效语义多样性明显低于 record 数。** `/tmp/cnm-train31/data/{train,valid,test}.jsonl` 的 1,120 条记录来自 280 个 family，每个 family 恰好 4 个 style/guidance variant；全量还有 140 组完全相同的 target（280 条记录，最大重复次数 2）。这种结构适合做 style-conditioned 对照，但不应把 `records=1120` 当成 1,120 个独立 diff/语义样本。残余风险是模型记住机械模板（尤其 `Tracking: #123`）而非学会一般化 guidance。

- **manifest 可校验产物，但不足以单独复现/审计来源过程。** `/tmp/cnm-train31/data/manifest.json:1-62` 记录 seed、数量、style/language 分布及 split 文件 SHA-256，且本次实算一致；但没有生成脚本及其哈希、上游数据版本/哈希、过滤规则、secret scan 结果或许可决策版本。`/tmp/cnm-train31/eval/manifest.json:1-13` 比 data manifest 更好地记录了 harness/output 哈希，但 `source_harness` 是临时绝对路径 `/tmp/cnm-pierce/run_corpus.py`，不构成稳定来源定位。对于“一次性穿刺”可接受为后续补强项，但在发布/复现实验前应补齐。

### Low / Correct

- **split 隔离正确。** 全量检查 train↔valid、train↔test、valid↔test：family、repo、user prompt/diff hash 和完整 target 的交集均为 0。每个 family 的 4 个 variant 都留在同一 split；没有发现同 family 跨 split 泄漏。
- **未发现高风险 eval 与 SFT 数据的直接污染。** `/tmp/cnm-train31/eval/high-risk.jsonl` 共 26 条；未发现 dataset family/repo/target 字面命中，也未发现 substantial eval prompt 与数据 user prompt 的完全匹配。残余风险：本次是精确/字面检查，没有做代码语义 clone 检测。
- **data manifest 准确。** 实算 SHA-256 分别为 train `9ffc6a...7c84`、valid `c4244f...96b7`、test `4ba883...d12d`，均与 `/tmp/cnm-train31/data/manifest.json` 一致；数量 840/140/140、family 数 210/35/35，`families.jsonl` 共 280 条且 split 映射完整。
- 抽样中其余 55 条未发现足以单独阻断的 diff 不支持、style/guidance 违例或明显幻觉；但这不抵消上述系统性截断和 secret 问题。

## 建议的最小修复门槛

1. 禁止字符级硬截 subject；对所有 1,120 条运行“完整句/悬空介词或形容词/末尾冒号”检查，并人工复核命中项。
2. 删除或脱敏 Redis 凭据 family，对全量 corpus 做 secret scan。
3. 修复 test custom-bullets target，并人工复核所有非模板化 custom guidance。
4. 重新生成 data/families/manifest，复跑相同固定哈希 60-family 抽样；无明显坏 target 后再 **GO**。