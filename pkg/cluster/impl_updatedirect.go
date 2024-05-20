/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

func updateDirect(asp istructs.IAppStructsProvider, appQName istructs.AppQName, wsid istructs.WSID, qNameToUpdate appdef.QName, appDef appdef.IAppDef) error {
	targetAppStructs, err := asp.AppStructs(appQName)
	if err != nil {
		// test here
		return err
	}
	var as istorage.IAppStorage
	as.Read()
}
