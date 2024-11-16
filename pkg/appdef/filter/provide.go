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
