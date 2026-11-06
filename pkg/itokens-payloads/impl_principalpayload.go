/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package payloads

import (
	"encoding/json"
	"errors"
)

type roleScopeClaim struct {
	Scope json.RawMessage
}

func (p *PrincipalPayload) ValidateTokenClaims(claims []byte) error {
	var wire struct {
		Roles      []roleScopeClaim
		IsAPIToken bool
	}
	if err := json.Unmarshal(claims, &wire); err != nil {
		return err
	}
	for i, role := range wire.Roles {
		if len(role.Scope) == 0 {
			p.Roles[i].Scope = RoleScopeWorkspacesOwnedBy
			if wire.IsAPIToken {
				p.Roles[i].Scope = RoleScopeWorkspace
			}
			continue
		}
		var scope *RoleScope
		if err := json.Unmarshal(role.Scope, &scope); err != nil {
			return err
		}
		if scope == nil {
			return errors.New("role scope cannot be null")
		}
		if *scope != RoleScopeWorkspacesOwnedBy && *scope != RoleScopeWorkspace {
			return errors.New("invalid role scope")
		}
	}
	return nil
}
