/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package pkg

import (
	mypkg2 "pkg2/pkg"

	_ "github.com/voedger/voedger/pkg/sys"
)

func MyPkg1() {
	println("mypkg2.MyPkg2")
	mypkg2.MyPkg2()
}
