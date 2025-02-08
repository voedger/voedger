/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

type ApiPath int

type QueryParams struct {
	Constraints *Constraints
	Argument    map[string]interface{}
}

type Constraints struct {
	Order   []string
	Limit   int
	Skip    int
	Include []string
	Keys    []string
	Where   map[string]interface{}
}

type IQueryMessage interface {
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Responder() bus.IResponder
	QueryParams() QueryParams
	DocID() istructs.IDType
	ApiPath() ApiPath
	RequestCtx() context.Context
	QName() appdef.QName
	PartitionID() istructs.PartitionID
	Host() string
	Token() string
}

type IApiPathHandler interface {
	CheckRateLimit(ctx context.Context, qw *queryWork) error
	CheckType(ctx context.Context, qw *queryWork) error
	ResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error
	AuthorizeRequest(ctx context.Context, qw *queryWork) error
	AuthorizeResult(ctx context.Context, qw *queryWork) error
	RowsProcessor(ctx context.Context, qw *queryWork) error
	Exec(ctx context.Context, qw *queryWork) error
}

type implIQueryMessage struct {
	appQName    appdef.AppQName
	wsid        istructs.WSID
	responder   bus.IResponder
	queryParams QueryParams
	docID       istructs.IDType
	apiPath     ApiPath
	requestCtx  context.Context
	qName       appdef.QName
	partition   istructs.PartitionID
	host        string
	token       string
}

func (qm *implIQueryMessage) AppQName() appdef.AppQName {
	return qm.appQName
}

func (qm *implIQueryMessage) WSID() istructs.WSID {
	return qm.wsid
}

func (qm *implIQueryMessage) Responder() bus.IResponder {
	return qm.responder
}

func (qm *implIQueryMessage) QueryParams() QueryParams {
	return qm.queryParams
}
func (qm *implIQueryMessage) DocID() istructs.IDType {
	return qm.docID
}

func (qm *implIQueryMessage) ApiPath() ApiPath {
	return qm.apiPath
}

func (qm *implIQueryMessage) RequestCtx() context.Context {
	return qm.requestCtx
}
func (qm *implIQueryMessage) QName() appdef.QName {
	return qm.qName
}

func (qm *implIQueryMessage) PartitionID() istructs.PartitionID {
	return qm.partition
}

func (qm *implIQueryMessage) Host() string {
	return qm.host
}

func (qm *implIQueryMessage) Token() string {
	return qm.token
}
