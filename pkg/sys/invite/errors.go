/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"errors"
)

var (
	errInviteNotExists               = errors.New("invite not exists")
	errInviteExpired                 = errors.New("invite expired")
	errInviteTemplateInvalid         = errors.New("invite template invalid, it must be prefixed with 'text:' or 'resource:'")
	errInviteVerificationCodeInvalid = errors.New("invite verification code invalid")
	errInviteStateInvalid            = errors.New("invite state invalid")
)
