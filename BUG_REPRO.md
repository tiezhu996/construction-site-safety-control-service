# Bug Reproduction

## Bug

User lookup errors lose their identity across repository, service, and HTTP layers. Missing users are misclassified, duplicate registration cannot reach its conflict path, and profile lookup returns 500 instead of 404.

## Trigger

Run the four targeted user error-classification tests from `backend/` against an empty test database.

## Observed Errors

```text
--- FAIL: TestUserMissingErrorChain
    missing phone must preserve ErrNotFound, got find user by phone: not found
--- FAIL: TestUserDuplicateRegistrationConflict
    first registration failed: User[phone=13900000001] register check failed
--- FAIL: TestUserInvalidLoginUnauthorized
    missing login must be unauthorized
--- FAIL: TestUserLookupNotFoundResponse
    missing profile must be 404/code 40400, got status=500
```
