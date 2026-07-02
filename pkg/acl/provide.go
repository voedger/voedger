/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package acl

func NewACLBuilder[Role, Operation, Resource comparable]() IACLBuilder[Role, Operation, Resource] {
	return &aclBuilder[Role, Operation, Resource]{
		permissions: make(map[Role]map[Operation]map[Resource]bool),
	}
}
