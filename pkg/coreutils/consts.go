/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io/fs"
	"math"
	"net"
	"syscall"
	"time"
)

const (
	Authorization                                 = "Authorization"
	ContentType                                   = "Content-Type"
	ContentDisposition                            = "Content-Disposition"
	Accept                                        = "Accept"
	ContentType_ApplicationJSON                   = "application/json"
	ContentType_ApplicationXBinary                = "application/x-binary"
	ContentType_TextPlain                         = "text/plain"
	ContentType_TextHTML                          = "text/html"
	ContentType_MultipartFormData                 = "multipart/form-data"
	BearerPrefix                                  = "Bearer "
	Localhost                                     = "127.0.0.1"
	shortRetryOn503Delay                          = 100 * time.Millisecond
	longRetryOn503Delay                           = time.Second
	shortRetriesOn503Amount                       = 10
	CRC16Mask                                     = uint32(math.MaxUint32 >> 16)
	EmailTemplatePrefix_Text                      = "text:"
	emailTemplatePrefix_Resource                  = "resource:"
	emailVerificationCodeLength                   = 6
	emailVerificationCodeAlphabet                 = "1234567890"
	lowercaseDigitsAlphabet                       = "abcdefghijklmnopqrstuvwxyz234567"
	retryOn_WSAECONNREFUSED_Delay                 = 20 * time.Millisecond
	retryOn_WSAECONNREFUSED_Timeout               = 4 * time.Second
	WSAECONNRESET                   syscall.Errno = 10054
	WSAECONNREFUSED                 syscall.Errno = 10061
	FileMode_rwxrwxrwx              fs.FileMode   = 0777 // default for directory
	FileMode_rw_rw_rw_              fs.FileMode   = 0666 // default for file
	maxHTTPRequestTimeout                         = time.Hour
	defaultRetryDelay                             = 500 * time.Millisecond
	deviceLoginAndPwdLen                          = 26
)

var (
	LocalhostIP = net.IPv4(127, 0, 0, 1)
)
