/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package verifier

import (
	"net/http"

	"github.com/voedger/voedger/pkg/coreutils"
)

var (
	ErrVerificationCodeExpired = coreutils.NewHTTPErrorf(http.StatusBadRequest, "your verification code has expired")
	ErrInvalidVerificationCode = coreutils.NewHTTPErrorf(http.StatusBadRequest, "invalid verification code")
)
