# Package httpu

HTTP client utility with configurable options and automatic retry
logic for robust network communication.

## Problem

Standard HTTP clients lack built-in retry mechanisms and flexible
configuration options needed for reliable service-to-service
communication in distributed systems.

## Features

- **Configurable requests** - Headers, cookies, methods, and auth
- **Automatic retries** - Built-in retry logic for network failures
- **Status code handling** - Expected codes and custom error handling
- **Response processing** - Custom handlers and body management
- **Connection management** - Idle connection cleanup and timeouts
- **Windows error handling** - WSAE error detection and handling

## Platform Support

Includes Windows-specific WSAE (Windows Socket API Error) handling
for connection reset and refused errors commonly encountered on
Windows systems.
