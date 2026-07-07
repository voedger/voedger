/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registry

import (
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

// sys/registry, pseudoProfileWSID translated to appWSID
// creation of CDoc<Login> triggers opAsyncProjectorInvokeCreateWorkspaceID
func execCmdCreateLogin(args istructs.ExecCommandArgs) error {
	return createLogin(args, args.ArgumentObject.AsString(authnz.Field_Login))
}

// [~server.users/cmp.registry.CreateEmailLogin.go~impl]
func execCmdCreateEmailLogin(args istructs.ExecCommandArgs) error {
	return createLogin(args, args.ArgumentObject.AsString(authnz.Field_Email))
}

func createLogin(args istructs.ExecCommandArgs, login string) (err error) {
	appName := args.ArgumentObject.AsString(authnz.Field_AppName)

	subjectKind := args.ArgumentObject.AsInt32(authnz.Field_SubjectKind)
	if subjectKind >= int32(istructs.SubjectKind_FakeLast) || subjectKind <= int32(istructs.SubjectKind_null) {
		// TODO: cover it by tests
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "SubjectKind must be >0 and <", istructs.SubjectKind_FakeLast)
	}

	appQName, err := appdef.ParseAppQName(appName)
	if err != nil {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "failed to parse app qualified name", appQName.String(), ":", err)
	}

	appParts := args.Workpiece.(processors.IProcessorWorkpiece).AppPartitions()
	if _, err := appParts.AppDef(appQName); err != nil {
		if errors.Is(err, appparts.ErrNotFound) {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("target app %s is not found", appQName))
		}
		return err
	}

	// still need this check after https://github.com/voedger/voedger/issues/1311: the command is tkaen from AppWS, number of AppWS related to the login is checked here
	if err := CheckAppWSID(login, args.WSID, args.State.AppStructs().NumAppWorkspaces()); err != nil {
		return err
	}

	if err := validateSignInIdentifier(login); err != nil {
		return err
	}

	// deactivated cdoc.registry.Login is treated as missing by GetCDocLogin so the same login name can be registered again;
	// projectorLoginIdx then rewrites view.registry.LoginIdx by primary key (AppWSID, AppIDLoginHash)
	if err := assertIdentifierAvailable(args.State, appName, login, args.WSID, nil); err != nil {
		return err
	}

	wsKindInitializationData := args.ArgumentObject.AsString(authnz.Field_WSKindInitializationData)
	pwdSaltedHash, err := GetPasswordSaltedHash(args.ArgumentUnloggedObject.AsString(field_Passwrd))
	if err != nil {
		return err
	}
	profileCluster := args.ArgumentObject.AsInt32(authnz.Field_ProfileCluster)
	if profileCluster <= 0 || profileCluster > math.MaxUint16 {
		// TODO: cover it by tests
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, "ProfileCluster must be >0 and <", math.MaxUint16)
	}

	kb, err := args.State.KeyBuilder(sys.Storage_Record, QNameCDocLogin)
	if err != nil {
		return err
	}
	cdocLogin, err := args.Intents.NewValue(kb)
	if err != nil {
		return err
	}
	cdocLogin.PutInt32(authnz.Field_ProfileCluster, profileCluster)
	cdocLogin.PutBytes(field_PwdHash, pwdSaltedHash)
	cdocLogin.PutString(authnz.Field_AppName, appName)
	cdocLogin.PutInt32(authnz.Field_SubjectKind, subjectKind)
	cdocLogin.PutString(authnz.Field_LoginHash, GetLoginHash(login))
	cdocLogin.PutRecordID(appdef.SystemField_ID, 1)
	cdocLogin.PutString(authnz.Field_WSKindInitializationData, wsKindInitializationData)

	return nil
}

// sys/registry, appWorkspace, triggered by CDoc<Login>
var projectorLoginIdx = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	for rec := range event.CUDs {
		if rec.QName() != QNameCDocLogin {
			continue
		}
		kb, err := s.KeyBuilder(sys.Storage_View, QNameViewLoginIdx)
		if err != nil {
			return err
		}
		kb.PutInt64(field_AppWSID, int64(event.Workspace()))
		kb.PutString(field_AppIDLoginHash, rec.AsString(authnz.Field_AppName)+"/"+rec.AsString(authnz.Field_LoginHash))

		vb, err := intents.NewValue(kb)
		if err != nil {
			return err
		}
		vb.PutInt64(field_CDocLoginID, int64(rec.AsRecordID(appdef.SystemField_ID)))
	}
	return nil
}
