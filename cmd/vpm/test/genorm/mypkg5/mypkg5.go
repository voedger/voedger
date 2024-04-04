/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package mypkg5

import (
	"mypkg1"

	_ "github.com/voedger/voedger/pkg/sys"
)

func MyFunc5() {
	println("mypkg5.MyFunc5")
	mypkg1.MyFunc1()
}
