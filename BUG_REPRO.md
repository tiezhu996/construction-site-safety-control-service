# Bug Reproduction

## Bug

Concurrent inspection-item batches can return before workers finish, report missing updates as successful, reuse result keys, and race in the HTTP handler.

## Trigger

Run the four targeted inspection-item batch tests from `backend/` with `-race`. They start concurrent workers, inject a missing row, and issue overlapping batch requests.

## Observed Errors

```text
--- FAIL: TestItemBatchWaitsForAllWorkers
    batch returned before workers completed: 0 results
--- FAIL: TestItemBatchErrorPathDoesNotHang
    missing item update was reported as successful
--- FAIL: TestItemBatchChannelClosesOnce
    two batch results share key "8"
WARNING: DATA RACE
--- FAIL: TestItemBatchConcurrentUpdatesRaceFree
    race detected during execution of test
```
