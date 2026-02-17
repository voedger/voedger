/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "github.com/voedger/voedger/pkg/goutils/timeu"

func NewIRequestSender(tm timeu.ITime, requestHandler RequestHandler) IRequestSender {
	return &implIRequestSender{
		tm:             tm,
		requestHandler: requestHandler,
	}
}
