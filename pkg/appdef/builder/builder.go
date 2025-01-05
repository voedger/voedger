/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package builder

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/apps"
)

func New() appdef.IAppDefBuilder {
	a := apps.NewAppDef()
	return apps.NewAppDefBuilder(a)
}
