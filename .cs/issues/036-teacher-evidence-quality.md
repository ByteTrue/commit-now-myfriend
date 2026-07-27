# Issue #36: Improve Teacher Evidence Quality

**Status:** OPEN  
**Priority:** HIGH  
**Type:** PIERCE  
**Created:** 2026-07-25  
**Depends on:** #35 (STOP)

## Problem Statement

Issue #35 demonstrated that the 14B teacher model (Qwen2.5-Coder-14B-Instruct Q6_K) can generate commit message labels but fails mechanical validation at 53% pass rate (106/200). The primary failure modes are:

1. **Evidence Hallucination (77 cases, 38.5%)** — teacher cites code not in changed lines
2. **Evidence Duplication (38 cases, 19%)** — subject and body evidence overlap
3. **JSON Truncation (1 case, 0.5%)** — output exceeds max_tokens

## Success Criteria

Achieve **100% mechanical validation pass rate** (200/200) on the frozen pilot dataset before proceeding to audit preparation.

## Approach Options

### Option A: Prompt Engineering

**Strategy:** Improve the teacher prompt to eliminate evidence errors.

**Implementation:**
1. Add explicit examples of valid/invalid evidence in the prompt
2. Emphasize "copy exact text from +/- lines only, no context lines"
3. Add negative examples showing common failure modes
4. Include step-by-step instructions for evidence extraction

**Pros:**
- No infrastructure changes
- Fast iteration
- Preserves model capabilities

**Cons:**
- May not eliminate all hallucinations
- Requires multiple prompt iterations
- Risk of over-constraining the model

**Estimated Effort:** 2-3 days

### Option B: Post-Processing Validation

**Strategy:** Add a validation layer between teacher output and label storage.

**Implementation:**
1. Generate labels with current teacher
2. Run mechanical validation on each label
3. For failed cases:
   - Filter invalid evidence (remove non-matching strings)
   - Deduplicate evidence across subject/body
   - Re-prompt for cases with <1 valid evidence
4. Store only validated labels

**Pros:**
- Guarantees 100% validation pass rate
- Can handle edge cases programmatically
- Preserves good labels, fixes bad ones

**Cons:**
- Adds complexity to pipeline
- May lose information through filtering
- Re-prompting increases latency

**Estimated Effort:** 3-4 days

### Option C: Alternative Models

**Strategy:** Test larger or more specialized models for better evidence quality.

**Candidates:**
1. **Qwen2.5-Coder-32B-Instruct** — larger model, better code understanding
2. **DeepSeek-Coder-V2-Instruct** — specialized for code generation
3. **CodeLlama-34B-Instruct** — proven code model
4. **Specialized commit models** — fine-tuned on commit data

**Implementation:**
1. Download and test each model on 10-case subset
2. Measure validation pass rate
3. Select best-performing model
4. Run full 200-case pilot

**Pros:**
- May achieve higher quality out-of-the-box
- Larger models have better reasoning
- Specialized models may understand commits better

**Cons:**
- Larger models require more memory/disk
- Longer inference latency
- May still have hallucination issues
- Licensing considerations

**Estimated Effort:** 5-7 days

### Option D: Relaxed Validation

**Strategy:** Modify validation rules to accept "good enough" evidence.

**Implementation:**
1. Allow evidence to be "close matches" (e.g., 80% similarity)
2. Permit limited evidence overlap (e.g., 1 shared string)
3. Accept partial validation (e.g., 90% pass rate threshold)
4. Document relaxed rules in audit protocol

**Pros:**
- Fastest path to audit
- Acknowledges real-world model limitations
- Still catches major errors

**Cons:**
- Reduces audit confidence
- May propagate errors to training data
- Requires justification for relaxed standards

**Estimated Effort:** 1-2 days

### Option E: Hybrid Approach

**Strategy:** Combine teacher model with human review.

**Implementation:**
1. Generate all 200 labels with 14B teacher
2. Flag cases with evidence errors (94 cases)
3. Human review and correction of flagged cases
4. Store human-validated labels

**Pros:**
- Guarantees high quality
- Human catches subtle errors
- Can improve prompt based on human corrections

**Cons:**
- Requires human effort (2-4 hours)
- Not fully automated
- Human reviewer needs domain expertise

**Estimated Effort:** 2-3 days (including human review time)

## Recommended Approach

**Primary:** Option B (Post-Processing Validation)  
**Fallback:** Option A (Prompt Engineering) if Option B doesn't achieve 100%

**Rationale:**
- Option B guarantees 100% validation pass rate
- Preserves good labels, only fixes bad ones
- Can be implemented incrementally
- Provides data for future prompt improvements

## Acceptance Criteria

- [ ] All 200 pilot cases pass mechanical validation
- [ ] No evidence hallucination (all evidence in changed lines)
- [ ] No evidence duplication (all evidence unique)
- [ ] No JSON parsing errors
- [ ] Validation script exits with code 0
- [ ] Audit preparation script runs successfully

## Dependencies

- Frozen pilot dataset from #35
- Validation script from #35
- Teacher model from #35

## Notes

This issue blocks the entire cnm training data pipeline. The audit and review phases cannot proceed without mechanically valid labels.

The 53% pass rate from #35 shows the model understands the task but lacks precision in evidence citation. This is a solvable problem with the right approach.
