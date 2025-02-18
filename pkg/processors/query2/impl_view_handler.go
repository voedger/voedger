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
	// ws := qw.iWorkspace
	// if ws == nil {
	// 	// workspace is dummy
	// 	ws = qw.iView.Workspace()
	// }
	// ok, err := qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, qw.msg.QName(), nil, qw.roles)
	// if err != nil {
	// 	return err
	// }
	// if !ok {
	// 	return coreutils.NewHTTPError(http.StatusForbidden, errors.New(""))
	// }
	return nil
}

func (h *viewHandler) AuthorizeResult(ctx context.Context, qw *queryWork) error {
	// if qw.iQuery.Result() != appdef.AnyType {
	// 	// will authorize result only if result is sys.Any
	// 	// otherwise each field is considered as allowed if EXECUTE ON QUERY is allowed
	// 	return nil
	// }
	// ws := qw.iWorkspace
	// if ws == nil {
	// 	// workspace is dummy
	// 	ws = qw.iQuery.Workspace()
	// }
	// for _, elem := range qw.queryParams.Elements() {
	// 	nestedPath := elem.Path().AsArray()
	// 	nestedType := qw.resultType
	// 	for _, nestedName := range nestedPath {
	// 		if len(nestedName) == 0 {
	// 			// root
	// 			continue
	// 		}
	// 		// incorrectness is excluded already on validation stage in [queryParams.validate]
	// 		containersOfNested := nestedType.(appdef.IWithContainers)
	// 		// container presence is checked already on validation stage in [queryParams.validate]
	// 		nestedContainer := containersOfNested.Container(nestedName)
	// 		nestedType = nestedContainer.Type()
	// 	}
	// 	requestedfields := []string{}
	// 	for _, resultField := range elem.ResultFields() {
	// 		requestedfields = append(requestedfields, resultField.Field())
	// 	}

	// 	// TODO: temporary solution. To be eliminated after implementing ACL in VSQL for Air
	// 	ok := oldacl.IsOperationAllowed(appdef.OperationKind_Select, nestedType.QName(), requestedfields, oldacl.EnrichPrincipals(qw.principals, qw.msg.WSID()))
	// 	if !ok {
	// 		if ok, err = qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, nestedType.QName(), requestedfields, qw.roles); err != nil {
	// 			return err
	// 		}
	// 	}
	// 	if !ok {
	// 		return coreutils.NewSysError(http.StatusForbidden)
	// 	}
	// }
	return nil
}

func (h *viewHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	oo := make([]*pipeline.WiredOperator, 0)
	if len(qw.queryParams.Constraints.Order) != 0 || qw.queryParams.Constraints.Skip > 0 || qw.queryParams.Constraints.Limit > 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Aggregator", newAggregator(qw.queryParams)))
	}
	if len(qw.queryParams.Constraints.Keys) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Keys", newKeys(qw.queryParams.Constraints.Keys)))
	}
	oo = append(oo, pipeline.WireAsyncOperator("Sender", &sender{responder: qw.msg.Responder()}))
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", oo[0], oo[1:]...)
	return
}
func (h *viewHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	kb := qw.appStructs.ViewRecords().KeyBuilder(qw.iView.QName())
	kb.PutFromJSON(qw.queryParams.Constraints.Where)
	return qw.appStructs.ViewRecords().Read(ctx, qw.msg.WSID(), kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		obj := objectBackedByMap{}
		obj.data = coreutils.FieldsToMap(key, qw.appStructs.AppDef())
		for k, v := range coreutils.FieldsToMap(value, qw.appStructs.AppDef()) {
			obj.data[k] = v
		}
		return qw.callbackFunc(obj)
	})
}
