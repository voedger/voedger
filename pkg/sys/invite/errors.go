/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"errors"
)

var (
	ErrInviteNotExists               = errors.New("invite not exists")
	errInviteExpired                 = errors.New("invite expired")
	errInviteTemplateInvalid         = errors.New("invite template invalid, it must be prefixed with 'text:' or 'resource:'")
	errInviteVerificationCodeInvalid = errors.New("invite verification code invalid")
	ErrInviteStateInvalid            = errors.New("invite state invalid")

	// [~server.invites.invite/err.State~impl]
	ErrReInviteNotAllowedForState = errors.New("re-invite not allowed for state")
)
