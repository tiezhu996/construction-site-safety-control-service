# Bug Reproduction

## Bug

Asynchronous audit delivery loses error identity and stable detail snapshots. Queue saturation and database failures cannot be classified, while the public list reports a healthy state.

## Trigger

Run the four targeted audit delivery tests from `backend/` with a full queue, a database failure, and a detail map mutated after enqueue.

## Observed Errors

```text
--- FAIL: TestAuditQueueFullErrorClassified
    queue error is not classifiable: audit delivery: audit queue full
--- FAIL: TestAuditDatabaseErrorUnwraps
    database error chain = audit database write failed: database offline
--- FAIL: TestAuditDetailSurvivesAsyncWrite
    stored detail = {"decision":"overwritten"}
--- FAIL: TestAuditListReportsDegradedState
    status = 200, delivery_degraded=false
```
