# Progress

- Completed read-only review of `/tmp/cnm-train31/build_dataset.py`; confirmed old CommitPackFT path reconstructs one file per family and intentionally drops bodies, while the LCA path aggregates all `mods` into a commit.
- Researched official/primary sources for CommitChronicle, LCA commit-message-generation, CommitBench, CommitPack, and CommitPackFT.
- Recommendation: stream a small, repository-isolated sample from CommitChronicle only; reject CommitPackFT for incomplete shape and CommitBench for CC BY-NC plus bounded-diff completeness.
- Full report written to `/tmp/cnm-34-dataset-research.md` with severity-tagged findings, executable 8k-family plan, sources, gaps, residual risks, and acceptance report.
- Runtime limitation recorded: no shell/`gh`, and requested `fetch_content`/`get_search_content` child tools were unavailable, so GitHub #34 and candidate files could not be directly fetched/sample-validated.
