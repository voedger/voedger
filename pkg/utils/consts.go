/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io/fs"
	"math"
	"syscall"
	"time"
)

const (
	Authorization                                              = "Authorization"
	ContentType                                                = "Content-Type"
	ApplicationJSON                                            = "application/json"
	BearerPrefix                                               = "Bearer "
	shortRetryDelay                                            = 100 * time.Millisecond
	longRetryDelay                                             = time.Second
	shortRetriesAmount                                         = 10
	CRC16Mask                                                  = uint32(math.MaxUint32 >> 16)
	EmailTemplatePrefix_Text                                   = "text:"
	emailTemplatePrefix_Resource                               = "resource:"
	emailVerificationCodeLength                                = 6
	emailVerificationCodeSymbols                               = "1234567890"
	maxByte                                                    = ^byte(0)
	byteRangeToEmailVerifcationSymbolsRangeCoeff               = (float32(maxByte) + 1) / float32(len(emailVerificationCodeSymbols))
	requestRetryDelayOnConnRefused                             = 20 * time.Millisecond
	requestRetryTimeout                                        = 4 * time.Second
	WSAECONNRESET                                syscall.Errno = 10054
	WSAECONNREFUSED                              syscall.Errno = 10061
	FileMode_rwxrwxrwx                           fs.FileMode   = 0777 // default for directory
	FileMode_rw_rw_rw_                           fs.FileMode   = 0666 // default for file
)
