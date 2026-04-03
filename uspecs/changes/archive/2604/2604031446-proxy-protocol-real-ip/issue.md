# AIR-3510: voedger: try proxy protocol to get the real request IP

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Why

We're behind Hetzner's load balancer so we do not have the real IP of the request. `http.Request.Host` is `10.0.0.2`, no `X-Forwarded-For`, `X-Real-IP` headers are empty.

## What

Use proxy protocol in router.

