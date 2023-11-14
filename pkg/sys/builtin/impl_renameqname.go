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

func proivideRenameQName(cfg *istructsmem.AppConfigType, adb appdef.IAppDefBuilder, asp istorage.IAppStorageProvider) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "RenameQName"),
		appdef.NullQName,
		appdef.NullQName,
		appdef.NullQName,
		provideExecCmdRenameQName(asp, cfg)))
}

func provideExecCmdRenameQName(asp istorage.IAppStorageProvider, cfg *istructsmem.AppConfigType) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		storage, err := asp.AppStorage(cfg.Name)
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
