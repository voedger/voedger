/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 *
 * @author Daniil Solovyov
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
	"github.com/voedger/voedger/pkg/sys/collection"
)

// [~server.apiv2.docs/cmp.cdocsHandler~impl]
func cdocsHandler() apiPathHandler {
	return apiPathHandler{
		requestOpKind:   appdef.OperationKind_Select,
		isArrayResult:   true,
		checkRateLimit:  nil, // TODO: implement rate limit for CDocs
		setRequestType:  cdocsSetRequestType,
		setResultType:   cdocsSetResultType,
		authorizeResult: cdocsAuthorizeResult,
		rowsProcessor:   cdocsRowsProcessor,
		exec:            cdocsExec,
	}
}

func cdocsSetRequestType(_ context.Context, qw *queryWork) (err error) {
	var f appdef.FindType
	if qw.iWorkspace == nil {
		f = qw.appStructs.AppDef().Type
	} else {
		f = qw.iWorkspace.Type
	}
	if qw.iDoc = appdef.CDoc(f, qw.msg.QName()); qw.iDoc != nil {
		return
	}
	return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("document or record %s is not defined in %v", qw.msg.QName(), qw.iWorkspace))
}
func cdocsSetResultType(_ context.Context, qw *queryWork, _ istructsmem.IStatelessResources) (err error) {
	qw.resultType = qw.iDoc
	return
}
func cdocsAuthorizeResult(_ context.Context, qw *queryWork) (err error) {
	ws := qw.iWorkspace
	if ws == nil {
		return errWorkspaceIsNil
	}
	var requestedFields []string
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		requestedFields = qw.queryParams.Constraints.Keys
	} else {
		for _, field := range qw.iDoc.Fields() {
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
	return
}
func cdocsRowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	oo := make([]*pipeline.WiredOperator, 0)
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Include) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Include", newInclude(qw, true)))
	}
	if qw.queryParams.Constraints != nil && (len(qw.queryParams.Constraints.Order) != 0 || qw.queryParams.Constraints.Skip > 0 || qw.queryParams.Constraints.Limit > 0) {
		oo = append(oo, pipeline.WireAsyncOperator("Aggregator", newAggregator(qw.queryParams)))
	}
	if qw.queryParams.Constraints != nil && len(qw.queryParams.Constraints.Keys) != 0 {
		oo = append(oo, pipeline.WireAsyncOperator("Keys", newKeys(qw.queryParams.Constraints.Keys)))
	}
	sender := &sender{responder: qw.msg.Responder(), isArrayResponse: true}
	oo = append(oo, pipeline.WireAsyncOperator("Sender", sender))
	qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "CDocs rows processor", oo[0], oo[1:]...)
	qw.responseWriterGetter = func() bus.IResponseWriter {
		return sender.respWriter
	}
	return
}
func cdocsExec(ctx context.Context, qw *queryWork) (err error) {
	kb := qw.appStructs.ViewRecords().KeyBuilder(collection.QNameCollectionView)
	kb.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
	kb.PutQName(collection.Field_DocQName, qw.msg.QName())
	return qw.appStructs.ViewRecords().Read(ctx, qw.msg.WSID(), kb, func(_ istructs.IKey, value istructs.IValue) (err error) {
		r := value.AsRecord(collection.Field_Record)
		if r.QName() != qw.msg.QName() {
			return
		}
		obj := objectBackedByMap{}
		obj.data = coreutils.FieldsToMap(r, qw.appStructs.AppDef())
		return qw.callbackFunc(obj)
	})
}
