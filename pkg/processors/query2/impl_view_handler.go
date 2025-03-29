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
	"github.com/voedger/voedger/pkg/processors/oldacl"
)

type viewHandler struct {
}

var _ IApiPathHandler = (*viewHandler)(nil) // ensure that viewHandler implements IApiPathHandler

func (h *viewHandler) IsArrayResult() bool {
	return true
}

func (h *viewHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	// TODO: implement rate limits check
	return nil
}

func (h *viewHandler) SetRequestType(ctx context.Context, qw *queryWork) error {
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

func (h *viewHandler) SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	qw.resultType = qw.iView
	return nil
}

func (h *viewHandler) RequestOpKind() appdef.OperationKind {
	return appdef.OperationKind_Select
}

func (h *viewHandler) AuthorizeResult(ctx context.Context, qw *queryWork) (err error) {
	if qw.resultType != appdef.AnyType {
		// will authorize result only if result is sys.Any
		// otherwise each field is considered as allowed if EXECUTE ON QUERY is allowed
		return nil
	}
	ws := qw.iWorkspace
	if ws == nil {
		// workspace is dummy
		panic("")
	}
	var requestedFields []string
	if len(qw.queryParams.Constraints.Keys) != 0 {
		requestedFields = qw.queryParams.Constraints.Keys
	} else {
		for _, field := range qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Key().Fields() {
			requestedFields = append(requestedFields, field.Name())
		}
	}
	// TODO: temporary solution. To be eliminated after implementing ACL in VSQL for Air
	ok := oldacl.IsOperationAllowed(appdef.OperationKind_Select, qw.resultType.QName(), requestedFields, oldacl.EnrichPrincipals(qw.principals, qw.msg.WSID()))
	if !ok {
		if ok, err = qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, qw.resultType.QName(), requestedFields, qw.roles); err != nil {
			return err
		}
	}
	if !ok {
		return coreutils.NewSysError(http.StatusForbidden)
	}
	return nil
}
func (h *viewHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	err = h.validateFields(qw)
	if err != nil {
		return
	}
	oo := make([]*pipeline.WiredOperator, 0)
	if len(qw.queryParams.Constraints.Include) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Include", newInclude(qw, false)))
	}
	if len(qw.queryParams.Constraints.Order) != 0 || qw.queryParams.Constraints.Skip > 0 || qw.queryParams.Constraints.Limit > 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Aggregator", newAggregator(qw.queryParams)))
	}
	fields := make([]appdef.IField, 0)
	fields = append(fields, qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Key().ClustCols().Fields()...)
	fields = append(fields, qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Value().Fields()...)
	o, err := newFilter(qw, fields)
	if err != nil {
		return
	}
	if o != nil {
		oo = append(oo, pipeline.WireAsyncOperator("Filter", o))
	}
	if len(qw.queryParams.Constraints.Keys) != 0 {
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
func (h *viewHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	kk, err := h.getKeys(qw)
	if err != nil {
		return
	}
	for i := range kk {
		err = qw.appStructs.ViewRecords().Read(ctx, qw.msg.WSID(), kk[i], func(key istructs.IKey, value istructs.IValue) (err error) {
			obj := objectBackedByMap{}
			obj.data = coreutils.FieldsToMap(key, qw.appStructs.AppDef())
			for k, v := range coreutils.FieldsToMap(value, qw.appStructs.AppDef()) {
				obj.data[k] = v
			}
			return qw.callbackFunc(obj)
		})
		if err != nil {
			return
		}
	}
	return
}
func (h *viewHandler) getKeys(qw *queryWork) (keys []istructs.IKeyBuilder, err error) {
	fields := qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Key().Fields()
	values := make([][]interface{}, 0, len(fields))
	for i, field := range fields {
		switch field.DataKind() {
		case appdef.DataKind_int32:
			vv, err := qw.queryParams.Constraints.Where.getAsInt32(field.Name())
			if err != nil {
				return nil, err
			}
			if vv == nil {
				continue
			}
			values = append(values, make([]interface{}, 0))
			for _, v := range vv {
				values[i] = append(values[i], v)
			}
		case appdef.DataKind_string:
			vv, err := qw.queryParams.Constraints.Where.getAsString(field.Name())
			if err != nil {
				return nil, err
			}
			if vv == nil {
				continue
			}
			values = append(values, make([]interface{}, 0))
			for _, v := range vv {
				values[i] = append(values[i], v)
			}
		default:
			// do nothing
		}
	}
	cc := getCombinations(values)
	keys = make([]istructs.IKeyBuilder, len(cc))
	for i, c := range cc {
		keys[i] = qw.appStructs.ViewRecords().KeyBuilder(qw.iView.QName())
		for j, intf := range c {
			switch v := intf.(type) {
			case int32:
				keys[i].PutInt32(fields[j].Name(), v)
			case string:
				keys[i].PutString(fields[j].Name(), v)
			}
		}
	}
	return
}
func (h *viewHandler) validateFields(qw *queryWork) (err error) {
	view := qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView)

	if qw.queryParams.Constraints == nil {
		return errConstraintsAreNull
	}
	if len(qw.queryParams.Constraints.Where) == 0 {
		return errWhereConstraintIsEmpty
	}
	for _, field := range view.Key().PartKey().Fields() {
		if _, ok := qw.queryParams.Constraints.Where[field.Name()]; !ok {
			return errWhereConstraintMustSpecifyThePartitionKey
		}
	}

	ff := make(map[string]bool)
	for _, field := range view.Fields() {
		ff[field.Name()] = true
	}
	for k := range qw.queryParams.Constraints.Where {
		if !ff[k] {
			return fmt.Errorf("%w: '%s'", errUnexpectedField, k)
		}
	}
	return
}
