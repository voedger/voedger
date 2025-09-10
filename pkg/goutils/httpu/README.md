# httpu

HTTP client utility with configurable request options, automatic
retry logic, and robust error handling.

## Problem

Standard HTTP clients require repetitive boilerplate for common
patterns like retries, authentication, and response handling.

## Features

- **Request options** - Configurable headers, cookies, methods
- **Automatic retries** - Built-in retry logic for network errors
- **Response handling** - Custom response processors and validators
- **Authentication** - Bearer token and authorization helpers
- **Error recovery** - Windows socket error detection and handling
- **Concurrent safe** - Thread-safe client operations
- **Timeout control** - Request timeout and retry duration limits

## Platform Support

Includes Windows-specific socket error handling (WSAECONNRESET,
WSAECONNREFUSED) for improved network resilience on Windows systems.

## Usage Example

```go
client, cleanup := NewIHTTPClient()
defer cleanup()

resp, err := client.Req(context.Background(), 
    "https://api.example.com/data", 
    "request body",
    WithMethod("POST"),
    WithHeaders("Content-Type", "application/json"),
    WithAuthorizeBy("your-token"),
    WithExpectedCode(201))
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Body)
```
