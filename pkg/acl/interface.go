/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package acl

type IACL[Role, Operation, Resource comparable] interface {

	// HasPermission checks if the specified combination was granted via IACLBuilder.Grant() call.
	HasPermission(role Role, o Operation, r Resource) bool
}

type IACLBuilder[Role, Operation, Resource comparable] interface {
	Grant(role Role, op Operation, rp Resource)
	Build() IACL[Role, Operation, Resource]
}
