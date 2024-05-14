/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package pkg

import (
	_ "github.com/voedger/voedger/pkg/sys"

	mypkg3 "mypkg3/pkg"
)

func MyFunc4() {
	println("mypkg4.MyFunc4")
	mypkg3.MyFunc3()
}
