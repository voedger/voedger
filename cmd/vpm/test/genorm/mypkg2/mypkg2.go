/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package mypkg2

import (
	"mypkg1"

	_ "github.com/voedger/voedger/pkg/sys"
)

func MyFunc2() {
	println("mypkg2.MyFunc2")
	mypkg1.MyFunc1()
}
