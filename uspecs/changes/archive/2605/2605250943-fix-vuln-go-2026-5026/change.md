---
registered_at: 2026-05-25T09:40:36Z
change_id: 2605250940-fix-vuln-go-2026-5026
type: fix
baseline: 4344f6ebd11c8a3cdf7509027b56717ce5982fed
issue_url: https://untill.atlassian.net/browse/AIR-4036
archived_at: 2026-05-25T09:43:29Z
---

# Change request: Fix vulnerability GO-2026-5026 in golang.org/x/net

## Why

Vulnerability GO-2026-5026 affects `golang.org/x/net@v0.54.0`, currently used by the project. The flaw is a failure to reject ASCII-only Punycode-encoded labels in `golang.org/x/net/idna`, reachable from `router.Provide` via `autocert.HostWhitelist` → `idna.Profile.ToASCII`. The fix is available in `golang.org/x/net@v0.55.0`.

## What

Upgrade the affected dependency to a version that contains the patch:

- Bump `golang.org/x/net` to `v0.55.0` or later
