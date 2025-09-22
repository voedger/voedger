/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"math"
)

const (
	BlobName                      = "Blob-Name"
	CRC16Mask                     = uint32(math.MaxUint32 >> 16)
	EmailTemplatePrefix_Text      = "text:"
	emailTemplatePrefix_Resource  = "resource:"
	emailVerificationCodeLength   = 6
	emailVerificationCodeAlphabet = "1234567890"
	lowercaseDigitsAlphabet       = "abcdefghijklmnopqrstuvwxyz234567"
	deviceLoginAndPwdLen          = 26
)
