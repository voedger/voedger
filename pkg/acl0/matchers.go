/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl0

type ResourceWithFields[QName comparable] struct {
	Resource QName
	Fields   []string
}

// Pattern matches ResourceWithFields if all of the	following is true:
// 1. ResourceWithFields.Resource is in Resources or Resources is empty
// 2. ResourceWithFields.Fields is in Resources
type ResourceWithFieldsPattern[QName comparable] struct {
	Resources []QName
	Fields    []string
}

type IResourceWithFieldsMatcher[QName comparable] interface {
	IResourceMatcher[ResourceWithFields[QName], ResourceWithFieldsPattern[QName]]
}
