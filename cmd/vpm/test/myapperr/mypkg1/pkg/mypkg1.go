/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package pkg

import (
	reg "github.com/voedger/voedger/pkg/registry"
	_ "github.com/voedger/voedger/pkg/sys"
)

func MyPkg1() {
	println("mypkg1.MyPkg1")
	reg.GetLoginHash("test")
}
