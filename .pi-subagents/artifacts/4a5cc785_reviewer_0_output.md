# #31 修正后微调数据复审

## Review

### 结论：**STOP**

修正已解决上一轮 7 个具体坏例，split 隔离、manifest 哈希、secret/机械格式审计及全部 custom bullet 路径检查也通过；但固定方法抽到的 valid 样本仍有 1 个由 diff 不支持的 target。对仅 35 个 valid family 的选择集，这个明显错标不能放行。

### 抽样方法

- 固定 seed：`cnm-31-review-v1`。
- 每个 split 按 `SHA-256(seed + '\0' + family)` 升序取 20 个 family；每个 family 内按 `SHA-256(seed + '\0' + meta.id)` 升序取 1 个 variant。
- 共逐条复核 60 条（train/valid/test 各 20 条），检查 target 是否由 diff 支持、subject 是否完整、style/guidance、secret、幻觉及机械坏样本。
- 对全部 1,120 条另做 JSON/结构、数量、family 归属、repo/family/diff/target 跨 split 交集、manifest SHA-256、secret 与 subject/style/guidance 机械规则审计。

### Blocker

- `/tmp/cnm-train31/data/valid.jsonl:125`（record `commitpack:samiuelson/Hipstore:63abcc1a7a806e0665ce9382ff9a0ec480cd9576:hipstore/src/test/java/tech/lab23/hipstore/EntityStorageTest.kt:v0`）：target 为 `Implement unit tests for EntityStorage`，但 diff 没有实现新单测。它把已有 `put()` 测试移动到文件末尾并给 `put()`、`remove()` 加 `@Throws(Exception::class)`，另有空行格式变化。把“重排/标注已有测试”写成“实现单测”夸大了变更，属于 diff 不支持的训练标签。固定 60 条样本直接命中该记录，因此 **STOP**；至少应修正或过滤整个 family 的对应错误 target，并用同一抽样复跑。

### High

- 无新增 high finding。专用审计脚本在项目 venv 下对 1,120 条记录返回 `ok: true`，未命中其内置 secret scan、悬空 subject、style/guidance 或 target shape 规则。

### Medium

- 无新增 medium finding。

### Low / Correct

- **上一轮 7 个具体坏例均不再出现。** 全量精确检查以下旧内容命中均为 0：train 的 `remove unnecessary test deps from`；valid 的 `improve scrabble by removing unnecessary`、`make snippet run successfully again:` 和 Redis 凭据片段 `p4d9`；test 的构造器自相矛盾 target、Antimony `getting a new` 截断 target、`Update src to test whether required class exists` 幻觉 bullet。旧 Message.kt family 仍存在，但当前抽到的修正 target 是 `/tmp/cnm-train31/data/test.jsonl` 中的 `[CNM] Test whether required class exists`，与单文件 diff 一致。
- **split 隔离通过。** train/valid/test 两两之间的 repo、family、规范化 diff SHA-256、完整 target 交集均为 0。记录数为 840/140/140，family 数为 210/35/35，每个 family 恰有 4 个 variant，且没有 family 跨 split。
- **manifest 通过。** 实算 train `d4a69413afd392de03ab507256fd6916fb385b59278a9bcc21fc0d1c3d25ba58`、valid `ab23f7a92673519adf4753efc113e510c0aa5ea78feee88465d3c9f89f404103`、test `7b38e301e5e10a8a53b892cc8ff74f924926c22bcefadf517d4e09463cc607f8`，均与 `/tmp/cnm-train31/data/manifest.json` 一致；manifest 自身 SHA-256 为 `8228d83046f754b718c680f764f5809940d093f1450cb0c90871dc9b46fdf154`。
- **全部 custom bullet target 通过路径支持检查。** 共 6 条：train 4、valid 1、test 1。每条恰有两条 bullet，且 bullet 命名的两个 basename 均对应各自 diff 的真实 `diff --git` 路径：`test_profile.py`/`docker_host.py`、`_data_type_functions.py`/`_dtypes.py`、`listener.py`/`mic.py`、`complex.py`/`distmult.py`、`decorators.py`/`utils.py`、`autostarting.md`/`tasks.py`。其中 relation-prediction diff 实际改 7 个文件，但 guidance 只要求“exactly two ... naming two changed files”，所选两文件都确实改变，因此不构成幻觉。
- 除上述 valid 错标外，其余 59 个固定样本未发现 target 无 diff 支持、subject 残缺、style/guidance 违例、secret、明显幻觉或机械坏样本。

## 最小放行条件

修正或移除 `/tmp/cnm-train31/data/valid.jsonl:125` 所属 family 的错误语义 target，重生成 manifest/hash，并以完全相同的固定 60-family 方法复审无明显坏样本后再 **GO**。