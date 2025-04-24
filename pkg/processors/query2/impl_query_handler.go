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

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
)

func queryHandler() apiPathHandler {
	return apiPathHandler{
		isArrayResult:   true,
		requestOpKind:   appdef.OperationKind_Execute,
		checkRateLimit:  queryRateLimitExceeded,
		setRequestType:  querySetRequestType,
		setResultType:   querySetResultType,
		authorizeResult: queryAuthorizeResult,
		rowsProcessor:   queryRowsProcessor,
		exec:            queryExec,
	}
}
func querySetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
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
	qw.resultType = ws.Type(qNameResultType)
	if qw.resultType.Kind() == appdef.TypeKind_null {
		return coreutils.NewHTTPError(http.StatusBadRequest, fmt.Errorf("%s query result type %s does not exist in %v", qw.iQuery.QName(), qNameResultType, ws))
	}
	return nil
}
func queryAuthorizeResult(ctx context.Context, qw *queryWork) error {
	// the entire result is allowed if execute on query is granted
	return nil
}
func queryRowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	result := qw.appStructs.AppDef().Type(qw.iQuery.QName()).(appdef.IQuery).Result()
	if result == nil {
		return nil
	}
	oo := make([]*pipeline.WiredOperator, 0)
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Include) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Include", newInclude(qw, false)))
	}
	if qw.queryParams.Constraints != nil && (len(qw.queryParams.Constraints.Order) != 0 || qw.queryParams.Constraints.Skip > 0 || qw.queryParams.Constraints.Limit > 0) {
		oo = append(oo, pipeline.WireAsyncOperator("Aggregator", newAggregator(qw.queryParams)))
	}
	resultType := qw.appStructs.AppDef().Type(result.QName())
	o, err := newFilter(qw, resultType.(appdef.IWithFields).Fields())
	if err != nil {
		return err
	}
	if o != nil {
		oo = append(oo, pipeline.WireAsyncOperator("Filter", o))
	}
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Keys", newKeys(qw.queryParams.Constraints.Keys)))
	}
	sender := &sender{responder: qw.msg.Responder(), isArrayResponse: true}
	oo = append(oo, pipeline.WireAsyncOperator("Sender", sender))
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", oo[0], oo[1:]...)
	qw.responseWriterGetter = func() bus.IResponseWriter {
		return sender.respWriter
	}
	return
}
func queryExec(ctx context.Context, qw *queryWork) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%v\n%s", r, stack)
		}
	}()
	return qw.appPart.Invoke(ctx, qw.msg.QName(), qw.state, qw.state)
}
