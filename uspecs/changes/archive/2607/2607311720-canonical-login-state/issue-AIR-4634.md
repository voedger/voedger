# Add enable/disable state for canonical login

- URL: https://untill.atlassian.net/browse/AIR-4634
- ID: AIR-4634
- State: To Do
- Author: Maksim Geraskin
- Labels: none

## Goal

Introduce a reversible enable/disable state for a canonical login identifier.

This state must be separate from `sys.IsActive`: disabling the canonical login must retain the login identity and keep the canonical identifier reserved without affecting its active alias or other login operations.

## Required behavior

* Provide System-authorized operations to disable and enable a canonical login.
* A disabled canonical login cannot issue a principal token when the canonical identifier is submitted.
* A disabled canonical login cannot initiate password reset when the canonical identifier is submitted.
* Sign-in and password reset through an active login alias remain available while the canonical login is disabled.
* Password change, password-reset completion, and principal-token validation and renewal are not affected by canonical login disablement.
* Public canonical sign-in and recovery-initiation endpoints must not disclose that the canonical login is disabled; use the same externally observable failure as for an unknown login or invalid credentials where applicable.
* Creating another login with the disabled canonical identifier remains a conflict.
* Enable and disable operations are idempotent.

## Functional design

Update the authentication scenarios in `uspecs/specs/prod/auth/authn.feature`.

The scenarios should cover at least:

* disabling and re-enabling a canonical login;
* rejection of principal-token issue through the disabled canonical identifier;
* rejection of password-reset initiation through the disabled canonical identifier;
* continued sign-in and password reset through an active alias;
* completion of a password reset initiated before canonical login disablement;
* duplicate login creation while the canonical identifier is disabled;
* restoration of the two canonical entry operations after re-enabling.
