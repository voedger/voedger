/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Tag is a type that groups other types.
type ITag interface {
	IType
}

// IWithTags is an interface for types that have tags.
type IWithTags interface {
	// Returns tags.
	Tags(func(ITag) bool)
}

type ITagBuilder interface {
	// Adds new tags with specified name.
	//
	// # Panics:
	//   - if tag with specified name is not found.
	SetTag(QName, ...QName)
}
