/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package pkg

import (
	mypkg1 "mypkg1/pkg"

	_ "github.com/voedger/voedger/pkg/sys"
)

func MyFunc2() {
	println("mypkg2.MyFunc2")
	mypkg1.MyFunc1()
}
