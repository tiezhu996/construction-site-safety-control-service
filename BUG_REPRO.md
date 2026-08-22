# Bug Reproduction

## Bug

Training participant slices share backing arrays across snapshots, filters, repository cache reads, and HTTP responses. Later mutations corrupt previously returned data.

## Trigger

Run the four targeted training snapshot tests from `backend/`. They retain an earlier slice, mutate a later result, and read the original snapshot again.

## Observed Errors

```text
--- FAIL: TestTrainingSnapshotStableAfterRecord
    old snapshot changed to u7,u8
--- FAIL: TestTrainingFilterDoesNotMutateSource
    source changed to [crew-a crew-b crew-b]
--- FAIL: TestTrainingRepositoryReturnsIndependentSlice
    repository exposed cached slice: [changed u2 u3]
--- FAIL: TestTrainingHandlerResponsesDoNotAlias
    handler returned contaminated slice
```
