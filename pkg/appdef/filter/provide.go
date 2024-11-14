/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

func QNames(name appdef.QName, names ...appdef.QName) appdef.IFilter {
	return qNames(name, names...)
}

func Types(t appdef.TypeKind, tt ...appdef.TypeKind) appdef.IFilter {
	return types(t, tt...)
}
