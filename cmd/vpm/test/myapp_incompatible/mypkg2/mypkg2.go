/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package mypkg2

import (
	"mypkg1"

	_ "github.com/voedger/voedger/pkg/sys"
)

func MyPkg2() {
	println("mypkg2.MyPkg2")
	mypkg1.MyPkg1()
}
