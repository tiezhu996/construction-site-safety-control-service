# Bug Reproduction

## Bug

The authentication chain accepts expired credentials, panics on malformed tokens, trusts an unsigned role header, changes the user ID type, and authorizes role substrings.

## Trigger

Run the five targeted authentication-state tests from `backend/` with expired and malformed tokens, a forged `X-Role` header, and the role `superadmin`.

## Observed Errors

```text
--- FAIL: TestAuthRejectsExpiredCredential
    expired credential status = 204
--- FAIL: TestAuthMalformedCredentialDoesNotPanic
    malformed credential panicked: runtime error: invalid memory address or nil pointer dereference
--- FAIL: TestAuthIgnoresUnsignedRoleHeader
    unsigned role header status = 204
--- FAIL: TestAuthPreservesNumericUserID
    identity status = 500
--- FAIL: TestRoleRequiresExactSignedValue
    role prefix status = 204
```
