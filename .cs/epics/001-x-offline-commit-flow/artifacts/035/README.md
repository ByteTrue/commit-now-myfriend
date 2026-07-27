# #35 Local Teacher Label Pierce

This artifact freezes the public-data-only teacher-label pilot authorized after #34 stopped on noisy source commit messages.

Current order:

1. freeze a 200-family unlabeled pilot and all exclusions;
2. acquire and hash the single Qwen2.5-Coder-14B-Instruct Q6_K teacher;
3. run one near-limit local smoke;
4. generate one evidence-backed record per case without retries;
5. validate mechanically, then obtain two independent blind scores per record;
6. GO only if every critical-error and quality threshold in GitHub #35 passes.

The teacher is build tooling, not a shipped dependency. The 0.5B student and Git workflow remain blocked.
