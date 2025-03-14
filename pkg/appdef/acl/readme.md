# ACL Package

The `acl` package provides functionality for Access Control Lists (ACL) in the `voedger` application framework. It includes methods to check permissions for various operations on resources within a workspace.

## Functions

### IsOperationAllowed

```go
func IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, rol []appdef.QName) (bool, error)
```

Checks if a specified operation is allowed on a specified resource for any of the specified roles within a workspace.

### PublishedTypes

```go
func PublishedTypes(ws appdef.IWorkspace, role appdef.QName) iter.Seq2[appdef.IType, iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]
```

Lists the resources allowed to the published role in the workspace and its ancestors. Types are enumerated in alphabetical order.

### RecursiveRoleAncestors

```go
func RecursiveRoleAncestors(role appdef.IRole, ws appdef.IWorkspace) (roles appdef.QNames)
```

Returns a recursive list of role ancestors for a specified role within a workspace. The result is an alphabetically sorted list of role names.

## Usage

To use the `acl` package, import it into your Go project and utilize the provided functions to manage and check access control within your application.

```go
import "github.com/voedger/voedger/pkg/appdef/acl"
```

## License

This project is licensed under the terms of the MIT license.
