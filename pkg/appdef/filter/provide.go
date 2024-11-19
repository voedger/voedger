/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

// QNames is a filter that matches types by their qualified names.
func QNames(name appdef.QName, names ...appdef.QName) appdef.IFilter {
	return makeQNamesFilter(name, names...)
}

// Tags is a filter that matches types by their tags.
//
// Deprecated: not ready IType.Tags().
func Tags(tag string, tags ...string) appdef.IFilter {
	return makeTagsFilter(tag, tags...)
}

// Types is a filter that matches types by their kind.
func Types(t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	return makeTypesFilter(t, tt...)
}

// And returns a filter that matches types that match all filters.
func And(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	return makeAndFilter(f1, f2, ff...)
}

// Or returns a filter that matches types that match any filter.
func Or(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	return makeOrFilter(f1, f2, ff...)
}

// Not returns a filter that invert matches for specified filter.
func Not(f appdef.IFilter) appdef.IFilter {
	return makeNotFilter(f)
}

// AllTables returns a filter that matches all structured types, see appdef.TypeKind_Structures
func AllTables() appdef.IFilter {
	s := appdef.TypeKind_Structures.AsArray()
	return makeTypesFilter(s[0], s[1:]...)
}

// AllFunctions returns a filter that matches all functions types, see appdef.TypeKind_Functions
func AllFunctions() appdef.IFilter {
	f := appdef.TypeKind_Functions.AsArray()
	return makeTypesFilter(f[0], f[1:]...)
}

// Matches returns all types that match the filter.
func Matches(f appdef.IFilter, types appdef.SeqType) appdef.SeqType {
	return allMatches(f, types)
}
