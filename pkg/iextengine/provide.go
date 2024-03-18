/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iextengine

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// creates and returns new instance of IExtensionIO
func NewExtensionIO(appdef appdef.IAppDef, state istructs.IState, intents istructs.IIntents) IExtensionIO {
	return newExtensionIO(appdef, state, intents)
}
