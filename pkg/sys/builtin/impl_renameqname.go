/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package builtin

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/istructsmem/qrename"
)

func proivideRenameQName(sr istructsmem.IStatelessResources, asp istorage.IAppStorageProvider) {
	sr.AddCommands(appdef.SysPackagePath, istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "RenameQName"),
		provideExecCmdRenameQName(asp)))
}

func provideExecCmdRenameQName(asp istorage.IAppStorageProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		args.State.AppStructs()
		appQName := args.State.AppStructs().AppQName()
		storage, err := asp.AppStorage(appQName)
		if err != nil {
			// notest
			return err
		}
		existingQName := args.ArgumentObject.AsQName(field_ExistingQName)
		newQNameStr := args.ArgumentObject.AsString(field_NewQName)
		newQName, err := appdef.ParseQName(newQNameStr)
		if err != nil {
			return err
		}
		return qrename.Rename(storage, existingQName, newQName)
	}
}
