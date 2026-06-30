---
change_id: 2606301014-reset-password-by-alias
type: rsch
domains: [prod]
---

# Change request: Reset password by login alias

## Why

In the prod domain's authentication context, the reset-password-by-verified-email flow resolves the account only by its primary login, never by a login alias. A user whose alias is itself an email address therefore cannot recover access through that alias, even though the alias uniquely identifies the underlying account. This change researches and enables alias-aware password reset so alias holders are not locked out.

## What

Enable password reset to work when the supplied email is a login alias, not only the primary login:

- A user who enters an alias-email receives the verification code at that alias-email
- After verification, the password of the account that owns the alias is updated, and the user can sign in with the new password
- The reset continues to work unchanged for a primary login email
- A reset attempt using a previously assigned or cleared alias is rejected
- The verification step remains the sole proof of ownership; resolving an alias never lets an unverified value alter another account's password
