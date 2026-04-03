# How: Use proxy protocol to get the real request IP

## Approach

- Use `github.com/pires/go-proxyproto` library to wrap the `net.Listener` in `pkg/router/impl_http.go` (`httpServer.prepareBasicServer`). After `net.Listen("tcp", ...)` returns, wrap the listener with `proxyproto.Listener` so that `net.Conn.RemoteAddr()` automatically returns the real client IP from the PROXY protocol header
- Add a `UseProxyProtocol` flag to `HTTPServerParams` in `pkg/router/types.go` so the wrapping is opt-in (only enable when behind a proxy-protocol-capable LB)
- No downstream changes needed: `remoteIP(req.RemoteAddr)` in `pkg/router/utils.go` (`createBusRequest`) already extracts the host from `RemoteAddr`, which will now contain the real client IP
- The same wrapping applies to the HTTPS listener path since `httpsService` reuses the same `prepareBasicServer` method
- Consider whether `pkg/ihttpimpl/impl.go` (the alternative HTTP processor) also needs the same wrapping
- Append all request headers to the router log context under a `headers` attribute in `withLogAttribs` in `pkg/router/utils.go` to verify real IP propagation in production

References:

- [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)
- [pkg/router/types.go](../../../../pkg/router/types.go)
- [pkg/router/utils.go](../../../../pkg/router/utils.go)
- [pkg/ihttpimpl/impl.go](../../../../pkg/ihttpimpl/impl.go)
