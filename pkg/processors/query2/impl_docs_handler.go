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

type docsHandler struct {
}

var _ IApiPathHandler = (*docsHandler)(nil) // ensure that queryHandler implements IApiPathHandler

func (h *docsHandler) IsArrayResult() bool {
	return false
}

func (h *docsHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	// TODO: implement rate limits check
	return nil
}

func (h *docsHandler) SetRequestType(ctx context.Context, qw *queryWork) error {
	switch qw.iWorkspace {
	case nil:
		// workspace is dummy
		if qw.iQuery = appdef.Query(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iQuery == nil {
			if qw.iDoc = appdef.CDoc(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iDoc != nil {
				return nil
			}
			if qw.iDoc = appdef.WDoc(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iDoc != nil {
				return nil
			}
			if qw.iRecord = appdef.CRecord(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iRecord != nil {
				return nil
			}
			if qw.iRecord = appdef.WRecord(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iRecord != nil {
				return nil
			}
		}
	default:
		if qw.iDoc = appdef.CDoc(qw.iWorkspace.Type, qw.msg.QName()); qw.iDoc != nil {
			return nil
		}
		if qw.iDoc = appdef.WDoc(qw.iWorkspace.Type, qw.msg.QName()); qw.iDoc != nil {
			return nil
		}
		if qw.iRecord = appdef.CRecord(qw.iWorkspace.Type, qw.msg.QName()); qw.iRecord != nil {
			return nil
		}
		if qw.iRecord = appdef.WRecord(qw.iWorkspace.Type, qw.msg.QName()); qw.iRecord != nil {
			return nil
		}
	}
	return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("document or record %s is not defined in %v", qw.msg.QName(), qw.iWorkspace))
}

func (h *docsHandler) SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	qw.resultType = qw.iDoc
	if qw.resultType == nil {
		qw.resultType = qw.iRecord
	}
	return nil
}

func (h *docsHandler) RequestOpKind() appdef.OperationKind {
	return appdef.OperationKind_Select
}

func (h *docsHandler) AuthorizeResult(ctx context.Context, qw *queryWork) (err error) {
	ws := qw.iWorkspace
	if ws == nil {
		return fmt.Errorf("workspace is nil")
	}
	var requestedFields []string
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		requestedFields = qw.queryParams.Constraints.Keys
	} else {
		var structure appdef.IStructure
		if qw.iDoc != nil {
			structure = qw.iDoc
		} else {
			structure = qw.iRecord
		}
		for _, field := range structure.Fields() {
			requestedFields = append(requestedFields, field.Name())
		}
	}
	// TODO: what to do with included objects?
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

func (h *docsHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	oo := make([]*pipeline.WiredOperator, 0)
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Include) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Include", newInclude(qw, true)))
	}
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Keys", newKeys(qw.queryParams.Constraints.Keys)))
	}
	sender := &sender{
		responder:          qw.msg.Responder(),
		isArrayResponse:    false,
		contentType:        coreutils.ApplicationJSON,
		rowsProcessorErrCh: qw.rowsProcessorErrCh,
	}
	oo = append(oo, pipeline.WireAsyncOperator("Sender", sender))
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", oo[0], oo[1:]...)
	qw.responseWriterGetter = func() bus.IResponseWriter {
		return sender.respWriter
	}
	return
}

func (h *docsHandler) Exec(_ context.Context, qw *queryWork) (err error) {
	var rec istructs.IRecord

	if qw.iDoc != nil && qw.iDoc.Singleton() {
		if qw.msg.DocID() != 0 {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("document %s is singleton. DocID must be 0", qw.msg.QName()))
		}
		rec, err = qw.appStructs.Records().GetSingleton(qw.msg.WSID(), qw.msg.QName())
		if err != nil {
			return err
		}
		if rec.QName() == appdef.NullQName {
			return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("singleton %s not found", qw.msg.QName()))
		}
	} else {
		rec, err = qw.appStructs.Records().Get(qw.msg.WSID(), true, istructs.RecordID(qw.msg.DocID()))
		if err != nil {
			return err
		}
		if rec.QName() == appdef.NullQName {
			if qw.iDoc != nil {
				return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("document %s with ID %d not found", qw.msg.QName(), qw.msg.DocID()))
			}
			return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("record %s with ID %d not found", qw.msg.QName(), qw.msg.DocID()))
		}
	}
	obj := objectBackedByMap{}
	obj.data = coreutils.FieldsToMap(rec, qw.appStructs.AppDef())
	return qw.callbackFunc(obj)
}
