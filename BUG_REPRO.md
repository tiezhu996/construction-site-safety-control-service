# Bug Reproduction

## Bug

The safety-inspection execution state machine disagrees across partial execution, completion, terminal-state protection, and report visibility.

## Trigger

Run the four targeted inspection execution tests from `backend/`. They execute one plan in stages, attempt a stale reopen, and request its terminal report.

## Observed Errors

```text
--- FAIL: TestInspectionPartialMovesInProgress
    partial execution status = "completed"
--- FAIL: TestInspectionCompletionFromProgress
    code=40901 message=SafetyInspection[id=1] execute invalid transition=in_progress->completed
--- FAIL: TestInspectionTerminalCannotReopen
    stale scheduled state reopened a completed inspection
--- FAIL: TestInspectionReportVisibleAtTerminal
    completed report status = 409
```
