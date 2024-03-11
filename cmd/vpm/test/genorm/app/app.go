/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package app

import (
	"mypkg1"
	"mypkg2"

	_ "github.com/voedger/voedger/pkg/sys"
)

func main() {
	println("app")
	mypkg1.MyFunc1()
	mypkg2.MyFunc2()
}
