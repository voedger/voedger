/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package mypkg4

import (
	_ "github.com/voedger/voedger/pkg/sys"

	"mypkg3"
)

func MyFunc4() {
	println("mypkg4.MyFunc4")
	mypkg3.MyFunc3()
}
