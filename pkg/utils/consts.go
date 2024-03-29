/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"math"
	"time"
)

const (
	Authorization                                = "Authorization"
	ContentType                                  = "Content-Type"
	ApplicationJSON                              = "application/json"
	BearerPrefix                                 = "Bearer "
	shortRetryDelay                              = 100 * time.Millisecond
	longRetryDelay                               = time.Second
	shortRetriesAmount                           = 10
	CRC16Mask                                    = uint32(math.MaxUint32 >> 16)
	EmailTemplatePrefix_Text                     = "text:"
	emailTemplatePrefix_Resource                 = "resource:"
	emailVerificationCodeLength                  = 6
	emailVerificationCodeSymbols                 = "1234567890"
	maxByte                                      = ^byte(0)
	byteRangeToEmailVerifcationSymbolsRangeCoeff = (float32(maxByte) + 1) / float32(len(emailVerificationCodeSymbols))
)
