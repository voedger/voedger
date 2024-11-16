/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import "github.com/voedger/voedger/pkg/appdef"

// NullResults is a result then filter matches nothing.
var NullResults appdef.IWithTypes = makeResults()
