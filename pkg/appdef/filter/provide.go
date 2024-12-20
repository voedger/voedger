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
func Tags(tag appdef.QName, tags ...appdef.QName) appdef.IFilter {
	return makeTagsFilter(tag, tags...)
}

// Types is a filter that matches types by their kind.
//
// If workspace is not empty, then matched types should be located in the specified workspace.
func Types(ws appdef.QName, t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	return makeTypesFilter(ws, t, tt...)
}

// And returns a filter that matches types that match all children filters.
func And(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	return makeAndFilter(f1, f2, ff...)
}

// Or returns a filter that matches types that match any children filter.
func Or(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	return makeOrFilter(f1, f2, ff...)
}

// Not returns a filter that invert matches for specified filter.
func Not(f appdef.IFilter) appdef.IFilter {
	return makeNotFilter(f)
}

// True returns filter that always matches any type
func True() appdef.IFilter { return trueFlt }

// AllTables returns a filter that matches all structured types from workspace, see appdef.TypeKind_Structures
func AllTables(ws appdef.QName) appdef.IFilter {
	s := appdef.TypeKind_Structures.AsArray()
	return makeTypesFilter(ws, s[0], s[1:]...)
}

// AllFunctions returns a filter that matches all functions types from workspace, see appdef.TypeKind_Functions
func AllFunctions(ws appdef.QName) appdef.IFilter {
	f := appdef.TypeKind_Functions.AsArray()
	return makeTypesFilter(ws, f[0], f[1:]...)
}
