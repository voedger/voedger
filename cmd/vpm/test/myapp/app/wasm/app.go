/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	mypkg1 "mypkg1/pkg"
	mypkg2 "mypkg2/pkg"
)

func main() {
	appFunc()
	mypkg1.MyPkg1()
	mypkg2.MyPkg2()
}

func appFunc() {
	println("app.AppFunc")
}
