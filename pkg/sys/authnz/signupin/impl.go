/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package signupin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// sys/registry, pseudoProfileWSID translated to appWSID
// creation of CDoc<Login> triggers opAsyncProjectorInvokeCreateWorkspaceID
func execCmdCreateLogin(asp istructs.IAppStructsProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		loginStr := args.ArgumentObject.AsString(authnz.Field_Login)
		appName := args.ArgumentObject.AsString(Field_AppName)

		subjectKind := istructs.SubjectKindType(args.ArgumentObject.AsInt32(authnz.Field_SubjectKind))
		if subjectKind >= istructs.SubjectKind_FakeLast || subjectKind <= istructs.SubjectKind_null {
			return errors.New("wrong subject kind")
		}

		appQName, err := istructs.ParseAppQName(appName)
		if err != nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "failed to parse app qualified name", appQName.String(), ":", err)
		}

		as, err := asp.AppStructs(appQName)
		if err != nil {
			if err == istructs.ErrAppNotFound {
				return coreutils.NewHTTPErrorf(http.StatusBadRequest, "unknown application ", appName)
			}
			return err
		}

		if err = CheckAppWSID(loginStr, args.Workspace, as.WSAmount()); err != nil {
			return
		}

		// see https://dev.untill.com/projects/#!537026
		if strings.HasPrefix(loginStr, "-") || strings.HasPrefix(loginStr, ".") || strings.HasPrefix(loginStr, " ") ||
			strings.HasSuffix(loginStr, "-") || strings.HasSuffix(loginStr, ".") || strings.HasSuffix(loginStr, " ") ||
			strings.Contains(loginStr, "..") || !validLoginRegexp.MatchString(loginStr) {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "incorrect login format: ", loginStr)
		}

		cdocLoginID, err := GetCDocLoginID(args.State, args.Workspace, appName, loginStr)
		if err != nil {
			return err
		}
		if cdocLoginID > 0 {
			return coreutils.NewHTTPErrorf(http.StatusConflict, "login already exists")
		}

		wsKindInitializationData := args.ArgumentObject.AsString(authnz.Field_WSKindInitializationData)
		pwdSaltedHash, err := GetPasswordSaltedHash(args.ArgumentUnloggedObject.AsString(field_Passwrd))
		if err != nil {
			return err
		}
		profileCluster := args.ArgumentObject.AsInt32(authnz.Field_ProfileClusterID)

		kb, err := args.State.KeyBuilder(state.RecordsStorage, authnz.QNameCDocLogin)
		if err != nil {
			return err
		}
		cdocLogin, err := args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		cdocLogin.PutInt32(authnz.Field_ProfileClusterID, profileCluster)
		cdocLogin.PutBytes(field_PwdHash, pwdSaltedHash)
		cdocLogin.PutString(Field_AppName, appName)
		cdocLogin.PutInt32(authnz.Field_SubjectKind, args.ArgumentObject.AsInt32(authnz.Field_SubjectKind))
		cdocLogin.PutString(authnz.Field_LoginHash, GetLoginHash(loginStr))
		cdocLogin.PutRecordID(appdef.SystemField_ID, 1)
		cdocLogin.PutString(authnz.Field_WSKindInitializationData, wsKindInitializationData)

		return
	}
}

// sys/registry, appWorkspace, triggered by CDoc<Login>
var projectorLoginIdx = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return event.CUDs(func(rec istructs.ICUDRow) (err error) {
		if rec.QName() != authnz.QNameCDocLogin {
			return nil
		}
		kb, err := s.KeyBuilder(state.ViewRecordsStorage, QNameViewLoginIdx)
		if err != nil {
			return
		}
		kb.PutInt64(field_AppWSID, int64(event.Workspace()))
		kb.PutString(field_AppIDLoginHash, rec.AsString(Field_AppName)+"/"+rec.AsString(authnz.Field_LoginHash))

		vb, err := intents.NewValue(kb)
		if err != nil {
			return
		}
		vb.PutInt64(field_CDocLoginID, int64(rec.AsRecordID(appdef.SystemField_ID)))
		return nil
	})
}
