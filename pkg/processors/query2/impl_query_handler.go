/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
)

type queryHandler struct {
}

var _ IApiPathHandler = (*queryHandler)(nil) // ensure that queryHandler implements IApiPathHandler

func (h *queryHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	if qw.appStructs.IsFunctionRateLimitsExceeded(qw.msg.QName(), qw.msg.WSID()) {
		return coreutils.NewSysError(http.StatusTooManyRequests)
	}
	return nil
}

func (h *queryHandler) SetRequestType(ctx context.Context, qw *queryWork) error {
	switch qw.iWorkspace {
	case nil:
		// workspace is dummy
		if qw.iQuery = appdef.Query(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iQuery == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist", qw.msg.QName()))
		}
	default:
		if qw.iQuery = appdef.Query(qw.iWorkspace.Type, qw.msg.QName()); qw.iQuery == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
		}
	}
	return nil
}

func (h *queryHandler) SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	qw.resultType = qw.iQuery.Result()
	if qw.resultType == nil || qw.resultType.QName() != appdef.QNameANY {
		return nil
	}
	// ANY -> exact type according to PrepareArgs
	iResource := qw.appStructs.Resources().QueryResource(qw.msg.QName())
	var iQueryFunc istructs.IQueryFunction
	if iResource.Kind() != istructs.ResourceKind_null {
		iQueryFunc = iResource.(istructs.IQueryFunction)
	} else {
		for _, qry := range statelessResources.Queries {
			if qry.QName() == qw.msg.QName() {
				iQueryFunc = qry
				break
			}
		}
	}
	qNameResultType := iQueryFunc.ResultType(qw.execQueryArgs.PrepareArgs)

	ws := qw.iWorkspace
	if ws == nil {
		// workspace is dummy
		ws = qw.iQuery.Workspace()
	}
	qw.resultType = ws.Type(qNameResultType)
	if qw.resultType.Kind() == appdef.TypeKind_null {
		return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%s query result type %s does not exist in %v", qw.iQuery.QName(), qNameResultType, ws))
	}
	return nil
}

func (h *queryHandler) RequestOpKind() appdef.OperationKind {
	return appdef.OperationKind_Execute
}

func (h *queryHandler) AuthorizeResult(ctx context.Context, qw *queryWork) error {
	// the entire result is allowed if execute on query is granted
	return nil
}

func (h *queryHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	oo := make([]*pipeline.WiredOperator, 0)
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Include) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Include", newInclude(qw)))
	}
	if qw.queryParams.Constraints != nil && (len(qw.queryParams.Constraints.Order) != 0 || qw.queryParams.Constraints.Skip > 0 || qw.queryParams.Constraints.Limit > 0) {
		oo = append(oo, pipeline.WireAsyncOperator("Aggregator", newAggregator(qw.queryParams)))
	}
	o, err := newFilter(qw, qw.appStructs.AppDef().Type(qw.appStructs.AppDef().Type(qw.iQuery.QName()).(appdef.IQuery).Result().QName()).(appdef.IWithFields).Fields())
	if err != nil {
		return
	}
	if o != nil {
		oo = append(oo, pipeline.WireAsyncOperator("Filter", o))
	}
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Keys", newKeys(qw.queryParams.Constraints.Keys)))
	}
	sender := &sender{responder: qw.msg.Responder()}
	oo = append(oo, pipeline.WireAsyncOperator("Sender", sender))
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", oo[0], oo[1:]...)
	qw.responseWriterGetter = func() bus.IResponseWriter {
		return sender.respWriter
	}
	return
}

func (h *queryHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	now := time.Now()
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%v\n%s", r, stack)
		}
		qw.metrics.Increase(queryprocessor.Metric_ExecSeconds, time.Since(now).Seconds())
	}()
	return qw.appPart.Invoke(ctx, qw.msg.QName(), qw.state, qw.state)
}
