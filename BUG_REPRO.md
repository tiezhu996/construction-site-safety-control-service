# Bug Reproduction

## Bug

Concurrent requests race while updating rate-limit buckets. Active buckets can be removed, excess requests can pass, and one router can start duplicate cleanup workers.

## Trigger

Run the four targeted limiter tests from `backend/` with the race detector. They issue simultaneous requests against the same limiter and exercise cleanup and router isolation.

## Observed Errors

```text
WARNING: DATA RACE
--- FAIL: TestLimiterCleanupKeepsActiveBucket
    active bucket was pruned, count=0
--- FAIL: TestLimiterRejectsAfterCapacity
    accepted=8 want=2
--- FAIL: TestLimiterParallelIsolation
    router started 2 limiter cleaners
```
