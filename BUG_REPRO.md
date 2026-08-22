# Bug Reproduction

## Bug

Safety incident reads and rectification do not follow the current request context. A canceled request can succeed, while a later fresh request can inherit an earlier cancellation.

## Trigger

Run the four targeted incident context tests from `backend/`. They cancel HTTP and repository contexts, then make a fresh request through the same service path.

## Observed Errors

```text
--- FAIL: TestIncidentCanceledQueryStops
    canceled query must stop, got <nil>
--- FAIL: TestIncidentFreshRequestSurvivesOldCancel
    fresh request inherited old cancellation: find safety incident by id: context canceled
--- FAIL: TestIncidentHandlerPassesRequestContext
    canceled HTTP request reached a successful query
--- FAIL: TestIncidentRectifyHonorsDeadline
    canceled rectification must fail
```
