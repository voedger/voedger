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
)

type ApiPath int

type QueryParams struct {
	Constraints *Constraints           `json:"constraints"`
	Argument    map[string]interface{} `json:"argument,omitempty"`
}

type Constraints struct {
	Order   []string               `json:"order"`
	Limit   int                    `json:"limit"`
	Skip    int                    `json:"skip"`
	Include []string               `json:"include"`
	Keys    []string               `json:"keys"`
	Where   map[string]interface{} `json:"where"`
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
