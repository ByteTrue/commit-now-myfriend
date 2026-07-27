## Review

- **Correct:** The vertical pierce is appropriately isolated from the rewrite. `.cs/epics/001-o-offline-commit-flow/spec.md:152-155` and GitHub #30’s **Exclude** section prevent premature replacement of the existing CLI and provider architecture.
- **Correct:** GitHub #30 orders work around explicit stop conditions and requires raw failed outputs rather than hiding them. The staged-only, unstaged, cancellation, and drift paths cover the main product branches.
- **Correct:** The 300 MB installed ceiling is consistent between GitHub #30 and `.cs/epics/001-o-offline-commit-flow/spec.md:121-124`.

- **Blocker:** `.cs/epics/001-o-offline-commit-flow/spec.md:39-44,110,129-132` promises that committed content exactly matches the generation snapshot while also using ordinary `git commit` so hooks run. A recheck before `git add -A` or `git commit` leaves a TOCTOU window, and a pre-commit hook can modify the index after the recheck. GitHub #30’s drift test only says to mutate input “after generation”; that can pass without proving check-to-commit atomicity or hook behavior. Before execution, define and test the invariant using tree OIDs, including:
  - mutation after the final recheck but before staging/commit;
  - a hook that modifies the index;
  - the expected abort/restoration behavior when a hook fails or changes content.

- **Blocker:** The quality gate is vulnerable to a lucky or cherry-picked run. GitHub #30 requires a “fixed corpus” and says outputs must not “frequently” violate requirements, but it defines neither the corpus manifest, decoding parameters, repeat count, nor pass threshold. A single favorable generation could satisfy the checklist without proving reliability. Pin fixture IDs/hashes, prompt envelope, model revision/hash, runtime version, seed/temperature, exact command, number of repetitions, and a predeclared acceptance rubric. Retain every attempted output.

- **Blocker:** Context capacity is not acceptance-gated. `.cs/epics/001-o-offline-commit-flow/spec.md:52` requires explicit refusal rather than silent truncation, but GitHub #30 only calls for small/medium diffs and has no at-limit or over-limit test. Consequently, a candidate that handles only trivial diffs—or silently omits part of a large or untracked file—could pass. Establish a minimally useful supported input budget, verify complete tracked/untracked representation, and add boundary and oversize refusal cases.

- **Note:** Several GitHub #30 criteria are not independently reproducible:
  - “Complete installed bundle” lacks an accounting boundary and manifest. State whether it includes executable, runtime libraries, model, tokenizer, licenses/notices, and distribution wrapper.
  - “Cold” and “warm” are undefined. Specify process/model-cache state and whether warm means a second generation in the same process.
  - Offline proof does not specify network denial or cache isolation. Run the assembled bundle with an empty application cache/home and denied egress, then record the command or trace.
  - “No material invention,” “substantive editing,” and “where practical” peak memory remain subjective. Give the maintainer a per-sample attestation table; either require peak memory with a named measurement tool or cut it from acceptance.

- **Note — scope cuts before execution:**
  1. Do not build the confirmation/Git skeleton until license, raw artifact size, and model-quality gates pass. A throwaway runner is sufficient for those gates.
  2. Do not exercise every message style through every Git branch. Test styles in the deterministic model corpus; test staged/unstaged/cancel/drift once through the Git harness.
  3. Do not implement Git Config precedence, production CLI parsing, npm packaging, signing, or cross-platform integration in this pierce. Pass style and guidance directly to the runner and assemble one macOS release-shaped directory.
  4. Treat arbitrary conflicting Custom Guidance as out of scope; test a fixed representative set subordinate to the output safety/format guards.

### Recommended first candidate and run sequence

1. **Candidate:** Start with the named Gemma 3 270M instruction-tuned/commit-message candidate using one pinned GGUF quantization and a pinned `llama.cpp` revision. Do not treat it as selected until both Gemma/checkpoint redistribution terms and runtime obligations are recorded.
2. **Cheap gates first:** Record exact artifact hashes and licenses; place executable, runtime assets, tokenizer/model, and notices in the release-shaped directory; measure installed and compressed size.
3. **Deterministic smoke:** One small diff, fixed decoding, each required output shape: plain, constrained prefix, custom guidance, and multiline.
4. **Predeclared corpus:** Run the complete style/history/guidance matrix and all repetitions without changing prompts; retain raw inputs, outputs, checks, and failures. Stop if the agreed quality threshold fails.
5. **Capacity gate:** Run exact-boundary and over-budget inputs, including tracked deletion and untracked content; prove there is no silent truncation and oversize input is rejected.
6. **Only then build the Git harness:** Prove staged-only, tracked-plus-untracked, cancellation, drift, check-to-commit race, and hook mutation/failure using before/after index and tree OIDs.
7. **Final delivery measurements:** Run the assembled bundle with network denied and empty caches; measure defined cold/warm latency, generation time, and required memory metric with the machine profile.
8. **Maintainer attestation:** Review the complete sample table and decide candidate pass/fail and the acceptable latency threshold before retaining production code.