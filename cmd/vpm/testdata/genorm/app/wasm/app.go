/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	mypkg1 "mypkg1/pkg"
	mypkg2 "mypkg2/pkg"

	_ "github.com/voedger/voedger/pkg/sys"
)

func main() {
	AppFunc()
}

func AppFunc() {
	println("app")
	mypkg1.MyFunc1()
	mypkg2.MyFunc2()
}
