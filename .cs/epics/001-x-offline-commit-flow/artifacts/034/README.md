# cnm #34 — 0.5B structured commit model pierce

This directory contains the small, persistent control plane for GitHub #34:

- frozen public manifests and hashes;
- one-off data/evaluation/training scripts;
- sanitized hardware and size evidence;
- final report.

Raw licensed training rows, private shadow cases, model weights, adapters, and large logs are not committed. They live under `/tmp/cnm-pierce34/` during the run and are referenced by SHA-256 plus reproducible acquisition commands.

The public 26-case corpus from #31–#33 is regression-only. Final GO requires the independent shadow/historical gate whose raw cases remain unread by the training operator until the selected clean-retrained candidate is frozen.

No file in this directory is part of the shipped `cnm` runtime.

## Current gate state

- Specification re-review: PASS; the prior six blockers are closed.
- Pre-training macOS size skeleton: PASS at `577,907,938` projected bytes including a 32 MB growth reserve (`122,092,062` bytes headroom).
- Independent shadow/historical gate: frozen and hash-verified; private raw cases remain unread by the training operator.
- Complete-commit data gate: **STOP**. First blind audit stopped at 4 critical errors in 160 rows; the one permitted redesign improved groundedness to 157/160 but still had 1 critical unsupported label. No model output, smoke, or training was run.

## Reproduction

```bash
source /tmp/cnm-train31/venv/bin/activate
cd .cs/epics/001-o-offline-commit-flow/artifacts/034/scripts
python test_pipeline.py
python build_size_skeleton.py
```
