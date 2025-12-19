/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	"github.com/voedger/voedger/pkg/state"
)

func queryRateLimitExceeded(ctx context.Context, qw *queryWork) error {
	if qw.appStructs.IsFunctionRateLimitsExceeded(qw.msg.QName(), qw.msg.WSID()) {
		return coreutils.NewSysError(http.StatusTooManyRequests)
	}
	return nil
}
func querySetRequestType(ctx context.Context, qw *queryWork) error {
	if qw.iQuery = appdef.Query(qw.iWorkspace.Type, qw.msg.QName()); qw.iQuery == nil {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("query %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
	}
	return nil
}

type queryProcessorMetrics struct {
	vvm     string
	app     appdef.AppQName
	metrics imetrics.IMetrics
}

var _ queryprocessor.IMetrics = (*queryProcessorMetrics)(nil) // ensure that queryProcessorMetrics implements IMetrics

func (m *queryProcessorMetrics) Increase(metricName string, valueDelta float64) {
	m.metrics.IncreaseApp(metricName, m.vvm, m.app, valueDelta)
}

type queryWork struct {
	// input
	msg      IQueryMessage
	appParts appparts.IAppPartitions
	// work
	requestData          map[string]interface{}
	state                state.IHostState
	queryParams          QueryParams
	appPart              appparts.IAppPartition
	appStructs           istructs.IAppStructs
	resultType           appdef.IType
	execQueryArgs        istructs.ExecQueryArgs
	maxPrepareQueries    int
	rowsProcessor        pipeline.IAsyncPipeline
	rowsProcessorErrCh   chan error // will contain the first error from rowProcessor if any. The rest of errors in rowsProcessor will be just logged
	metrics              queryprocessor.IMetrics
	principals           []iauthnz.Principal
	roles                []appdef.QName
	secretReader         isecrets.ISecretReader
	iWorkspace           appdef.IWorkspace
	iQuery               appdef.IQuery
	iView                appdef.IView
	iDoc                 appdef.IDoc
	iRecord              appdef.IContainedRecord
	wsDesc               istructs.IRecord
	callbackFunc         istructs.ExecQueryCallback
	responseWriterGetter func() bus.IResponseWriter
	apiPathHandler       apiPathHandler
	federation           federation.IFederation
	profileWSID          istructs.WSID
}

var _ pipeline.IWorkpiece = (*queryWork)(nil) // ensure that queryWork implements pipeline.IWorkpiece

func (qw *queryWork) Release() {
	if ap := qw.appPart; ap != nil {
		qw.appStructs = nil
		qw.appPart = nil
		ap.Release()
	}
}

// borrows app partition for query
func (qw *queryWork) borrow() (err error) {
	if qw.appPart, err = qw.appParts.Borrow(qw.msg.AppQName(), qw.msg.PartitionID(), appparts.ProcessorKind_Query); err != nil {
		return err
	}
	qw.appStructs = qw.appPart.AppStructs()
	return nil
}

func (qw *queryWork) isDeveloper() bool {
	for _, prn := range qw.principals {
		if prn.Kind == iauthnz.PrincipalKind_Role && prn.QName == appdef.QNameRoleDeveloper {
			return true
		}
	}
	return false
}

func (qw *queryWork) isDocSingleton() bool {
	if qw.iDoc == nil {
		return false
	}
	iSingleton, ok := qw.iDoc.(appdef.ISingleton)
	if !ok {
		return false
	}
	return iSingleton.Singleton()
}

func newQueryWork(msg IQueryMessage, appParts appparts.IAppPartitions,
	maxPrepareQueries int, metrics *queryProcessorMetrics, secretReader isecrets.ISecretReader, federation federation.IFederation) *queryWork {
	return &queryWork{
		msg:                msg,
		appParts:           appParts,
		requestData:        make(map[string]interface{}),
		maxPrepareQueries:  maxPrepareQueries,
		metrics:            metrics,
		secretReader:       secretReader,
		rowsProcessorErrCh: make(chan error, 1),
		queryParams:        msg.QueryParams(),
		federation:         federation,
	}
}

func borrowAppPart(_ context.Context, qw *queryWork) error {
	switch err := qw.borrow(); {
	case err == nil:
		return nil
	case errors.Is(err, appparts.ErrNotAvailableEngines), errors.Is(err, appparts.ErrNotFound): // partition is not deployed yet -> ErrNotFound
		return coreutils.WrapSysError(err, http.StatusServiceUnavailable)
	default:
		return coreutils.WrapSysError(err, http.StatusBadRequest)
	}
}

func operator(name string, doSync func(ctx context.Context, qw *queryWork) (err error)) *pipeline.WiredOperator {
	return pipeline.WireFunc(name, doSync)
}

func NewIQueryMessage(requestCtx context.Context, appQName appdef.AppQName, wsid istructs.WSID, responder bus.IResponder,
	queryParams QueryParams, docID istructs.IDType, apiPath processors.APIPath,
	qName appdef.QName, partition istructs.PartitionID, host string, token string, workspaceQName appdef.QName, headerAccept string) IQueryMessage {
	return &implIQueryMessage{
		appQName:       appQName,
		wsid:           wsid,
		responder:      responder,
		queryParams:    queryParams,
		docID:          docID,
		apiPath:        apiPath,
		requestCtx:     requestCtx,
		qName:          qName,
		partition:      partition,
		host:           host,
		token:          token,
		workspaceQName: workspaceQName,
		headerAccept:   headerAccept,
	}
}

func (qw *queryWork) getArraySender() (pipeline.IAsyncOperator, func() bus.IResponseWriter) {
	res := &arraySender{
		sender: sender{
			responder:          qw.msg.Responder(),
			rowsProcessorErrCh: qw.rowsProcessorErrCh,
		},
	}
	return res, func() bus.IResponseWriter {
		return res.respWriter
	}
}

func (qw *queryWork) getObjectSender() pipeline.IAsyncOperator {
	return &objectSender{
		sender: sender{
			responder:          qw.msg.Responder(),
			rowsProcessorErrCh: qw.rowsProcessorErrCh,
		},
		contentType: httpu.ContentType_ApplicationJSON,
	}
}

func (qw *queryWork) SetPrincipals(prns []iauthnz.Principal) {
	qw.principals = prns
}
