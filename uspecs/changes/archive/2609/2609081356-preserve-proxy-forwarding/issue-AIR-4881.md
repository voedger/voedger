# voedger, reverse proxy: migrate to Rewrite and fix drops forwarding headers

- URL: https://untill.atlassian.net/browse/AIR-4881
- ID: AIR-4881
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: [AIR-3883: fix rules for revive linter](https://untill.atlassian.net/browse/AIR-3883) (parent)

## Why

* [httputil.ReverseProxy.Director](https://github.com/host6/voedger/blob/64edb2046f78b8e4c0929805ba4d645d9b2670d5/pkg/router/impl_reverseproxy.go#L43) field is deprecated
* Replacing `httputil.ReverseProxy.Director` with an empty `Rewrite` callback changed header handling: incoming forwarding headers are stripped, and the client IP is no longer appended to `X-Forwarded-For`. Upstream services lose forwarding information, and `TestBasicUsage_ReverseProxy` fails.

### Why did worked with Director?

The empty `Director` callback did nothing, but **Go’s** `ReverseProxy` **added** `X-Forwarded-For` **automatically after calling it**. 

For ordinary HTTP requests:

| Header | Empty `Director` | Empty `Rewrite` |
| --- | --- | --- |
| `X-Forwarded-For` | Creates it or appends the client IP | Removes it |
| `X-Forwarded-Host` | Preserves incoming value | Removes it |
| `X-Forwarded-Proto` | Preserves incoming value | Removes it |
| `Forwarded` | Preserves incoming value | Removes it |

## What

* use `Rewrite` instead of `Director`
* Restore forwarding-header preservation in both proxy implementations (`router` and `ihttp`)
* Append the client IP to `X-Forwarded-For`, preserving the existing chain.
* Preserve incoming `X-Forwarded-Host`, `X-Forwarded-Proto`, and `Forwarded` headers.
* Add regression coverage and ensure subtest assertions report failures against the correct subtest.


