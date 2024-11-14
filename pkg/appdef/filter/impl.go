/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

func qNames(name appdef.QName, names ...appdef.QName) IFilter {
	f := &qNamesFilter{names: appdef.QNamesFrom(name)}
	f.names.Add(names...)
	return f
}
