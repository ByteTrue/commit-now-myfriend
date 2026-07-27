# Research: GitHub #31 — Apple M5 Pro 52GB 上微调 `google/gemma-3-270m-it` 的最小可靠路径（截至 2026-07-25）

## Summary

最小可靠路径是：**先接受 Gemma 条款并锁定官方 HF revision；在原生 Apple Silicon 上用 MLX-LM 对 BF16 checkpoint 做非量化 LoRA SFT；只有 LoRA 未过既有 9 个高风险用例时，才跑一次 full SFT；融合后用锁定 commit 的 llama.cpp 从 HF/safetensors 转 BF16 GGUF，再量化 Q8_0 与 Q4_K_M 并逐个回归。** 270M 在 52GB unified memory 上没有使用 QLoRA 的必要；Transformers+PEFT 支持 Gemma 3/LoRA，但 Apple 上的 bitsandbytes 4-bit QLoRA 不是这次穿刺的最低风险路径。

> 证据边界：运行时未提供所请求的 `fetch_content` / `get_search_content`，本报告只使用搜索索引中的官方文档/仓库证据，未声称已抓取全文或在 M5 Pro 上实跑。2026-07-25 也是未来截止点；应在实际执行日重新锁定并验证版本。

## Findings

1. **[blocker] 官方模型是 gated，不是 Apache/MIT；开始前必须由维护者本人接受 Gemma Terms。** `google/gemma-3-270m-it` 仓库公开可见，但文件下载要求登录并接受 Google usage license。微调权重属于 Model Derivative；若发布 fused/GGUF，需要随分发满足 Gemma Terms（包括向下游传递适用条款/限制），不能仅标成项目自身许可。[HF model](https://huggingface.co/google/gemma-3-270m-it) · [Gemma Terms](https://ai.google.dev/gemma/terms)

2. **[high] 首选 MLX-LM，而不是在 Mac 上搭 Transformers/PEFT+QLoRA。** MLX-LM 官方文档明确支持 LoRA、量化模型上的 QLoRA、本地 JSONL 的 `chat`/`completions`/`text` 格式，以及 `mlx_lm.fuse` 和 `--export-gguf`。社区已有 `mlx-community/gemma-3-270m-it-bf16`，证明 Gemma 3 270M 可被 MLX 转换；但可重复性更好的是从已授权的 Google checkpoint 自己转换，并记录源 revision，而不是把社区转换当供应链根。[MLX-LM](https://github.com/ml-explore/mlx-lm) · [LoRA guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md) · [community BF16 tree](https://huggingface.co/mlx-community/gemma-3-270m-it-bf16/tree/main)

3. **[medium] Transformers/PEFT 是可行备选，但不是最小 Apple 路径。** Transformers 的 PEFT 集成支持 LoRA，并能保存/加载 adapter；Google 也有官方 Transformers+TRL QLoRA 教程。问题在于该教程面向 CUDA/4-bit 工作流，Apple MPS 上引入 bitsandbytes/QLoRA 会增加后端变量。若 MLX-LM 对当时的 Gemma config/chat template 出现回归，可改用 BF16 `AutoModelForCausalLM` + PEFT LoRA + TRL `SFTTrainer`，仍不要为了 270M 使用 4-bit。[Transformers PEFT](https://huggingface.co/docs/transformers/peft) · [Google QLoRA guide](https://ai.google.dev/gemma/docs/core/huggingface_text_finetune_qlora)

4. **[high] 只跑两个配置：A 为全层 LoRA，B 为 full SFT fallback；不要做网格搜索。** 270M 的目标是严格的 commit-message style/格式遵循；LoRA 先验证是否足够，失败才升级 full SFT。两者都从 BF16 权重训练，固定 split/seed/chat template，并保留多行 body 与全部 style 条件。

   `configs/gemma270m-lora.yaml`（候选；执行前用 `mlx_lm.lora --help` 校验当时字段）：

   ```yaml
   model: artifacts/base/gemma-3-270m-it-bf16
   data: data/sft
   train: true
   fine_tune_type: lora
   num_layers: -1
   batch_size: 8
   iters: 1000
   learning_rate: 0.0001
   steps_per_report: 10
   steps_per_eval: 50
   save_every: 100
   adapter_path: artifacts/runs/lora-r8
   seed: 31
   lora_parameters:
     rank: 8
     scale: 16.0
     dropout: 0.05
   ```

   `configs/gemma270m-full.yaml`（仅 A 未过门槛才跑）：

   ```yaml
   model: artifacts/base/gemma-3-270m-it-bf16
   data: data/sft
   train: true
   fine_tune_type: full
   num_layers: -1
   batch_size: 4
   iters: 1000
   learning_rate: 0.00002
   steps_per_report: 10
   steps_per_eval: 50
   save_every: 100
   adapter_path: artifacts/runs/full
   seed: 31
   ```

   `batch_size`、`iters` 是起始值，不是假定最优值；以 validation loss 和既有 9-case pass rate 早停。若当前 MLX-LM YAML 将 LoRA 字段命名为 `rank/scale/dropout` 之外的形式，以锁定版本的 `mlx_lm.lora --help`/示例为准；manifest 必须保存最终解析配置。

5. **[high] 数据格式必须复用官方 chat template，不能手拼控制 token。** 建议 `data/sft/{train,valid,test}.jsonl` 每行使用 `messages`：system/user 明确 style 与自定义 guidance，assistant 保存精确的单行或多行 commit message。test 中保留 #30 的 9 个高风险用例且不得进入 train/valid。MLX-LM 文档确认本地 JSONL 支持 chat/completions/text，chat template 应来自 tokenizer。[MLX-LM LoRA guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md)

6. **[high] GGUF 采用“先融合 HF，再由 llama.cpp 转换”的可审计路径；MLX 直接 `--export-gguf` 只作交叉检查。** llama.cpp 曾有 Gemma 3 adapter conversion 问题，且 270M-family tokenizer 出现过 unknown BPE pre-tokenizer；因此必须锁定 llama.cpp commit、先转 BF16、再分别量化，不能假定 main 永远兼容。[Gemma3 adapter issue](https://github.com/ggml-org/llama.cpp/issues/12551) · [270M tokenizer issue](https://github.com/ggml-org/llama.cpp/issues/20111)

## 精确命令候选

以下命令是“候选执行清单”，本研究未运行。路径全部显式，便于 manifest 记录。

```bash
# 0. 原生 arm64 环境；不要在 Rosetta 下训练
uname -m                         # 期望 arm64
sw_vers
system_profiler SPHardwareDataType

# 1. 隔离并记录依赖（实际穿刺应把解析出的精确版本写入 requirements.lock）
python3.12 -m venv .venv
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install mlx-lm huggingface_hub
python -m pip freeze | tee artifacts/manifest/pip-freeze.txt
mlx_lm.lora --help | tee artifacts/manifest/mlx-lm-lora-help.txt

# 2. 维护者需先在网页接受条款，再登录；下载必须锁 revision
hf auth login
BASE_REV='<full-hugging-face-commit-sha>'
hf download google/gemma-3-270m-it \
  --revision "$BASE_REV" \
  --local-dir artifacts/base/gemma-3-270m-it-hf

# 3. 转为本地 MLX BF16；避免依赖可漂移的社区转换
mlx_lm.convert \
  --hf-path artifacts/base/gemma-3-270m-it-hf \
  --mlx-path artifacts/base/gemma-3-270m-it-bf16 \
  --dtype bfloat16

# 4A. 第一且默认的穿刺
mlx_lm.lora --config configs/gemma270m-lora.yaml \
  2>&1 | tee artifacts/runs/lora-r8/train.log

# 训练中/后先测 adapter；prompt 应由同一 tokenizer chat template 构造
mlx_lm.generate \
  --model artifacts/base/gemma-3-270m-it-bf16 \
  --adapter-path artifacts/runs/lora-r8 \
  --prompt '<one frozen smoke-test prompt>' \
  --max-tokens 256

# 4B. 仅当 A 未通过固定验收集时执行
mlx_lm.lora --config configs/gemma270m-full.yaml \
  2>&1 | tee artifacts/runs/full/train.log

# 5. LoRA 情况融合为 HF/safetensors（full run 的输出目录按锁定版本帮助调整）
mlx_lm.fuse \
  --model artifacts/base/gemma-3-270m-it-bf16 \
  --adapter-path artifacts/runs/lora-r8 \
  --save-path artifacts/fused/gemma-3-270m-it-cnm \
  --de-quantize

# 可选交叉检查，不作为唯一发布链
mlx_lm.fuse \
  --model artifacts/base/gemma-3-270m-it-bf16 \
  --adapter-path artifacts/runs/lora-r8 \
  --save-path artifacts/fused/gemma-3-270m-it-cnm-mlx \
  --export-gguf

# 6. 锁 llama.cpp commit，编译原生 Metal 工具
LLAMA_CPP_REV='<full-git-commit-sha>'
git clone https://github.com/ggml-org/llama.cpp.git vendor/llama.cpp
git -C vendor/llama.cpp checkout "$LLAMA_CPP_REV"
cmake -S vendor/llama.cpp -B vendor/llama.cpp/build \
  -DCMAKE_BUILD_TYPE=Release -DGGML_METAL=ON
cmake --build vendor/llama.cpp/build --config Release -j
python -m pip install -r vendor/llama.cpp/requirements.txt

# 7. HF/safetensors -> BF16 GGUF -> 两个且仅两个量化产物
python vendor/llama.cpp/convert_hf_to_gguf.py \
  artifacts/fused/gemma-3-270m-it-cnm \
  --outfile artifacts/gguf/gemma-3-270m-it-cnm-bf16.gguf \
  --outtype bf16
vendor/llama.cpp/build/bin/llama-quantize \
  artifacts/gguf/gemma-3-270m-it-cnm-bf16.gguf \
  artifacts/gguf/gemma-3-270m-it-cnm-q8_0.gguf Q8_0
vendor/llama.cpp/build/bin/llama-quantize \
  artifacts/gguf/gemma-3-270m-it-cnm-bf16.gguf \
  artifacts/gguf/gemma-3-270m-it-cnm-q4_k_m.gguf Q4_K_M

# 8. 固定 sampling（优先 greedy）分别跑 BF16/Q8/Q4 的同一验收集
vendor/llama.cpp/build/bin/llama-cli \
  -m artifacts/gguf/gemma-3-270m-it-cnm-q8_0.gguf \
  -ngl 99 -c 4096 -n 256 --temp 0 --seed 31 \
  -p '<rendered frozen prompt>'

# 9. 产物与数据哈希
find artifacts data configs -type f -print0 | sort -z | \
  xargs -0 shasum -a 256 > artifacts/manifest/sha256.txt
```

**命令风险说明：** MLX-LM CLI 参数会随版本演进；尤其 `convert --dtype`、YAML LoRA 子字段、full-run 保存/融合语义须以锁定版本的 `--help` 为准。不要静默改命令；把实际命令和 help 快照写入 manifest。`Q4_K_M` 对极小模型未必比 `Q4_0` 更合适，先用 llama.cpp 当前 build 的 `llama-quantize --help` 确认支持，再以 9-case 正确率决定是否保留。

## 可重复 manifest 最少字段

建议单个 `artifacts/manifest/run.json`（外加原始日志/hash 文件）记录：

- **身份/许可**：run UUID、UTC 开始/结束、操作者、HF model ID、完整 HF revision SHA、接受 Gemma Terms 的日期/主体、发布时附带条款 URL；不要记录 token。
- **硬件/OS**：Mac model identifier、M5 Pro CPU/GPU core 数、52GB unified memory、macOS build、`uname -m`、电源模式/是否接电。
- **软件供应链**：Python 与全部 pip 精确版本，`mlx`/`mlx-lm` 版本，llama.cpp 完整 git SHA、CMake flags、Xcode/clang 版本；保存 `pip freeze` 和各 CLI `--help`。
- **模型/Tokenizer**：官方 checkpoint 每文件 SHA-256、MLX 转换命令和输出 hash、`config.json`/tokenizer/chat template hash、special-token IDs、dtype。
- **数据**：训练数据来源/许可/版本、去重与清洗脚本 git SHA、schema、train/valid/test 每份 SHA-256/行数/token 数、split seed、泄漏检查结果；冻结 9-case test 的 hash。
- **训练**：最终解析后的完整 config（不仅是手写 YAML）、seed、LoRA target modules/rank/alpha(scale)/dropout、训练层、batch 与 gradient accumulation、LR/scheduler/warmup/weight decay、序列长度、mask/completion-only 行为、eval/save cadence、resume checkpoint。
- **结果**：step-wise train/valid loss、wall time、峰值 unified memory（若可测）、adapter/full/fused 每文件 hash、融合命令、是否 dequantize。
- **GGUF**：转换器 SHA、输入 hash、BF16/Q8/Q4 输出 hash、quantization type、GGUF metadata dump、文件大小。
- **验收**：基础模型、未量化 fused、BF16 GGUF、Q8、Q4 对同一 case 的逐条输出与 pass/fail；固定 prompt/rendered tokens、seed、temperature/top-p/top-k/context/max tokens；style/custom-guidance/multiline 三类必须分别计分。

## 推荐决策门槛

- A（LoRA）只有在 **9/9 高风险集通过**、validation 未明显退化、MLX adapter 与 fused HF 输出一致时才进入 GGUF。
- 只要 A 有格式/风格漏失，执行 B（full SFT），不调第二组 LoRA 超参数；这满足“最多两个配置”。
- BF16 GGUF 必须与 fused HF 的规范化文本逐例一致；Q8 必须 9/9。Q4 任何一例回退就不发布 Q4，直接保留 Q8——270M 很小，节省的体积不值得牺牲正确性。
- 许可尚未由维护者接受时，**不下载、不训练、不提交或发布衍生权重**。

## 已知风险 / residual risks

1. **blocker — license/access**：HF gate 必须由有权接受条款的人操作；自动化 token 不能替代法律接受。
2. **high — llama.cpp conversion drift**：Gemma 3 adapter 与 270M tokenizer 已有转换故障记录；锁 commit 并用 BF16 中间产物隔离问题。
3. **high — tiny-model capacity/forgetting**：#30 的 0/9 表明基础能力不足；LoRA 可能只学表面模板，full SFT 可能灾难性遗忘。固定 held-out diff 与多 style 验收是唯一可靠判据。
4. **medium — M5-specific evidence absent**：没有找到 M5 Pro 52GB 的官方训练 benchmark；结论基于 MLX 的 Apple Silicon 原生定位与 270M 尺寸，不给出未经实测的耗时/吞吐承诺。
5. **medium — CLI/schema drift**：未来 MLX-LM/Transformers 参数可能改变，必须锁版本并保存 `--help` 和解析配置。
6. **medium — quantization sensitivity**：小模型对 Q4 可能格外敏感；正确性优先，Q8 是保守发布候选。

## Sources

### Kept

- [google/gemma-3-270m-it](https://huggingface.co/google/gemma-3-270m-it) — 官方模型、gated access 与模型卡。
- [Gemma Terms of Use](https://ai.google.dev/gemma/terms) — 官方许可与衍生模型/分发约束。
- [Gemma 3 model card](https://ai.google.dev/gemma/docs/core/model_card_3) — 官方 270M 定位与训练信息。
- [MLX-LM repository](https://github.com/ml-explore/mlx-lm) — Apple Silicon 原生训练/推理工具。
- [MLX-LM LoRA guide](https://github.com/ml-explore/mlx-lm/blob/main/mlx_lm/LORA.md) — LoRA/QLoRA、数据格式、fuse、GGUF export 的一手文档。
- [mlx-community/gemma-3-270m-it-bf16](https://huggingface.co/mlx-community/gemma-3-270m-it-bf16/tree/main) — Gemma 3 270M MLX 转换存在的实证；不作为许可或供应链根。
- [Transformers PEFT](https://huggingface.co/docs/transformers/peft) — 官方 adapter/LoRA 支持说明。
- [Google Transformers QLoRA guide](https://ai.google.dev/gemma/docs/core/huggingface_text_finetune_qlora) — Google 官方 SFT/TRL/QLoRA 参考。
- [llama.cpp Gemma3 adapter conversion issue #12551](https://github.com/ggml-org/llama.cpp/issues/12551) — GGUF adapter 转换历史风险。
- [llama.cpp 270M tokenizer issue #20111](https://github.com/ggml-org/llama.cpp/issues/20111) — 270M-family BPE pre-tokenizer 转换失败的直接风险证据。

### Dropped

- TinyWeights/Medium/Substack 教程 — 可提供经验性命令，但版本、复现与一手性弱于官方 MLX-LM 文档。
- LM Studio/community quantizations — 可验证生态兼容性，但不能替代官方 checkpoint revision 与本地可审计转换。
- Reddit/LinkedIn — 无法作为可靠支持或许可证据。
- Google Gemma GitHub `LICENSE` 搜索结果 — 指向代码仓 Apache 文本，容易与模型的 Gemma Terms 混淆，因此不用作模型许可依据。

## Gaps

- 未能全文抓取锁定版本的 MLX-LM CLI/YAML schema，也未在 M5 Pro 上执行 smoke run；执行者应先保存 `mlx_lm.lora --help`，用 2 条 train/1 条 valid 做 10-step dry-run，再启动两个正式配置。
- 无法提前证明当前 llama.cpp commit 已修复 270M tokenizer；必须以锁定 commit 的实际转换结果为准。
- 没有数据集规模、现有 #31 schema 或 9-case harness 的仓库内容，因此 `iters`/batch 只是安全起点，不能声称是最优超参数。

## Review findings

- **blocker: `artifacts/manifest/run.json`（执行时应创建）— 未记录 Gemma Terms 接受主体/日期与官方 HF revision 时，不得训练或发布。**
- **high: `data/sft/test.jsonl`（执行时应创建）— 必须冻结并隔离 #30 的 9 个高风险用例，避免数据泄漏。**
- **high: `artifacts/gguf/*.gguf`（执行时应创建）— 每个量化产物都须独立跑 9-case；不能只验证 BF16。**
- 本只读研究未修改项目源文件；仅写入运行时要求的研究 artifact。

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "研究给出带 blocker/high/medium 严重度的 review findings、具体候选路径（configs/gemma270m-{lora,full}.yaml、data/sft/test.jsonl、artifacts/manifest/run.json）、精确命令候选和 residual risks。"
    }
  ],
  "changedFiles": [
    "/Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/outputs/c92190c0/research.md"
  ],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "web_search across official Gemma, MLX-LM, Transformers/PEFT, and llama.cpp sources",
      "result": "passed",
      "summary": "Found official model/license, MLX-LM LoRA/fuse docs, PEFT/Google QLoRA docs, and llama.cpp Gemma 3 conversion risk records."
    },
    {
      "command": "M5 Pro training / conversion / quantization commands listed in this report",
      "result": "not-run",
      "summary": "Read-only research; no hardware execution was requested or claimed."
    }
  ],
  "validationOutput": [
    "Primary-source URLs retained and weaker community sources explicitly dropped.",
    "Runtime configuration gap disclosed: fetch_content/get_search_content were unavailable, so no full-page fetch was claimed.",
    "Plan is bounded to two training configurations and two quantization outputs."
  ],
  "residualRisks": [
    "blocker: maintainer must explicitly accept Gemma Terms before gated download or derivative distribution",
    "high: llama.cpp Gemma 3/270M tokenizer conversion can regress; pin and test an exact commit",
    "high: LoRA may not overcome the 270M model's prior 0/9 behavior; full SFT is the only second configuration",
    "medium: no direct M5 Pro 52GB benchmark or on-device run was available",
    "medium: MLX-LM CLI/YAML fields must be verified against the pinned release"
  ],
  "noStagedFiles": true,
  "diffSummary": "No project source changes; created only the required research artifact.",
  "reviewFindings": [
    "blocker: artifacts/manifest/run.json - record license acceptance and exact HF revision before training/distribution",
    "high: data/sft/test.jsonl - freeze and isolate the nine #30 high-risk cases",
    "high: artifacts/gguf/*.gguf - independently validate BF16, Q8_0, and Q4_K_M outputs"
  ],
  "manualNotes": "Requested fetch_content/get_search_content tools were not installed for this child; findings rely on web search index evidence and linked official sources."
}
```
