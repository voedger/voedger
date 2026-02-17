/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "errors"

// happens when router is busy writing to the slow http client and does not read the response channel
var ErrSendResponseTimeout = errors.New("timeout sending response")
