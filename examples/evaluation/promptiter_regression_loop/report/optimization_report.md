# Optimization Report

## Decision

Accepted: false

Reason: newly failed cases: validation_overfit_guard; critical case regressed: validation_overfit_guard

## Baseline vs Candidate

| Name | Score | Passed | Failed | Total |
| --- | ---: | ---: | ---: | ---: |
| baseline | 0.3333 | 1 | 2 | 3 |
| candidate | 0.5333 | 1 | 2 | 3 |

## Case Delta

| Case | Type | Baseline | Candidate | Delta |
| --- | --- | ---: | ---: | ---: |
| validation_no_effect | unchanged | 0.0000 | 0.0000 | 0.0000 |
| validation_overfit_guard | newly_failed | 1.0000 | 0.6000 | -0.4000 |
| validation_prompt_fixable | newly_passed | 0.0000 | 1.0000 | 1.0000 |

## Failure Attribution

| Case | Category | Reason |
| --- | --- | --- |
| validation_prompt_fixable | final_response_mismatch | final response mismatch |
| validation_no_effect | tool_arg_error | arguments mismatch: expected V-200 |

## Cost / Latency

- API calls: 12
- Cost: 0.0000
- Latency: 1 ms
