/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Enumeration of ACL operation kinds.
type OperationKind uint8

//go:generate stringer -type=OperationKind -output=stringer_operationkind.go

const (
	OperationKind_null OperationKind = iota

	// # Insert records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are not applicable.
	OperationKind_Insert

	// # Update records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are applicable and specify fields of records or view records that can be updated.
	OperationKind_Update

	// # Select records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are applicable and specify fields of records or view records that can be selected.
	OperationKind_Select

	// # Execute functions.
	// 	- Operation applicable on commands, queries.
	// 	- Fields are not applicable.
	OperationKind_Execute

	// # Inherit ACL records from other roles.
	// 	- Operation applicable on roles only.
	// 	- Fields are not applicable.
	OperationKind_Inherits

	OperationKind_count
)

// Enumeration of ACL operation policy.
type PolicyKind uint8

//go:generate stringer -type=PolicyKind -output=stringer_policykind.go

const (
	PolicyKind_null PolicyKind = iota

	PolicyKind_Allow

	PolicyKind_Deny

	PolicyKind_count
)

type IResourcePattern interface {
	// Returns resource names that match the pattern.
	//
	// # insert, update and select:
	//	- records or view records names
	//
	// # execute:
	//	- commands & queries names
	//
	// # inherits:
	//	- roles names.
	//
	// `QNameANY` or `QNameAny×××` patterns are not allowed in resource names
	On() QNames

	// Returns fields (of records or views) then insert, update or select operation is described.
	Fields() []FieldName
}

// Represents a ACL rule record (specific rights or permissions) to be granted to role or revoked from.
type IACLRule interface {
	IWithComments

	// Returns operations that was granted or revoked.
	Ops() []OperationKind

	// Returns operations are granted or revoked.
	Policy() PolicyKind

	// Returns resource on which rule is applicable.
	Resources() IResourcePattern

	// Returns the role to which the operations was granted or revoked.
	Principal() IRole
}

// IWithACL is an interface for entities that have ACL.
type IWithACL interface {
	// Enumerates all ACL rules.
	//
	// Rules are enumerated in the order they are added.
	ACL(func(IACLRule) bool)
}

type IACLBuilder interface {
	// Grants specified operations on specified resources to specified role.
	//
	// # Panics:
	//   - if ops is empty,
	//	 - if resources are empty,
	//	 - if resources contains unknown names,
	//	 - if resources are mixed, e.g. records and commands,
	//	 - if ops are not compatible with resources,
	//	 - if fields are not applicable for ops,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Grant(ops []OperationKind, resources []QName, fields []FieldName, toRole QName, comment ...string) IACLBuilder

	// Grants all available operations on specified resources to specified role.
	//
	// If the resources are records or view records, then insert, update, and select are granted.
	//
	// If the resources are commands or queries, their execution is granted.
	//
	// If the resources are roles, then all operations from these roles are granted to specified role.
	//
	// No mixed resources are allowed.
	GrantAll(resources []QName, toRole QName, comment ...string) IACLBuilder

	// Revokes specified operations on specified resources from specified role.
	//
	// Revoke inherited roles is not supported
	Revoke(ops []OperationKind, resources []QName, fields []FieldName, fromRole QName, comment ...string) IACLBuilder

	// Remove all available operations on specified resources from specified role.
	//
	// Revoke inherited roles is not supported
	RevokeAll(resources []QName, fromRole QName, comment ...string) IACLBuilder
}
