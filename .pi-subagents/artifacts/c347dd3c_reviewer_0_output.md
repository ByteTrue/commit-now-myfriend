# #31 微调数据第三次最终只读复审

## Review

### 结论：**GO**

第三次复审通过。严格复用 seed `cnm-31-review-v1` 和前两轮固定抽样算法后，train/valid/test 各 20 个 family、共 60 条逐条检查均未发现 blocker/high/medium 问题。上一轮 Hipstore family 已不在固定样本或数据中，首轮 7 个具体坏内容全量命中均为 0。当前工件满足本次 bounded pierce 的放行条件。

### 抽样方法与人工检查

- 固定 seed：`cnm-31-review-v1`。
- 每个 split 按 `SHA-256(seed + '\0' + family)` 升序取 20 个 family；每个 family 内按 `SHA-256(seed + '\0' + meta.id)` 升序取 1 个 variant。
- 共逐条核对 60 条、60 个不同 family（train/valid/test 各 20 条），检查 target 是否受 diff 支持、subject 是否完整、style/guidance 是否满足、有无 secret、明显幻觉或机械截断。
- **结果：60/60 通过。** 当前 valid 固定样本已不含上一轮 `samiuelson/Hipstore` family；替换后的固定样本也通过语义复核。其余两 split 的固定样本未发现回归。

### Blocker / High / Medium

- 无。

### Correct

- **上一轮 Hipstore 问题已消失。** 上一轮命中的 `commitpack:samiuelson/Hipstore:...:EntityStorageTest.kt` 不再出现在相同固定抽样结果中；在 family 排序和 seed 均未改变的前提下，这确认该 family 已从当前数据移除。全量文本中旧错误 target `Implement unit tests for EntityStorage` 命中 0 次。
- **首轮 7 个坏例均消失。** 全量精确检查下列旧片段均命中 0 次：`remove unnecessary test deps from`、`improve scrabble by removing unnecessary`、`make snippet run successfully again:`、Redis 凭据标记 `p4d9`、构造器自相矛盾 target、Antimony 的 `getting a new` 截断、`Update src to test whether required class exists` 幻觉 bullet。
- **全量结构和机械审计通过。** `/tmp/cnm-train31/audit_dataset.py` 在项目 venv 下对 1,120 条记录返回 `ok: true, errors: []`；记录数为 840/140/140，family 数为 210/35/35。其内置检查覆盖 secret pattern、subject shape/悬空结尾、style grammar 和 guidance 形态。
- **split 隔离通过。** `/tmp/cnm-train31/data/{train,valid,test}.jsonl` 两两之间的 repo、family、规范化 diff SHA-256、完整 target 交集均为 0。
- **manifest hash 通过。** 实算 SHA-256：train `d4a69413afd392de03ab507256fd6916fb385b59278a9bcc21fc0d1c3d25ba58`、valid `69897bc2fd1e132537cccf7ddbd90715da508c0baf801a821bfa47802a0eb42c`、test `7b38e301e5e10a8a53b892cc8ff74f924926c22bcefadf517d4e09463cc607f8`，均匹配 `/tmp/cnm-train31/data/manifest.json`。manifest 自身 SHA-256 为 `f1102ad5b674f6873c7df874fdb83bf2b0d1fa07fb9df5416a37d3bcf073e872`。
- **全部 custom bullet 路径受 diff 支持。** 共 6 条（train 4、valid 1、test 1），每条均恰有两条 bullet，所命名 basename 都对应真实 `diff --git` 路径：
  - `/tmp/cnm-train31/data/train.jsonl:112`：`test_profile.py`、`docker_host.py`
  - `/tmp/cnm-train31/data/train.jsonl:292`：`_data_type_functions.py`、`_dtypes.py`
  - `/tmp/cnm-train31/data/train.jsonl:472`：`listener.py`、`mic.py`
  - `/tmp/cnm-train31/data/train.jsonl:832`：`complex.py`、`distmult.py`
  - `/tmp/cnm-train31/data/valid.jsonl:112`：`decorators.py`、`utils.py`
  - `/tmp/cnm-train31/data/test.jsonl:112`：`autostarting.md`、`tasks.py`

### Residual risk

- 固定人工语义复核只覆盖 60/280 个 family；其余记录依赖全量机械/secret 审计，无法证明不存在未知格式 secret 或细微语义错标。这是该 bounded pierce 明示接受的小样本天然残余风险，**不单独阻断 GO**。