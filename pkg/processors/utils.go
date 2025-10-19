/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func CheckResponseIntent(st state.IHostState) error {
	kb, err := st.KeyBuilder(sys.Storage_Response, appdef.NullQName)
	if err != nil {
		// notest
		return err
	}
	respIntent := st.FindIntent(kb)
	if respIntent == nil {
		return nil
	}
	respIntentValue := respIntent.BuildValue()
	statusCode := respIntentValue.AsInt32(sys.Storage_Response_Field_StatusCode)
	if statusCode == http.StatusOK {
		return nil
	}
	return coreutils.NewHTTPErrorf(int(statusCode), respIntentValue.AsString(sys.Storage_Response_Field_ErrorMessage))
}

// returns ErrWSNotInited
func GetWSDesc(wsid istructs.WSID, appStructs istructs.IAppStructs) (wsDesc istructs.IRecord, err error) {
	wsDesc, err = appStructs.Records().GetSingleton(wsid, authnz.QNameCDocWorkspaceDescriptor)
	if err == nil && wsDesc.QName() == appdef.NullQName {
		err = ErrWSNotInited
	}
	return wsDesc, err
}

func GetRoles(principals []iauthnz.Principal) (roles []appdef.QName) {
	for _, prn := range principals {
		if prn.Kind != iauthnz.PrincipalKind_Role {
			continue
		}
		roles = append(roles, prn.QName)
	}
	return roles
}

func SetPrincipalsForAnonymousOnlyFunc(appDef appdef.IAppDef, funcQName appdef.QName, wsid istructs.WSID, setter interface{ SetPrincipals([]iauthnz.Principal) }) (ok bool) {
	queryType := appDef.Type(funcQName)
	rulesForQuery := []appdef.IACLRule{}
	for _, acl := range appDef.ACL() {
		if acl.Filter().Match(queryType) {
			rulesForQuery = append(rulesForQuery, acl)
		}
	}
	if len(rulesForQuery) == 1 {
		if len(rulesForQuery[0].Ops()) == 1 &&
			rulesForQuery[0].Ops()[0] == appdef.OperationKind_Execute &&
			rulesForQuery[0].Policy() == appdef.PolicyKind_Allow &&
			rulesForQuery[0].Principal().Kind() == appdef.TypeKind_Role &&
			rulesForQuery[0].Principal().QName() == iauthnz.QNameRoleAnonymous {
			setter.SetPrincipals([]iauthnz.Principal{{
				Kind:  iauthnz.PrincipalKind_Role,
				WSID:  wsid,
				QName: iauthnz.QNameRoleAnonymous,
			}})
			return true
		}
	}
	return false
}

