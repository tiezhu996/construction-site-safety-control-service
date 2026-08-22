# Bug Reproduction

## Bug

Certification review mishandles zero-value review state and typed-nil policies. The review can panic, persist an approved state after failure, or surface validation failures as HTTP 500.

## Trigger

Run the four targeted certification review tests from `backend/` with an empty note map, empty evidence, and a failing policy.

## Observed Errors

```text
--- FAIL: TestCertReviewNoNilMapPanic
    review panicked for a zero-value note map: assignment to entry in nil map
--- FAIL: TestCertTypedNilPolicyRejected
    empty evidence returned a non-nil policy of type *model.PhotoReviewPolicy
--- FAIL: TestCertReviewFailureKeepsPending
    failed review persisted status "approved"
--- FAIL: TestCertReviewHandlerReportsValidation
    status = 500
```
