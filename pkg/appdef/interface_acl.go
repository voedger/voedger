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

// Represents a ACL rule record (specific rights or permissions) to be granted to role or revoked from.
type IACLRule interface {
	IWithComments

	// Returns operation kinds
	Ops() []OperationKind

	// Returns is privilege has been granted. The opposite of `IsRevoked()`
	Policy() PolicyKind

	// Returns objects names on which privilege was applied.
	//
	// # For PrivilegeKind_Insert, GrantKind_Update and GrantKind_Select:
	//	- records or view records names or
	//	- workspaces names.
	//
	// # For PrivilegeKind_Execute:
	//	- commands & queries names or
	//	- workspaces names.
	//
	// # For PrivilegeKind_Inherits:
	//	- roles names.
	On() QNames

	// Returns fields (of objects) which was granted or revoked.
	//
	// For PrivilegeKind_Update and PrivilegeKind_Select returns field names of records or view records.
	Fields() []FieldName

	// Returns the role to which the privilege was granted or revoked.
	To() IRole
}

// IWithACL is an interface for entities that have ACL.
type IWithACL interface {
	// Enumerates all ACL rules.
	//
	// Rules are enumerated in the order they are added.
	Privileges(func(IACLRule))

	// Returns all privileges on specified entities, which contains at least one from specified kinds.
	//
	// If no kinds specified then all privileges on entities are returned.
	//
	// Privileges are returned in the order they are added.
	PrivilegesOn(on []QName, kind ...OperationKind) []IACLRule
}

type IPrivilegesBuilder interface {
	// Grants new privilege with specified kinds on specified objects to specified role.
	//
	// # Panics:
	//   - if kinds is empty,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if objects are mixed, e.g. records and commands,
	//	 - if kinds are not compatible with objects,
	//	 - if fields are not applicable for privilege,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Grant(kinds []OperationKind, on []QName, fields []FieldName, toRole QName, comment ...string) IPrivilegesBuilder

	// Grants all available privileges on specified objects to specified role.
	// Object names can include `QNameANY` or `QNameAny×××` names.
	//
	// If the objects are records or view records, then insert, update, and select are granted.
	//
	// If the objects are commands or queries, their execution is granted.
	//
	// If the objects are workspaces, then:
	//	- insert, update and select records and view records of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	//
	// If the objects are roles, then all privileges from these roles are granted to specified role.
	//
	// No mixed objects are allowed.
	GrantAll(on []QName, toRole QName, comment ...string) IPrivilegesBuilder

	// Revokes privilege with specified kind on specified objects from specified role.
	Revoke(kinds []OperationKind, on []QName, fromRole QName, comment ...string) IPrivilegesBuilder

	// Remove all available privileges on specified objects from specified role.
	RevokeAll(on []QName, fromRole QName, comment ...string) IPrivilegesBuilder
}
