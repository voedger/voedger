/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Enumeration of ACL operation policy.
type PolicyKind uint8

//go:generate stringer -type=PolicyKind -output=stringer_policykind.go

const (
	PolicyKind_null PolicyKind = iota

	PolicyKind_Allow

	PolicyKind_Deny

	PolicyKind_count
)

type IACLFilter interface {
	IFilter

	// Returns fields iterator then insert, update or select operation is described.
	Fields() []FieldName

	// Resturns true if fields are specified.
	HasFields() bool
}

// Represents a ACL rule record (specific rights or permissions) to be granted to role or revoked from.
type IACLRule interface {
	IWithComments

	// Returns is this rule describes specified operation.
	Op(OperationKind) bool

	// Returns operations that was described (granted or revoked).
	Ops() []OperationKind

	// Returns the policy: operations are granted (policy is allow) or revoked (policy is deny).
	Policy() PolicyKind

	// Returns filter of types on which rule is applicable.
	//
	// If filter contains FilterKind_Types, then local types from workspace where rule is defined are used.
	Filter() IACLFilter

	// Returns the role to which the operations was granted or revoked.
	Principal() IRole

	// Returns workspace where the rule is defined.
	Workspace() IWorkspace
}

// IWithACL is an interface for entities that have ACL.
type IWithACL interface {
	// Enumerates all ACL rules.
	//
	// Rules are enumerated in the order they are added.
	ACL() []IACLRule
}

type IACLBuilder interface {
	// Grants operations on filtered types to role.
	//
	// # Panics:
	//   - if ops is empty,
	//	 - if ops contains incompatible operations (e.g. INSERT with EXECUTE),
	//	 - if filtered type is not compatible with operations,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Grant(ops []OperationKind, flt IFilter, fields []FieldName, toRole QName, comment ...string) IACLBuilder

	// Grants all available operations on filtered types to role.
	//
	// If the types are records or view records, then insert, update, and select are granted.
	//
	// If the types are commands or queries, their execution is granted.
	//
	// If the types are roles, then all operations from these roles are granted to specified role.
	//
	// No mixed types are allowed.
	GrantAll(flt IFilter, toRole QName, comment ...string) IACLBuilder

	// Revokes operations on filtered types from role.
	//
	// Revoke inherited roles is not supported
	Revoke(ops []OperationKind, resources IFilter, fields []FieldName, fromRole QName, comment ...string) IACLBuilder

	// Remove all available operations on filtered types from role.
	//
	// Revoke inherited roles is not supported
	RevokeAll(flt IFilter, fromRole QName, comment ...string) IACLBuilder
}
