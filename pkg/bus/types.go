/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

type Request struct {
	Method      string
	WSID        istructs.WSID
	PartitionID istructs.PartitionID
	Header      map[string]string `json:",omitempty"`
	Resource    string
	Query       map[string]string `json:",omitempty"`
	Body        []byte              `json:",omitempty"`
	AppQName    string
	Host        string // used by authenticator to emit Host principal
}

type ResponseMeta struct {
	ContentType string
	StatusCode  int
}

type implIRequestSender struct {
	timeout        SendTimeout
	tm             coreutils.ITime
	requestHandler RequestHandler
}

type SendTimeout time.Duration

type implIResponseSenderCloseable struct {
	ch          chan any
	clientCtx   context.Context
	sendTimeout SendTimeout
	tm          coreutils.ITime
	resultErr   *error
}

type implIResponder struct {
	respSender     IResponseSenderCloseable
	inited         bool
	responseMetaCh chan ResponseMeta
}
