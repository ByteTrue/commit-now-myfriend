# cnm #35 independent blind-review instructions

Review only the assigned frozen slice. Its rows expose an opaque `index`, complete diff, body policy, teacher target, and exact evidence. Repository identity, commit SHA, source message, teacher identity, latency, and the other reviewer’s scores are withheld.

For every row, emit exactly one JSON object with these keys:

- `index`: unchanged opaque index
- `reviewer`: the assigned literal `A` or `B`
- `input_slice_sha256`: the assigned frozen slice SHA-256
- `critical_error`: boolean; true for a material contradiction, invented primary behavior, wrong dominant change, or unsupported required-body claim
- `fully_grounded`: boolean; every material target claim is supported by the complete diff
- `subject_quality`: integer 0–2
  - 2: accurate, specific dominant change, concise and usable
  - 1: grounded but vague, incomplete, or secondary-focus
  - 0: misleading or unusable
- `body_quality`: `null` when body is optional and empty; otherwise integer 0–2
  - 2: useful rationale/detail supported by the diff
  - 1: grounded but weak, redundant, or minimally useful
  - 0: unsupported, contradictory, or useless where a body is required
- `evidence_quality`: integer 0–2; 2 only when snippets directly support all material target claims
- `reason`: one concise evidence-based explanation

Do not repair labels, adjudicate disagreements, use outside sources, search for the original commit, infer repository identity, inspect another slice, or inspect another reviewer’s output. Complete all 20 rows independently. The merge gate binds every score file to this reviewer, this input slice hash, and a distinct fresh-context subagent run.
