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

// # Implements:
//   - iextengine.IExtensionIO
type extensionIO struct {
	istructs.IState
	istructs.IIntents
	appDef appdef.IAppDef
}

func newExtensionIO(appDef appdef.IAppDef, state istructs.IState, intents istructs.IIntents) IExtensionIO {
	return &extensionIO{
		IState:   state,
		IIntents: intents,
		appDef:   appDef,
	}
}

func (eio extensionIO) PackageFullPath(localName string) string {
	return eio.appDef.PackageFullPath(localName)
}

func (eio extensionIO) PackageLocalName(fullPath string) string {
	return eio.appDef.PackageLocalName(fullPath)
}
