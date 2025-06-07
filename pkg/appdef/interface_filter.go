/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Filter kind enumeration
type FilterKind uint8

//go:generate stringer -type=FilterKind -output=stringer_filterkind.go
const (
	FilterKind_null FilterKind = iota

	FilterKind_True

	FilterKind_QNames
	FilterKind_Types
	FilterKind_Tags

	FilterKind_And
	FilterKind_Or
	FilterKind_Not

	FilterKind_count
)

type IFilter interface {
	// Filter kind
	Kind() FilterKind

	// switch members by kind

	// Return filtered QNames.
	// If kind is not FilterKind_QNames, returns empty iterator
	QNames() []QName

	// Return filtered type kinds.
	// If kind is not FilterKind_Types, returns empty iterator
	Types() []TypeKind

	// Return filtered types workspace.
	// If kind is not FilterKind_Types or not workspace assigned, returns NullQName.
	WS() QName

	// Return filtered tags.
	// If kind is not FilterKind_Tags, returns empty iterator
	Tags() []QName

	// Returns sub-filters to conjunct
	// If kind is not FilterKind_And, returns empty iterator
	And() []IFilter

	// Returns sub-filters to disjunct
	// If kind is not FilterKind_Or, returns empty iterator
	Or() []IFilter

	// Return negative sub-filter
	// If kind is not FilterKind_Not, returns nil
	Not() IFilter

	// Returns is type matched by filter
	Match(IType) bool
}
