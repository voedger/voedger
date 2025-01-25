/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newRole() *Role {
	return &Role{}
}

func (r *Role) read(role appdef.IRole) { r.Type.read(role) }
