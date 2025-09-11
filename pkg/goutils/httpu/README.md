# httpu

HTTP client utilities with configurable retry logic, authentication,
and response handling for robust HTTP communication.

## Problem

Standard HTTP clients require extensive boilerplate for common tasks
like retry logic, authentication, and response validation.

## Features

- **Retry logic** - Automatic retries with exponential backoff
- **Authentication** - Built-in Bearer token and cookie support
- **Response handling** - Configurable response processing and validation
- **Status validation** - Expected status code checking with helpers
- **Connection management** - Proper connection cleanup and linger settings
- **Long polling** - Support for long-running HTTP requests
- **Error matching** - Custom retry conditions based on error types
- **Platform handling** - Windows socket error handling (WSAE errors)

## Platform Support

Includes Windows-specific socket error handling for WSAECONNRESET and
WSAECONNREFUSED errors with automatic retry logic.

## Usage Example

See [example usage](example_test.go)
