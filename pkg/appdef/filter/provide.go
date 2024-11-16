/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

func QNames(name appdef.QName, names ...appdef.QName) appdef.IFilter {
	return makeQNamesFilter(name, names...)
}

func Types(t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	return makeTypesFilter(t, tt...)
}

func And(f1, f2 appdef.IFilter, ff ...appdef.IFilter) appdef.IFilter {
	return makeAndFilter(f1, f2, ff...)
}
