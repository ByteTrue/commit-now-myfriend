# #36 final report — mechanically valid labels fail semantic review

**Decision:** STOP

## What passed

- Post-processing and one re-generation produced 200/200 mechanically valid records.
- Both fresh-context reviewers completed all 200 frozen, identity-blind cases.
- The normalized frozen prompt was restored; every file referenced by the current freeze manifest matches its recorded size and SHA-256.

## Double-blind result

| Gate | Required | Actual |
|---|---:|---:|
| Critical errors from either reviewer | 0 | 29 unique cases |
| Fully grounded by both | >=190 | 190 |
| Subject quality 2 by both | >=180 | 138 |
| All required bodies useful by both | 60/60 | 58/60 |
| Required body quality 2 by both | >=54/60 | 48/60 |
| Mechanical validation | 200/200 | 200/200 |

Reviewer A marked 26 critical errors; reviewer B marked 10; both marked 7 of the same cases critical. Representative failures include:

- index 3: claims `writeStreamFrameHeader` returns header length, while the diff uses the return as permitted payload length;
- index 25: claims a designer file was deleted although only project inclusion was removed;
- index 33: claims an implementation was added when the complete diff only adds project references;
- several records describe a secondary removal/config edit while omitting the dominant functional addition or fix.

Mechanical evidence validity therefore did not establish semantic label quality.

## Evidence integrity caveats

- The original #35 gate merger could not finish because `/tmp/cnm-pierce35/logs/over-limit-entrypoint.json` is absent. This does not change the semantic STOP: the zero-critical gate already fails.
- The #36 candidate is not a reproducible production labeling pipeline: four records received case-specific manual evidence edits and one record was regenerated after increasing `max_tokens` from 384 to 1024.
- A later `/tmp/cnm-pierce37` prompt-redesign run was started after scores were visible, omitted `BODY_POLICY`, recorded stale server PIDs, and stopped at 195/200. It is excluded from all evidence and was not reviewed.

## Persisted artifacts

- `semantic-gate.json` — joint counts, failed checks, critical indices, run IDs, and score hashes.
- `audit/scores-a/` and `audit/scores-b/` — 400 independent score rows over the proper frozen 10×20 slices.
- `audit/review-input-manifest.json` — frozen slice and validation hashes.
- `audit/reviewer-provenance.json` — disjoint fresh-context reviewer runs.

## Conclusion

#36 proved that deterministic evidence cleanup can make schema/evidence checks pass, but it cannot repair wrong or incomplete semantic targets. These labels must not be used to train the 0.5B student. No full-corpus labeling, student training, or Git workflow rewrite may start from this candidate.
