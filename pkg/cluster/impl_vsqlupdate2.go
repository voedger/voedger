/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

type vSqlUpdate2Result struct {
	istructs.NullObject
	logWLogOffset istructs.Offset
	cudWLogOffset istructs.Offset
}

func (r *vSqlUpdate2Result) AsInt64(name string) int64 {
	switch name {
	case field_LogWLogOffset:
		return int64(r.logWLogOffset)
	case field_CUDWLogOffset:
		return int64(r.cudWLogOffset)
	}
	return 0
}

func (r *vSqlUpdate2Result) QName() appdef.QName { return qNameVSqlUpdate2Result }

func provideExecQryVSqlUpdate2(federation federation.IFederation, itokens itokens.ITokens, time timeu.ITime,
	asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
		query := args.ArgumentObject.AsString(field_Query)
		update, err := parseAndValidateQuery(args.Workpiece, query, asp)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}
		if update.Kind == dml.OpKind_InsertTable {
			return coreutils.NewHTTPError(http.StatusBadRequest,
				fmt.Errorf("'insert table' is not supported by q.cluster.VSqlUpdate2; use c.cluster.VSqlUpdate"))
		}

		logWLogOffset, err := logVSqlUpdate(federation, itokens, args.WSID, query)
		if err != nil {
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}

		cudWLogOffset, err := dispatchDML(update, federation, itokens, time, nil, nil)
		if err != nil {
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}

		return callback(&vSqlUpdate2Result{logWLogOffset: logWLogOffset, cudWLogOffset: cudWLogOffset})
	}
}

func logVSqlUpdate(federation federation.IFederation, itokens itokens.ITokens, wsid istructs.WSID, query string) (istructs.Offset, error) {
	sysToken, err := payloads.GetSystemPrincipalToken(itokens, istructs.AppQName_sys_cluster)
	if err != nil {
		// notest
		return 0, err
	}
	body := fmt.Sprintf(`{"args":{%q:%q}}`, field_Query, query)
	resp, err := federation.Func(fmt.Sprintf("api/%s/%d/c.cluster.LogVSqlUpdate", istructs.AppQName_sys_cluster, wsid), body,
		httpu.WithAuthorizeBy(sysToken))
	if err != nil {
		return 0, err
	}
	return resp.CurrentWLogOffset, nil
}
