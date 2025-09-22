# httpu

HTTP client utilities with built-in retry logic, flexible request
options, and robust error handling for production applications.

## Problem

Standard HTTP clients require extensive boilerplate for common
patterns like retries, authentication, and response handling.

## Features

- **Configurable client** - HTTP client with customizable default options
  - [Client creation: provide.go#L15](provide.go#L15)
  - [TCP linger setting: provide.go#L24](provide.go#L24)
  - [Default options: provide.go#L29](provide.go#L29)

- **Request options** - Flexible configuration system for HTTP requests
  - [Options interface: types.go#L26](types.go#L26)
  - [Option functions: impl_opts.go#L17](impl_opts.go#L17)
  - [Options validation: impl_opts.go#L199](impl_opts.go#L199)

- **Retry logic** - Automatic retry with configurable error matching
  - [Retry configuration: impl.go#L112](impl.go#L112)
  - [Error matchers: impl_opts.go#L133](impl_opts.go#L133)
  - [503 retry handling: impl.go#L132](impl.go#L132)

- **Error handling** - Platform-specific error detection and handling
  - [Windows socket errors: utils.go#L40](utils.go#L40)
  - [Status code errors: errors.go#L14](errors.go#L14)
  - [Default matchers: consts.go#L25](consts.go#L25)

- **Response handling** - Multiple response processing modes
  - [Body reading: utils.go#L17](utils.go#L17)
  - [Response discarding: utils.go#L22](utils.go#L22)
  - [Custom handlers: impl_opts.go#L17](impl_opts.go#L17)

## Platform Support

Includes Windows-specific socket error handling for WSAECONNRESET and 
WSAECONNREFUSED errors to improve retry behavior on Windows systems.

## Use

See [example usage](example_test.go)
