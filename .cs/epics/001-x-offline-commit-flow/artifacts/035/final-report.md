# cnm #35 Final Report: STOP

**Status:** STOP  
**Date:** 2026-07-25  
**Decision:** Formal STOP, create #36 for improvements

## Summary

The 14B teacher model (Qwen2.5-Coder-14B-Instruct Q6_K) successfully generated all 200 labels for the pilot dataset, but mechanical validation failed at 53% pass rate (106/200). This does not meet the 100% validation requirement for proceeding to audit preparation.

## What Passed

✅ **Label Generation (200/200)**
- All 200 pilot cases labeled successfully
- Average latency: ~11 seconds per case
- No timeouts or server errors
- All requests completed with finish_reason="stop"

✅ **Smoke Test**
- Near-limit case (index 180, 8182 tokens) processed correctly
- Server RSS: 14.1 GB (within 40 GB limit)
- Memory: 61% free (above 10% threshold)
- Pageout/swapout delta: 0
- Latency: 26.7 seconds (within 300 second limit)

✅ **Infrastructure**
- Freeze protocol executed correctly (48 files frozen)
- Environment verification passed (31 packages, 89% memory free)
- Deterministic pilot build verified

## What Failed

❌ **Mechanical Validation (106/200 = 53%)**

Error breakdown:
- **77 cases (38.5%):** `evidence_not_exact_changed_line` — teacher cited code not in changed lines
- **38 cases (19%):** `duplicate_evidence` — subject and body evidence overlap
- **1 case (0.5%):** `invalid_json` — truncated JSON output (index 177)

## Root Cause Analysis

**Evidence Hallucination (77 cases)**

The teacher model generates evidence strings that do not appear in the diff's changed lines. Example from index 2:
- Teacher cited: `'const error = require('./error')'`
- Actual changed lines: only `'const utils = require('../utils')'`

The `evidence_lines()` function correctly extracts only `+`/`-` lines and structural metadata. The model is either:
1. Inferring from context lines (not allowed)
2. Hallucinating code that doesn't exist
3. Paraphrasing rather than exact quoting

**Evidence Duplication (38 cases)**

Subject and body evidence fields contain overlapping strings. The validator requires all evidence to be unique across both fields. This suggests the model doesn't properly separate subject-level from body-level evidence.

**JSON Truncation (1 case)**

Index 177 hit max_tokens limit, producing incomplete JSON. This is a minor issue that could be fixed by increasing max_tokens or adding retry logic.

## Why This is a STOP Condition

The `prepare_audit.py` script requires:
```python
if validation["status"] != "pass" or validation["cases"] != 200:
    raise SystemExit("mechanical validation has not passed all 200 cases")
```

With 94 invalid cases, the audit preparation cannot proceed. The teacher labels cannot be trusted for blind review.

## Technical Details

**Model:** Qwen2.5-Coder-14B-Instruct Q6_K  
**Hardware:** Apple M5 Pro, 128GB unified memory  
**Runtime:** llama.cpp server (PID 5555)  
**Total Runtime:** ~36 minutes for 200 cases  
**Average Latency:** 10.8 seconds per case  
**Peak Memory:** 14.1 GB RSS  

**Validation Script:** `validate_labels.py`  
**Evidence Extraction:** `evidence_lines()` from `leakage_v2.py`  
**Validation Rules:**
- Evidence must be exact substring of changed lines
- All evidence must be unique (no duplicates)
- JSON must parse without errors

## Artifacts

- Labels: `/tmp/cnm-pierce35/labels/000.json` through `199.json`
- Validation: `/tmp/cnm-pierce35/labels/validation.json`
- Smoke test: `/tmp/cnm-pierce35/logs/teacher-smoke.json`
- Server logs: `/tmp/cnm-pierce35/logs/teacher-server-full.log`
- Provenance: `/tmp/cnm-pierce35/logs/teacher-server-full-provenance.json`

## Recommendations for #36

1. **Improve Teacher Prompt**
   - Add explicit examples of valid/invalid evidence
   - Emphasize "copy exact text from +/- lines only"
   - Add negative examples showing context lines are not allowed

2. **Add Evidence Validation to Generation**
   - Post-process teacher output to filter invalid evidence
   - Re-prompt for cases with evidence errors
   - Implement retry logic for truncated JSON

3. **Consider Alternative Models**
   - Test Qwen2.5-Coder-32B-Instruct (larger context window)
   - Test DeepSeek-Coder-V2-Instruct (better code understanding)
   - Test specialized commit message models

4. **Relax Validation Rules**
   - Allow evidence to be "close matches" (fuzzy matching)
   - Permit limited evidence overlap between subject/body
   - Accept partial validation (e.g., 90% pass rate)

5. **Hybrid Approach**
   - Use 14B model for initial labeling
   - Human review for evidence errors
   - Smaller model for bulk generation after validation

## Conclusion

The 14B teacher model demonstrates capability to generate commit message labels but lacks the precision required for evidence citation. The 53% validation pass rate indicates systematic issues with evidence hallucination and duplication.

This is a formal STOP. Issue #36 should explore the recommendations above to achieve 100% mechanical validation pass rate before proceeding to audit.
