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
	Method   string
	WSID     istructs.WSID // as it came in the request, could be pseudo
	Header   map[string][]string
	Resource string
	Query    map[string][]string
	Body     []byte
	AppQName string
	Host     string // used by authenticator to emit Host principal
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
