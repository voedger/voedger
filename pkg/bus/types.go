/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

type Request struct {
	Method   string
	WSID     istructs.WSID // as it came in the request, could be pseudo
	Header   map[string]string
	Resource string
	Body     []byte
	AppQName appdef.AppQName
	Host     string // used by authenticator to emit Host principal

	// apiV2
	Query          map[string]string
	QName          appdef.QName // e.g. DocName, extension QName, role Qname
	WorkspaceQName appdef.QName // actually wsKind
	IsAPIV2        bool
	DocID          istructs.IDType
	ApiPath        int
}

type ResponseMeta struct {
	ContentType string
	StatusCode  int
	mode        RespondMode
}

type RespondMode int

const (
	RespondMode_ApiArray RespondMode = iota
	RespondMode_Single
)

type implIRequestSender struct {
	timeout        SendTimeout
	tm             coreutils.ITime
	requestHandler RequestHandler
}

type SendTimeout time.Duration

type implResponseWriter struct {
	ch          chan any
	clientCtx   context.Context
	sendTimeout SendTimeout
	tm          coreutils.ITime
	resultErr   *error
}

type implIResponder struct {
	respWriter     *implResponseWriter
	responseMetaCh chan ResponseMeta
	started        bool
}
