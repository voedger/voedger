/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

func NewACLBuilder[QName comparable]() IACLBuilder[QName] {
	return &aclBuilder[QName]{}
}
