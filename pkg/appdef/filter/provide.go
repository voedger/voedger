/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

func QNames(name appdef.QName, names ...appdef.QName) IFilter {
	return qNames(name, names...)
}
