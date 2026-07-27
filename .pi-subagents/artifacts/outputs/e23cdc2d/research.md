# Research: sub-300 MB bundled model/runtime for git diff → commit message (2026-07)

## Summary

The smallest credible **shipping spike** is `hks350d/git-diff-to-commit-gemma-3-270m` converted to GGUF and quantized to Q4, because it is task-specific and Gemma 3 270M was explicitly designed for instruction following, structured text, and task-specific fine-tuning. The safest **redistribution fallback** is `HuggingFaceTB/SmolLM2-135M-Instruct` Q4_K_M (105 MB, Apache-2.0), but its likely quality risk is materially higher; Qwen2.5-Coder-0.5B already exceeds the 300 MB complete-bundle ceiling at Q4.

This is a candidate-screening result, not a quality attestation: no source provides evidence that any candidate reliably handles all seven styles, arbitrary guidance, and optional bodies. Those behaviors must be tested on the issue #30 corpus.

## Findings

1. **Best first spike — `hks350d/git-diff-to-commit-gemma-3-270m` (high quality uncertainty, medium licensing burden).** The Hugging Face repository identifies a Gemma 3 270M-derived, git-diff-to-commit fine-tune; the HF model index reports it as approximately **0.4B parameters**. Its published repository is the authoritative checkpoint location, but the available search index did not expose an exact weight-file byte count or a first-party GGUF artifact. Treat any UI-rounded size as insufficient for the 300 MB acceptance test; obtain exact bytes from the repository API/download before selecting packaging. Architecture is Gemma 3 text decoder and is supported by current llama.cpp/GGUF tooling. Task specialization makes this more credible than a generic 135M model for understanding diffs, but specialization may have narrowed instruction following: seven styles, arbitrary custom guidance, and multiline output are unproven and are the principal quality risk. [Model repository](https://huggingface.co/hks350d/git-diff-to-commit-gemma-3-270m) [HF quantized-model index](https://huggingface.co/models?other=base_model%3Aquantized%3Agoogle%2Fgemma-3-270m-it)

2. **Gemma 3 270M is technically well matched, but is not Apache-2.0 (medium legal/product severity).** Google describes Gemma 3 270M as a **270-million-parameter** compact model with instruction-following and text-structuring ability, designed to become useful through task-specific fine-tuning. Gemma 3 remains under the custom **Gemma Terms of Use**, not the Apache-2.0 license used by Gemma 4. Redistribution is allowed subject to the Agreement, but a distributor must pass the applicable use restrictions downstream, provide recipients a copy of the Agreement, preserve required notices/attributions, and comply with the incorporated prohibited-use policy; legal review should approve the installer/package notice and downstream terms before release. A derivative/fine-tune does not escape these obligations. [Google announcement](https://developers.googleblog.com/en/introducing-gemma-3-270m) [Gemma Terms](https://ai.google.dev/gemma/terms) [Gemma prohibited-use policy](https://ai.google.dev/gemma/prohibited_use_policy)

3. **Q8 → Q4 requantization is technically possible, but use an unquantized source when available (medium quality risk).** llama.cpp’s `llama-quantize` accepts GGUF input and documents conversion to Q4_K_M; the CLI has an `--allow-requantize` path. Therefore a valid Gemma Q8_0 GGUF can be requantized to Q4. However, quantizing from BF16/F16 is preferable because requantization compounds rounding loss. If only the fine-tune’s Q8 exists, Q8→Q4 is a credible spike path, but compare its outputs against a Q4 built from the original merged BF16/F16 checkpoint before shipping. The llama.cpp quantization table gives **4.83–4.84 bits/weight for Q4_K_M** versus **8.5 bits/weight for Q8_0**, so a 270M dense model’s raw-weight lower-order estimate is about **163 MB versus 287 MB**, before GGUF metadata and any tensors deliberately retained at higher precision. These are estimates, not exact artifact sizes. [llama.cpp quantize documentation](https://github.com/ggml-org/llama.cpp/blob/master/tools/quantize/README.md) [llama.cpp quantization table](https://github.com/ggml-org/llama.cpp/blob/ec450d3bbf9fdb3cd06b27c00c684fd1861cb0cf/examples/quantize/README.md) [requantization discussion/evidence](https://github.com/ggml-org/llama.cpp/discussions/5222)

4. **Gemma Q8 is unlikely to fit the complete installed bundle; Q4 plausibly fits (blocker for Q8, pending exact bytes for Q4).** At 8.5 bits/weight, 270M weights alone are roughly **286.9 MB decimal**; adding tokenizer/GGUF metadata and a statically redistributable runtime breaches or leaves no credible margin under 300 MB. Q4_K_M’s theoretical weight payload is roughly **163.0 MB**, leaving about 137 MB for higher-precision tensors, metadata, licenses, and runtime. Gemma’s unusually large vocabulary/embedding share can make real files substantially larger than the simple dense-weight estimate, so only the exact generated GGUF plus stripped per-platform runtime establishes compliance. Do not approve Q8 packaging under this ceiling.

5. **Smallest permissive fallback — `SmolLM2-135M-Instruct` Q4_K_M (105 MB).** The upstream model card states **135M parameters** and **Apache-2.0**. Published GGUF listings give exact named artifacts of **105 MB for `SmolLM2-135M-Instruct.Q4_K_M.gguf`**, approximately **112 MB Q5_K_M**, **145 MB Q8_0**, and **271 MB F16** (the latter three listing values are from the companion conversion listing; exact downloaded bytes should still be recorded in the spike). llama.cpp supports the SmolLM/Llama-family text architecture. All quants leave ample room for a stripped static runtime except F16. Quality risk is **high**: 135M generic instruction tuning may miss multi-file semantic intent, overfit wording, or ignore conflicting custom/style instructions; Q4 adds some degradation. Use it as a licensing/size baseline, not the presumptive winner. [Upstream model card and license](https://huggingface.co/HuggingFaceTB/SmolLM2-135M-Instruct) [GGUF Q4 listing](https://huggingface.co/prithivMLmods/SmolLM2-135M-Instruct-GGUF/blame/main/README.md) [broader GGUF table](https://huggingface.co/bartowski/SmolLM2-135M-Instruct-GGUF)

6. **Qwen2.5-Coder-0.5B-Instruct is a useful quality control, not a shippable candidate (blocker: size).** It is code-specialized, instruction-tuned, llama.cpp-compatible, and upstream **Apache-2.0**, but the published Q4_K_M GGUF is **0.40 GB** and Q5_K_M is **0.42 GB**, before runtime. It cannot satisfy a 300 MB complete installed bundle without an unusually destructive sub-Q4 quant that would itself need strong quality evidence. [Upstream model/license](https://huggingface.co/Qwen/Qwen2.5-Coder-0.5B-Instruct) [GGUF sizes](https://huggingface.co/bartowski/Qwen2.5-Coder-0.5B-Instruct-GGUF)

7. **Runtime choice — llama.cpp remains the smallest credible default (low architecture risk, unresolved exact bundle size).** It directly mmap-loads adjacent GGUF assets, has a quantization toolchain, supports CPU inference and multiple platform backends, and avoids Ollama/a service dependency. For issue #30, build only the minimal text-generation library/CLI surface and strip it; record exact installed bytes separately on each target architecture. Runtime size cannot be stated exactly without fixing build flags, backend, target OS/architecture, and static libraries, so no universal exact number is credible. [llama.cpp repository](https://github.com/ggml-org/llama.cpp) [quantize tool](https://github.com/ggml-org/llama.cpp/blob/master/tools/quantize/README.md)

## Candidate decision table

| Candidate | Parameters | Quantized artifact evidence | License / redistribution | Runtime support | Likely quality risk | Ceiling verdict |
|---|---:|---|---|---|---|---|
| hks350d git-diff Gemma 3 | HF index ~0.4B; base advertised 270M | No exact first-party GGUF byte count exposed in available results; generate Q4_K_M and record bytes | Gemma Terms; redistribution allowed with agreement/restriction/notice flow-down | Gemma 3 in llama.cpp/GGUF | Medium: domain fit good, arbitrary instruction retention unproven | Q4 likely; Q8 blocker/pending exact artifact |
| Gemma 3 270M IT | 270M | Q4 estimate ~163 MB raw; Q8 estimate ~287 MB raw, not artifact sizes | Same custom Gemma Terms | llama.cpp/GGUF | Medium-high without commit fine-tune | Q4 plausible; Q8 no |
| SmolLM2-135M-Instruct | 135M | Q4_K_M 105 MB; Q5_K_M 112 MB; Q8_0 145 MB; F16 271 MB listings | Apache-2.0, notices/license required | llama.cpp/GGUF | High | Q4/Q5/Q8 yes |
| Qwen2.5-Coder-0.5B-Instruct | 0.5B class | Q4_K_M 0.40 GB; Q5_K_M 0.42 GB | Apache-2.0 | llama.cpp/GGUF | Lower semantic risk than 135M, still untested | No |

## Recommended issue #30 spike order

1. Download the hks350d source checkpoint and record every file’s exact byte count and commit SHA.
2. Convert the merged checkpoint to BF16/F16 GGUF, produce Q4_K_M directly, and separately produce Q4_K_M from Q8_0 with `--allow-requantize`; compare exact sizes and the same deterministic test corpus.
3. Test all seven styles, adversarial/custom guidance, body requested/body forbidden, multi-file diffs, renames/deletes, and diff-size refusal. Require machine checks for wrappers/empty output/style syntax, plus human semantic scoring.
4. Run the identical corpus on SmolLM2 Q4_K_M as the permissive-license/size baseline and Qwen2.5-Coder-0.5B Q4 as a non-shipping quality ceiling.
5. Package the stripped runtime, model, tokenizer/metadata, required licenses/notices, and executable; measure **installed bytes**, not archive size or rounded model-page values.

## Sources

- Kept: [hks350d/git-diff-to-commit-gemma-3-270m](https://huggingface.co/hks350d/git-diff-to-commit-gemma-3-270m) — task-specific candidate.
- Kept: [Google: Introducing Gemma 3 270M](https://developers.googleblog.com/en/introducing-gemma-3-270m) — primary parameter count and intended use.
- Kept: [Gemma Terms of Use](https://ai.google.dev/gemma/terms) — primary redistribution terms.
- Kept: [Gemma prohibited-use policy](https://ai.google.dev/gemma/prohibited_use_policy) — incorporated downstream restrictions.
- Kept: [llama.cpp quantize README](https://github.com/ggml-org/llama.cpp/blob/master/tools/quantize/README.md) — primary runtime/tooling evidence.
- Kept: [SmolLM2 upstream card](https://huggingface.co/HuggingFaceTB/SmolLM2-135M-Instruct) — primary parameters/license.
- Kept: [Qwen2.5-Coder upstream card](https://huggingface.co/Qwen/Qwen2.5-Coder-0.5B-Instruct) — primary license/model identity.
- Dropped: TinyLlama 1.1B — even aggressive quantization leaves too little or no runtime margin, and it is less task-aligned than the code-specific control.
- Dropped: Ollama catalog entries — convenient runtime packaging but conflicts with the no-service preference and is not primary exact-file evidence.
- Dropped: SEO/legal commentary on Gemma — primary Google terms are controlling.

## Gaps

- **Blocking evidence gap:** exact file names and byte counts for the hks350d checkpoint/GGUF were not exposed by the available search results. The configured research child requested `fetch_content`/`get_search_content`, but those tools were unavailable; therefore this report does not claim full-content/API inspection succeeded. Resolve by querying the Hugging Face repository tree/LFS metadata or downloading the pinned revision.
- No candidate has published evidence against cnm’s seven-style/custom-guidance/multiline acceptance corpus.
- Exact llama.cpp installed size depends on the selected target and build configuration.
- Gemma obligations above are an engineering reading, not legal advice; counsel/release owner should approve the distribution notice and terms flow-down.
- The date context is 2026-07; repository artifacts and terms must be pinned and archived at the eventual release revision.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "review-findings identify concrete candidate repositories/artifact names and severity; residual-risks enumerate the missing exact hks350d bytes, model-quality proof, runtime size, and Gemma legal review"
    }
  ],
  "changedFiles": [
    "/Users/byte/workspace/projects/commit-now-myfriend/.pi-subagents/artifacts/outputs/e23cdc2d/research.md"
  ],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "read .cs/epics/001-o-offline-commit-flow/spec.md",
      "result": "passed",
      "summary": "Confirmed 300 MB installed ceiling and seven-style/custom/multiline requirements."
    },
    {
      "command": "focused web searches across model repositories, primary licenses, and llama.cpp quantization docs",
      "result": "passed",
      "summary": "Screened Gemma task fine-tune, Gemma base, SmolLM2, Qwen2.5-Coder, licensing, runtime support, and requantization."
    },
    {
      "command": "fetch full source pages / Hugging Face repository API",
      "result": "not-run",
      "summary": "Required fetch_content/get_search_content tools were unavailable in this child configuration."
    }
  ],
  "validationOutput": [
    "Qwen2.5-Coder Q4_K_M is 0.40 GB and fails the 300 MB ceiling before runtime.",
    "SmolLM2-135M-Instruct Q4_K_M listing is 105 MB and Apache-2.0.",
    "Gemma Q8 raw-weight estimate leaves no runtime margin; Q4 must be generated and measured exactly.",
    "No source establishes seven-style/custom-guidance/multiline quality; corpus evaluation remains mandatory."
  ],
  "residualRisks": [
    "blocker: hks350d repository exact artifact byte counts and pinned file paths remain unverified because full-content/API tools were unavailable",
    "high: no candidate is yet quality-attested for seven styles, arbitrary custom guidance, and optional multiline bodies",
    "medium: Gemma custom-license flow-down and prohibited-use obligations need release/legal approval",
    "medium: Q8-to-Q4 requantization may reduce quality versus direct BF16/F16-to-Q4 conversion",
    "medium: exact stripped llama.cpp per-platform installed size has not been built or measured"
  ],
  "noStagedFiles": true,
  "diffSummary": "Read-only model/runtime research; only the required external research artifact was written.",
  "reviewFindings": [
    "blocker: hks350d/git-diff-to-commit-gemma-3-270m - do not select a bundle until exact pinned GGUF bytes are measured",
    "blocker: Gemma 3 270M Q8 - theoretical weights alone consume approximately 287 MB, so complete bundle cannot credibly stay under 300 MB",
    "high: all candidates - seven-style/custom-guidance/multiline behavior lacks empirical evidence",
    "medium: Gemma-derived candidates - redistribution carries custom Gemma Terms obligations",
    "no source-code findings; task was research-only"
  ],
  "manualNotes": "The artifact distinguishes exact published listing sizes from theoretical estimates and explicitly reports the unavailable fetch-tool configuration rather than claiming full-page verification."
}
```
