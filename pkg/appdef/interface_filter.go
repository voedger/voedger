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

	// Return sorted slice of QNames.
	// If kind is not FilterKind_QNames, returns nil
	QNames() QNames

	// Return set of type kinds
	// If kind is not FilterKind_Types, returns nil
	Types() TypeKindSet

	// Return sorted slice of tags
	// If kind is not FilterKind_Tags, returns nil
	Tags() []string

	// Return slice of conjunct sub-filters
	// If kind is not FilterKind_And, returns nil
	And() []IFilter

	// Return slice of disjunct sub-filters
	// If kind is not FilterKind_Or, returns nil
	Or() []IFilter

	// Return negative sub-filter
	// If kind is not FilterKind_Not, returns nil
	Not() IFilter

	// Returns is type matched by filter
	Match(IType) bool

	// Returns names of types, what matches by filter
	Matches(IWithTypes) QNames
}
