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
	// 	- Operation applicable on records, view records or workspaces.
	// 	- Then applied to workspaces, it means insert on all tables and views of the workspace.
	// 	- Fields are not applicable.
	OperationKind_Insert

	// # Update records or view records.
	// 	- Operation applicable on records, view records or workspaces.
	// 	- Then applied to workspaces, it means update on all tables and views of the workspace.
	// 	- Fields are applicable and specify fields of records or view records that can be updated.
	OperationKind_Update

	// # Select records or view records.
	// 	- Operation applicable on records, view records or workspaces.
	// 	- Then applied to workspaces, it means select on all tables and views of the workspace.
	// 	- Fields are applicable and specify fields of records or view records that can be selected.
	OperationKind_Select

	// # Execute functions.
	// 	- Operation applicable on commands, queries or workspaces.
	// 	- Then applied to workspaces, it means execute on all queries and commands of the workspace.
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

	PolicyKind_Count
)

type IResourcePattern interface {
	// Returns resource names that match the pattern.
	//
	// # insert, update and select:
	//	- records or view records names or
	//	- workspaces names.
	//
	// # execute:
	//	- commands & queries names or
	//	- workspaces names.
	//
	// # inherits:
	//	- roles names.
	//
	// Resource names can include `QNameANY` or `QNameAny×××` patterns.
	On() QNames

	// Returns fields (of records or views) then update or select operation is described.
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
	ACL(func(IACLRule))

	// Returns all ACL rules on specified resources, which contains at least one from specified kinds.
	//
	// If no kinds specified then all rules are returned.
	//
	// Rules are returned in the order they are added.
	ACLForResources([]QName, ...OperationKind) []IACLRule
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
	// Resource names can include `QNameANY` or `QNameAny×××` patterns.
	//
	// If the resources are records or view records, then insert, update, and select are granted.
	//
	// If the resources are commands or queries, their execution is granted.
	//
	// If the resources are workspaces, then:
	//	- insert, update and select records and view records of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	//
	// If the resources are roles, then all operations from these roles are granted to specified role.
	//
	// No mixed resources are allowed.
	GrantAll(resources []QName, toRole QName, comment ...string) IACLBuilder

	// Revokes specified operations on specified resources from specified role.
	Revoke(ops []OperationKind, resources []QName, fromRole QName, comment ...string) IACLBuilder

	// Remove all available operations on specified resources from specified role.
	RevokeAll(resources []QName, fromRole QName, comment ...string) IACLBuilder
}
