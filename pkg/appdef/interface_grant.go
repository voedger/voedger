/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Enumeration of grants.
type GrantKind int8

//go:generate stringer -type=GrantKind -output=stringer_grantkind.go

const (
	GrantKind_null GrantKind = iota

	// GRANT INSERT,UPDATE ON ALL TABLES WITH TAG BackofficeTag TO LocationUser;
	// GRANT UPDATE (CloseDatetime, Client) ON TABLE Bill TO LocationUser;
	// GRANT SELECT ON TABLE Orders TO LocationUser;
	// GRANT INSERT ON WORKSPACE Workspace1 TO Role1;
	// GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
	GrantKind_Insert
	GrantKind_Update
	GrantKind_Select

	// GRANT EXECUTE ON COMMAND Orders TO LocationUser;
	// GRANT EXECUTE ON QUERY Query1 TO LocationUser;
	// GRANT EXECUTE ON WORKSPACE TO Role2;
	GrantKind_Execute

	// GRANT LocationUser TO LocationManager;
	GrantKind_Role
)

// Grant represents a grant of rights to a role.
type IGrant interface {
	IWithComments

	// Returns Grant kind
	Kind() GrantKind

	// Returns objects which was granted.
	//
	// For GrantKind_Role returns Role names.
	//
	// For GrantKind_Insert, GrantKind_Update and GrantKind_Select returns:
	//	- table names or
	//	- workspace names.
	//
	// For GrantKind_Execute returns:
	//	- commands & queries names or
	//	- workspaces names.
	Objects() QNames

	// Returns fields (of objects) which was granted.
	//
	// For GrantKind_Update and GrandKind_Select returns field names of table.
	Fields() []FieldName

	// Returns the role to which the grant was granted.
	Role() IRole
}

// IWithGrants is an interface for entities that have grants.
type IWithGrants interface {
	// Enumerates all grants.
	//
	// Grants are enumerated in the order they were added.
	Grants(func(IGrant))

	// Enumerates all grants with specified kind.
	GrantsByKind(GrantKind, func(IGrant))

	// Returns all grants for specified object.
	GrantsForObject(QName) []IGrant
}

type IGrantsBuilder interface {
	// Adds new Grant with specified kind to specified objects for specified role.
	//
	// # Panics:
	//   - if kind is GrantKind_null,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Grant(kind GrantKind, objects []QName, fields []FieldName, toRole QName, comment ...string) IGrantsBuilder

	// Adds all available grants to specified objects for specified role.
	//
	// If the objects are tables, then insert, update, and select operations are granted.
	//
	// If the objects are commands or queries, their execution is allowed.
	//
	// If the objects are workspaces, then:
	//	- insert, update and select from the tables of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	GrantAll(objects []QName, toRole QName, comment ...string) IGrantsBuilder

	// Adds new Grant with GrantKind_Role to specified roles for specified role.
	GrantRoles(roles []QName, toRole QName, comment ...string) IGrantsBuilder
}
