/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "iter"

// Tag is a type that groups other types.
type ITag interface {
	IType

	// Unwanted type assertion stub
	isTag()
}

// IWithTags is an interface for types that have tags.
type IWithTags interface {
	// HasTag returns has type specified tag.
	HasTag(QName) bool

	// Returns tags.
	//
	// Tags are returned in alphabetical order.
	Tags() iter.Seq[ITag]
}

// ITagger is an interface to set tags for type.
type ITagger interface {
	// Sets specified tags.
	//
	// # Panics:
	//   - if tag with specified name is not found.
	SetTag(tag ...QName)
}

// ITagsBuilder is an interface for building tags.
type ITagsBuilder interface {
	// Adds new tags with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddTag(name QName, comments ...string)
}
