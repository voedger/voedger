/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Enumeration of privileges.
type PrivilegeKind int8

//go:generate stringer -type=PrivilegeKind -output=stringer_privilegekind.go

const (
	PrivilegeKind_null PrivilegeKind = iota

	// GRANT INSERT,UPDATE ON ALL TABLES WITH TAG BackofficeTag TO LocationUser;
	// GRANT UPDATE (CloseDatetime, Client) ON TABLE Bill TO LocationUser;
	// GRANT SELECT ON TABLE Orders TO LocationUser;
	// GRANT INSERT ON WORKSPACE Workspace1 TO Role1;
	// GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
	PrivilegeKind_Insert
	PrivilegeKind_Update
	PrivilegeKind_Select

	// GRANT EXECUTE ON COMMAND Orders TO LocationUser;
	// GRANT EXECUTE ON QUERY Query1 TO LocationUser;
	// GRANT EXECUTE ON WORKSPACE TO Role2;
	PrivilegeKind_Execute

	// GRANT LocationUser TO LocationManager;
	PrivilegeKind_Role
)

// Represents a privilege (specific rights or permissions) granted to a role.
type IPrivilege interface {
	IWithComments

	// Returns privilege kind
	Kind() PrivilegeKind

	// Returns objects for which privilege was granted or revoked.
	//
	// For PrivilegeKind_Role returns Role names.
	//
	// For PrivilegeKind_Insert, GrantKind_Update and GrantKind_Select returns:
	//	- records or view records names or
	//	- workspaces names.
	//
	// For PrivilegeKind_Execute returns:
	//	- commands & queries names or
	//	- workspaces names.
	Objects() QNames

	// Returns fields (of objects) which was granted or revoked.
	//
	// For PrivilegeKind_Update and PrivilegeKind_Select returns field names of records or view records.
	Fields() []FieldName

	// Returns the role to which the privilege was granted or revoked.
	Role() IRole
}

// IWithPrivileges is an interface for entities that have grants.
type IWithPrivileges interface {
	// Enumerates all privileges.
	//
	// Privileges are enumerated in alphabetical order of roles, and within each role in the order they are added.
	Privileges(func(IPrivilege))

	// Enumerates all privileges with specified kind.
	PrivilegesByKind(PrivilegeKind, func(IPrivilege))

	// Returns all privileges for entity with specified QName.
	PrivilegesFor(QName) []IPrivilege
}

type IPrivilegesBuilder interface {
	// Grants new privilege with specified kind to specified objects for specified role.
	//
	// # Panics:
	//   - if kind is PrivilegeKind_null,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Grant(kind PrivilegeKind, objects []QName, fields []FieldName, toRole QName, comment ...string) IPrivilegesBuilder

	// Grants all available privileges to specified objects for specified role.
	//
	// If the objects are tables, then insert, update, and select privileges are granted.
	//
	// If the objects are commands or queries, their execution is granted.
	//
	// If the objects are workspaces, then:
	//	- insert, update and select from the tables and views of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	GrantAll(objects []QName, toRole QName, comment ...string) IPrivilegesBuilder

	// Grant new privilege to specified roles for specified role.
	// The result is that the specified role will inherits all privileges from specified roles.
	GrantRoles(roles []QName, toRole QName, comment ...string) IPrivilegesBuilder
}
