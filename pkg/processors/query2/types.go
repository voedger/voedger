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
	Partition() istructs.PartitionID
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
