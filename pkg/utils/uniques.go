/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func UniqueQName(docQName appdef.QName, name string) appdef.QName {
	return appdef.NewQName(docQName.Pkg(), fmt.Sprintf("%sUniques%s", docQName.Entity(), name)) // TODO: add $ ?
}
