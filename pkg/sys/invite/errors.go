/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"errors"
)

const errControllingInviteNotIdentifiedMessageFormat = "A workspace membership is already active for canonical login %q. The existing accepted invitation must be cancelled manually before another invitation can be accepted."

var (
	ErrInviteNotExists                = errors.New("invite not exists")
	ErrControllingInviteNotIdentified = errors.New("controlling invitation not identified")
	errInviteExpired                  = errors.New("invite expired")
	errInviteTemplateInvalid          = errors.New("invite template invalid, it must be prefixed with 'text:' or 'resource:'")
	errInviteVerificationCodeInvalid  = errors.New("invite verification code invalid")
	ErrInviteStateInvalid             = errors.New("invite state invalid")

	// [~server.invites.invite/err.State~impl]
	ErrReInviteNotAllowedForState = errors.New("re-invite not allowed for state")

	ErrRoleInvalid   = errors.New("invalid role")
	ErrRolesEmpty    = errors.New("roles must not be empty")
	ErrSystemRole    = errors.New("system roles cannot be assigned via invite")
	ErrRoleNotFound  = errors.New("role not found in workspace")
	ErrRoleDuplicate = errors.New("duplicate role")
)
