/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/schemas"
	"golang.org/x/exp/slices"
)

func IsSystemRole(role schemas.QName) bool {
	return slices.Contains(SysRoles, role)
}
