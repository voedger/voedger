/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Tag is a type that groups other types.
type ITag interface {
	IType
}

// IWithTags is an interface for types that have tags.
type IWithTags interface {
	// Tag returns tag by name.
	//
	// Returns nil if tag with specified name is not found.
	Tag(QName) ITag

	// Returns tags.
	//
	// Tags are returned in alphabetical order.
	Tags(func(ITag) bool)
}

// ITagBuilder is an interface for building a type with tags.
type ITagBuilder interface {
	// Adds new tags with specified name.
	//
	// # Panics:
	//   - if tag with specified name is not found.
	SetTag(QName, ...QName)
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
