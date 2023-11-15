/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

type IACL[QName comparable] interface {
	CheckTableAccess(op string, fields []string, table QName, roles []QName) bool
	CheckExecAccess(resource QName, roles []QName) bool
}

type IACLBuilder[QName comparable] interface {
	Role(name QName) IACLBuilder[QName]

	// Empty op means "all ops"
	// Empty fields means "all fields"
	GrantOpOnTable(op string, fields []string, table QName, role QName) IACLBuilder[QName]
	GrantOpOnTableByTag(op string, fields []string, taq QName, role QName) IACLBuilder[QName]

	GrantExec(resource QName, role QName) IACLBuilder[QName]

	// Must be the last method to call
	Build() IACL[QName]
}
