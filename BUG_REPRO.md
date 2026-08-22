# Bug Reproduction

## Bug

Batch upload cleanup is delayed or skipped, close errors overwrite primary write failures, invalid batches can succeed, and rejected files can become visible.

## Trigger

Run the four targeted upload batch tests from `backend/` with multiple readers and injected write, close, and validation failures.

## Observed Errors

```text
--- FAIL: TestUploadClosesEveryReader
    saved 0 files
--- FAIL: TestUploadKeepsPrimaryWriteFailure
    write error was lost: source close failed
--- FAIL: TestUploadCleansTemporaryBatch
    invalid batch unexpectedly succeeded
--- FAIL: TestUploadDoesNotExposePartialBatch
    batch max=1 want=2
    rejected batch published 3 files
```
