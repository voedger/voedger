/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"mypkg1"
	"mypkg2"

	_ "github.com/voedger/voedger/pkg/sys"

	"app/orm"
)

func main() {
	println("app")
	mypkg1.MyFunc1()
	mypkg2.MyFunc2()

	val := orm.Package_mypkg1.WDoc_mytable11.MustGet(0)
	val.Get_field22()
}
