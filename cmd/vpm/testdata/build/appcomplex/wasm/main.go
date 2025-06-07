/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	mypkg1 "pkg1/pkg"
	mypkg2 "pkg2/pkg"

	_ "github.com/voedger/voedger/pkg/sys"
)

func main() {
	appFunc()
	mypkg1.MyPkg1()
	mypkg2.MyPkg2()
}

func appFunc() {
	println("app.AppFunc")
}
