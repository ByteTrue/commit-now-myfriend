# #34 Blind Data Audit Rubric

The auditor receives a frozen slice and must not inspect training scripts, other splits, or prior scores.

## Base complete-commit rows

Emit one JSON object per input row with exactly:

```json
{"family":"...","critical_error":false,"fully_grounded":true,"subject_quality":2,"body_quality":2,"reason":"short factual reason"}
```

- `critical_error`: secret/credential/PII, corrupted diff, materially wrong label, or unsupported high-impact/security claim.
- `fully_grounded`: every subject/body claim is supported by the complete diff. Directly implied intent is allowed; invented behavior is not.
- `subject_quality`, `body_quality`: `2` correct/useful, `1` partial/noisy, `0` wrong/unusable. An empty body is `2` when the subject is sufficient.
- Do not reward repository popularity or match wording to external commit history.

Acceptance across the frozen 200-row sample: zero critical errors and at least 95% fully grounded.

## Deterministic guidance rows

Emit one JSON object per input row with exactly:

```json
{"id":"...","critical_error":false,"label_correct":true,"fully_grounded":true,"reason":"short factual reason"}
```

- `label_correct`: assistant JSON and rendered intent exactly satisfy the declared guidance transformation (reference, body prohibition, exactly two bullets, retained body, or requested prefix).
- `fully_grounded`: semantic content comes from the source diff/body; text explicitly requested by guidance (for example `#123` or `SECURITY:`) is authorized and not a hallucination.
- `critical_error`: secret/PII, corrupted diff, malformed label, or unsupported high-impact claim not authorized by guidance.

Acceptance across the frozen 100-row sample: zero critical errors, 100% label correctness, and at least 95% fully grounded.

Auditors write only JSONL scores; they must not modify the corpus.
