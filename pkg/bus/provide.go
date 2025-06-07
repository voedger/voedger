/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "github.com/voedger/voedger/pkg/goutils/timeu"

func NewIRequestSender(tm timeu.ITime, sendTimeout SendTimeout, requestHandler RequestHandler) IRequestSender {
	return &implIRequestSender{
		timeout:        sendTimeout,
		tm:             tm,
		requestHandler: requestHandler,
	}
}
