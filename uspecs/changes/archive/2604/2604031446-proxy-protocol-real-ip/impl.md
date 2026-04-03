# Implementation plan: Use proxy protocol to get the real request IP

## Provisioning and configuration

- [x] update: [go.mod](../../../../go.mod): Add go-proxyproto library
  - `go get github.com/pires/go-proxyproto@v0.11.0`

## Technical design

- [x] update: [prod/apps/logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - update: Router section — add `headers` attribute (all request headers as a formatted string) to `withLogAttribs` context attributes for production debugging of real IP propagation

## Construction

### Router changes

- [x] update: [pkg/router/types.go](../../../../pkg/router/types.go)
  - add: `UseProxyProtocol bool` field to `HTTPServerParams`
- [x] update: [pkg/router/consts.go](../../../../pkg/router/consts.go)
  - add: `logAttrib_Headers` constant for the `headers` log attribute key
- [x] update: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)
  - update: `prepareBasicServer` — wrap listener with `proxyproto.Listener` when `UseProxyProtocol` is true (after `net.Listen`, before `LimitListener`)
- [x] update: [pkg/router/utils.go](../../../../pkg/router/utils.go)
  - add: `formatHeaders` helper that formats `http.Header` as a single string
  - update: `withLogAttribs` — add `logAttrib_Headers: formatHeaders(req.Header)` to context attributes

### VVM wiring

- [x] update: [pkg/vvm/types.go](../../../../pkg/vvm/types.go)
  - add: `RouterUseProxyProtocol bool` field to `VVMConfig`
- [x] update: [pkg/vvm/provide.go](../../../../pkg/vvm/provide.go)
  - update: `provideRouterParams` — pass `cfg.RouterUseProxyProtocol` to `HTTPServerParams.UseProxyProtocol`
- [x] Review
