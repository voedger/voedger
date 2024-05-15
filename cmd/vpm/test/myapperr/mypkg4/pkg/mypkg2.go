/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package pkg

import (
	mypkg3 "mypkg3/pkg"

	"github.com/voedger/voedger/pkg/registry"
	_ "github.com/voedger/voedger/pkg/sys"
)

func MyPkg4() {
	println("mypkg2.MyPkg2")
	mypkg3.MyPkg3()
	registry.GetLoginHash("test")
}
