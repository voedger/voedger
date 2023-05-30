/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package signupin

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func provideChangePassword(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	pars := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "ChangePasswordParams"))
	pars.AddField(field_Login, appdef.DataKind_string, true).
		AddField(Field_AppName, appdef.DataKind_string, true)
	unl := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "ChangePasswordUnloggedParams"))
	unl.AddField(field_OldPassword, appdef.DataKind_string, true).
		AddField(field_NewPassword, appdef.DataKind_string, true)
	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdChangePassword, pars.QName(), unl.QName(), appdef.NullQName,
		cmdChangePaswordExec,
	))

	cfgRegistry.FunctionRateLimits.AddAppLimit(qNameCmdChangePassword, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 1,
	})
}

// sys/registry/pseudoWSID
// null auth
func cmdChangePaswordExec(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
	appName := args.ArgumentObject.AsString(Field_AppName)
	login := args.ArgumentObject.AsString(field_Login)
	oldPwd := args.ArgumentUnloggedObject.AsString(field_OldPassword)
	newPwd := args.ArgumentUnloggedObject.AsString(field_NewPassword)

	cdocLogin, err := GetCDocLogin(login, args.State, args.Workspace, appName)
	if err != nil {
		return err
	}

	if err := CheckPassword(cdocLogin, oldPwd); err != nil {
		return err
	}

	return ChangePasswordCDocLogin(cdocLogin, newPwd, args.Intents, args.State)
}
