/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/istructs"
	"golang.org/x/exp/slices"
)

func IsSystemRole(role istructs.QName) bool {
	return slices.Contains(SysRoles, role)
}
