/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	"golang.org/x/exp/slices"
)

func IsSystemRole(role appdef.QName) bool {
	return slices.Contains(SysRoles, role)
}
