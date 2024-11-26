/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// See [Issue #488](https://github.com/voedger/voedger/issues/488)
//
// Any type may have comment
type IWithComments interface {
	// Returns comment
	Comment() string

	// Returns comment as string array
	CommentLines() []string
}

// ICommenter is interface to set comment for an entity (type, field, etc.).
type ICommenter interface {
	// Sets comment as string with lines, concatenated with LF
	SetComment(...string)
}

// ITypeCommenter is interface to set comment for a type.
type ITypeCommenter interface {
	// Sets comments for type, specified by name.
	// For workspace type should be local.
	//
	// # Panics:
	//   - if type with specified name is not found.
	SetTypeComment(QName, ...string)
}
