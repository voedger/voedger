/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package payloads

import (
	"encoding/json"
	"errors"
	"fmt"
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
	if len(p.Roles) < len(wire.Roles) {
		return errors.New("payload has fewer roles than token claims")
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
		if scope == nil || (*scope != RoleScopeWorkspacesOwnedBy && *scope != RoleScopeWorkspace) {
			return fmt.Errorf("roles[%d].scope must be %d (owned-by) or %d (workspace)", i, RoleScopeWorkspacesOwnedBy, RoleScopeWorkspace)
		}
	}
	return nil
}
