/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideChangePassword(cfgRegistry *istructsmem.AppConfigType) {
	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		qNameCmdChangePassword,
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
	appName := args.ArgumentObject.AsString(field_AppName)
	login := args.ArgumentObject.AsString(field_Login)
	oldPwd := args.ArgumentUnloggedObject.AsString(field_OldPassword)
	newPwd := args.ArgumentUnloggedObject.AsString(field_NewPassword)

	cdocLogin, loginExists, err := GetCDocLogin(login, args.State, args.WSID, appName)
	if err != nil {
		return err
	}

	if !loginExists {
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
