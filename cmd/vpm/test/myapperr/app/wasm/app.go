/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	mypkg3 "mypkg3/pkg"
	mypkg4 "mypkg4/pkg"

	"github.com/voedger/voedger/pkg/registry"
	_ "github.com/voedger/voedger/pkg/sys"
)

func main() {
	println("mypkg3.MyPkg3")
	registry.GetLoginHash("test")
	mypkg3.MyPkg3()
	mypkg4.MyPkg4()
}
