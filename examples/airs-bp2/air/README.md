# Airs BP2 Example

This folder contains an example project for the Airs BP2 application. This example serves as a reference for developers to understand the structure and components of an Airs BP2 project.

# How to make tests run
- install `vpm`:
```bash
go install github.com/voedger/voedger/cmd/vpm@latest
```
- generate ORM for airs-bp2 application in `wasm/orm` dir
```bash
vpm orm
```
- run tests from `wasm/main_test.go`
```bash
go test ./wasm/...
```
# How to build the application
- ensure ORM is generated
- build the app using `vpm`. That will produce `air.var` file with the application
```bash
vpm build
```
