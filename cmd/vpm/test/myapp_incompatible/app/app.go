/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"mypkg1"
	"mypkg2"

	_ "github.com/voedger/voedger/pkg/sys"
)

func main() {
	println("mypkg3.MyPkg3")
	mypkg1.MyPkg1()
	mypkg2.MyPkg2()
}
