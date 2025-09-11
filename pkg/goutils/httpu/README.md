# Package httpu

HTTP client utilities with built-in retry logic, flexible request
options, and robust error handling for production applications.

## Problem

Standard HTTP clients require extensive boilerplate for common
patterns like retries, authentication, and response handling.

## Features

- **[Request builder](impl_opts.go#L17)** - Fluent API for HTTP
  requests with method chaining
- **[Automatic retries](impl.go#L112)** - Built-in retry logic for
  network failures and 503 errors
- **[Response handling](impl_opts.go#L17)** - Customizable response
  processors and body management
- **[Authentication](impl_opts.go#L81)** - Bearer token support with
  flexible authorization options
- **[Status validation](impl.go#L148)** - Expected status code
  checking with detailed error reporting
- **[Connection management](provide.go#L17)** - TCP connection tuning
  with proper cleanup
- **[Request options](impl_opts.go#L58)** - Headers, cookies, and
  method configuration
- **[Body streaming](types.go#L36)** - Support for both string and
  io.Reader request bodies
- **[Error matching](impl_opts.go#L133)** - Custom retry conditions
  based on error patterns
- **[Platform utilities](utils.go#L40)** - Windows socket error
  detection and handling

## Use

See [example usage](example_test.go)
