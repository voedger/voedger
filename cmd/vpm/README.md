# VPM (Voedger Package Manager)

The Voedger Package Manager (VPM) functions similarly to Go/NPM but is specifically designed for Voedger. Key commands include:

- `vpm compile`
- `vpm build`
- `vpm mod init`

## Principles

- **Go Installation Requirement**: To use VPM, Go must be installed.
    - **Rationale**: This approach simplifies the development process.
- **Integration with `go.mod`**: VPM commands like `compile` and `build` utilize `go.mod`.
- **Project Initialization**: The `vpm mod init` command generates a `main.go` file that incorporates the latest Voedger repository.
- **Underlying Use of Go Modules**: `go mod init` is leveraged in the background for effective module management.

## Commands

- [`compile`](./README-compile.md): For detailed instructions and information on the compile command.

## Technical Design

- Integrated tests are the first (`execRootCmd()` function is used)

