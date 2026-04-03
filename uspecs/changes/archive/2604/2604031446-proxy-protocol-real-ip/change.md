---
registered_at: 2026-04-03T14:07:52Z
change_id: 2604031407-proxy-protocol-real-ip
baseline: 5860bbfcd545246903d6a74ffcd40a2162abc828
issue_url: https://untill.atlassian.net/browse/AIR-3510
archived_at: 2026-04-03T14:46:19Z
---

# Change request: Use proxy protocol to get the real request IP

## Why

We are behind Hetzner's load balancer, so the real IP of the request is not available. `http.Request.Host` returns `10.0.0.2`, and `X-Forwarded-For` / `X-Real-IP` headers are empty.

See [issue.md](issue.md) for details.

## What

Use proxy protocol in the router to obtain the real client IP address from the load balancer:

- Enable proxy protocol support in the router to parse the PROXY protocol header
- Extract the real client IP from the proxy protocol header and make it available in request handling
