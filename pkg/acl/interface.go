/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

type IACL[QName comparable] interface {

	// Returs true if access is granted, false otherwise
	CheckTableAccess(op string, fields []string, table QName, roles []QName) bool

	// Returs true if access is granted, false otherwise
	CheckExecAccess(resource QName, roles []QName) bool
}

type IACLBuilder[QName comparable] interface {
	TagTable(tag QName, table QName) IACLBuilder[QName]

	// If empty op is used for a given table, all ops will be granted for this table and subsequent grants will be ignored
	// If empty fields is used for a given table, op will be granted for all fields subsequent grants will be ignored
	GrantOpOnTable(op string, fields []string, table QName, role QName) IACLBuilder[QName]

	// Same rules as for GrantOpOnTable, but grants are applied to all tables with given tag, see TagTable method
	GrantOpOnTableByTag(op string, fields []string, tag QName, role QName) IACLBuilder[QName]

	GrantExec(resource QName, role QName) IACLBuilder[QName]

	// Must be the last method to call
	Build() IACL[QName]
}
