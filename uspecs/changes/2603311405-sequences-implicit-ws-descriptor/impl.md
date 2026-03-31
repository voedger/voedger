# Implementation plan: Sequences: enforce implicit workspace descriptor

## Construction

- [x] update: [pkg/parser/impl.go](../../../pkg/parser/impl.go)
  - add: Create implicit `WsDescriptorStmt` with default name `<WorkspaceName>Descriptor` when workspace is non-abstract, non-pool, and has no explicit descriptor statement

- [x] update: [pkg/parser/impl_test.go](../../../pkg/parser/impl_test.go)
  - add: `Test_ImplicitWorkspaceDescriptor` with subtests for non-abstract, abstract, explicit descriptor, and name conflict scenarios

- [x] update: [pkg/appdefcompat/testdata/sys.old.vsql](../../../pkg/appdefcompat/testdata/sys.old.vsql)
  - add: `DescTestWs` workspace with explicit `OldDescriptor`
  - add: `ImplicitDescWs` workspace with no explicit descriptor (gets implicit `ImplicitDescWsDescriptor`)

- [x] update: [pkg/appdefcompat/testdata/sys.new.vsql](../../../pkg/appdefcompat/testdata/sys.new.vsql)
  - add: `DescTestWs` workspace with renamed `RenamedDescriptor` (triggers `ValueChanged`)
  - add: `ImplicitDescWs` workspace with explicit `CustomDescriptor` replacing the implicit one (triggers `ValueChanged`)

- [x] update: [pkg/appdefcompat/impl_test.go](../../../pkg/appdefcompat/impl_test.go)
  - add: Expected errors in `Test_Basic` for descriptor name changes (`NodeRemoved` for old descriptor types, `ValueChanged` on `Descriptor` nodes)
