/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
)

type viewHandler struct {
}

var _ IApiPathHandler = (*viewHandler)(nil) // ensure that viewHandler implements IApiPathHandler

func (h *viewHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	// TODO: implement rate limits check
	return nil
}

func (h *viewHandler) CheckType(ctx context.Context, qw *queryWork) error {
	switch qw.iWorkspace {
	case nil:
		// workspace is dummy
		if qw.iView = appdef.View(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iView == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("view %s does not exist", qw.msg.QName()))
		}
	default:
		if qw.iView = appdef.View(qw.iWorkspace.Type, qw.msg.QName()); qw.iView == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("view %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
		}
	}
	return nil
}

func (h *viewHandler) ResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	qw.resultType = qw.iView
	return nil
}

func (h *viewHandler) AuthorizeRequest(ctx context.Context, qw *queryWork) error {
	/*
		TODO: doesn't work with pkg/sys/it/impl_qpv2_test.go
		ws := qw.iWorkspace
		if ws == nil {
			// workspace is dummy
			ws = qw.iView.Workspace()
		}
		ok, err := qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, qw.msg.QName(), nil, qw.roles)
		if err != nil {
			return err
		}
		if !ok {
			return coreutils.WrapSysError(errors.New(""), http.StatusForbidden)
		}
	*/
	return nil
}

func (h *viewHandler) AuthorizeResult(ctx context.Context, qw *queryWork) error {
	// TODO: implement result authorization
	return nil
}

func (h *viewHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	resp := qw.msg.Responder().InitResponse(bus.ResponseMeta{
		ContentType: coreutils.ApplicationJSON,
		StatusCode:  http.StatusOK,
	})
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", pipeline.WireAsyncFunc("Send object", func(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
		return nil, resp.Send(work.(objectBackedByMap).data)
	}))
	return
}
func (h *viewHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	kb := qw.appStructs.ViewRecords().KeyBuilder(qw.iView.QName())
	kb.PutFromJSON(qw.queryParams.Constraints.Where)
	return qw.appStructs.ViewRecords().Read(ctx, qw.msg.WSID(), kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		obj := objectBackedByMap{}
		obj.data = coreutils.FieldsToMap(key, qw.appStructs.AppDef(), coreutils.WithNonNilsOnly())
		for k, v := range coreutils.FieldsToMap(value, qw.appStructs.AppDef(), coreutils.WithNonNilsOnly()) {
			obj.data[k] = v
		}
		return qw.callbackFunc(obj)
	})
}
