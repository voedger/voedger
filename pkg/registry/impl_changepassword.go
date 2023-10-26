/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func provideChangePassword(cfgRegistry *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdChangePassword,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "ChangePasswordParams")).
			AddField(authnz.Field_Login, appdef.DataKind_string, true).
			AddField(authnz.Field_AppName, appdef.DataKind_string, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "ChangePasswordUnloggedParams")).
			AddField(field_OldPassword, appdef.DataKind_string, true).
			AddField(field_NewPassword, appdef.DataKind_string, true).(appdef.IType).QName(),
		appdef.NullQName,
		cmdChangePasswordExec,
	))

	cfgRegistry.FunctionRateLimits.AddAppLimit(qNameCmdChangePassword, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 1,
	})
}

// sys/registry/pseudoWSID
// null auth
func cmdChangePasswordExec(args istructs.ExecCommandArgs) (err error) {
	appName := args.ArgumentObject.AsString(authnz.Field_AppName)
	login := args.ArgumentObject.AsString(authnz.Field_Login)
	oldPwd := args.ArgumentUnloggedObject.AsString(field_OldPassword)
	newPwd := args.ArgumentUnloggedObject.AsString(field_NewPassword)

	cdocLogin, doesLoginExist, err := GetCDocLogin(login, args.State, args.Workspace, appName)
	if err != nil {
		return err
	}

	if !doesLoginExist {
		return errLoginDoesNotExist(login)
	}

	isPasswordOK, err := CheckPassword(cdocLogin, oldPwd)
	if err != nil {
		return err
	}

	if !isPasswordOK {
		return errPasswordIsIncorrect
	}

	return ChangePasswordCDocLogin(cdocLogin, newPwd, args.Intents, args.State)
}
