/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package registry

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideCanonicalLoginEnablement(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandSetCanonicalLoginEnablement,
		execCmdSetCanonicalLoginEnablement,
	))
}

func execCmdSetCanonicalLoginEnablement(args istructs.ExecCommandArgs) error {
	login := args.ArgumentObject.AsString(field_Login)
	appName := args.ArgumentObject.AsString(field_AppName)
	enabled := args.ArgumentObject.AsBool(field_Enabled)
	disabled := !enabled

	if err := CheckAppWSID(login, args.WSID, args.State.AppStructs().NumAppWorkspaces()); err != nil {
		return err
	}

	cdocLogin, loginExists, err := GetCDocLogin(login, args.State, args.WSID, appName)
	if err != nil {
		return err
	}
	if !loginExists {
		return errLoginDoesNotExist(login)
	}

	if canonicalLoginDisabledIsSpecified(cdocLogin) && isCanonicalLoginEnabled(cdocLogin) == enabled {
		return nil
	}

	kb, err := args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return err
	}
	loginUpdater, err := args.Intents.UpdateValue(kb, cdocLogin)
	if err != nil {
		return err
	}
	loginUpdater.PutBool(field_CanonicalLoginDisabled, disabled)
	return nil
}

func isCanonicalLoginEnabled(cdocLogin istructs.IRowReader) bool {
	return !cdocLogin.AsBool(field_CanonicalLoginDisabled)
}

func canonicalLoginDisabledIsSpecified(cdocLogin istructs.IStateValue) bool {
	recordValue, ok := cdocLogin.(istructs.IStateRecordValue)
	if !ok {
		return false
	}

	specified := false
	recordValue.AsRecord().SpecifiedValues(func(field appdef.IField, _ any) bool {
		if field.Name() == field_CanonicalLoginDisabled {
			specified = true
			return false
		}
		return true
	})
	return specified
}
