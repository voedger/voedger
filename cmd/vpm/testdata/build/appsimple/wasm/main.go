/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import (
	"time"

	_ "github.com/voedger/voedger/pkg/sys"

	"appsimple/wasm/orm"
)

// Command
//
//export Pbill
func Pbill() {

	var refBill orm.Ref

	// Query untill.pbill from the ArgumentObject
	{
		pbill := orm.Package_air.Command_Pbill.ArgumentObject()

		// Basic types fields
		refBill = pbill.Get_id_bill()
		pbill.Get_id_untill_users()

		// Container
		pbill_items := pbill.Get_pbill_item()
		for i := 0; i < pbill_items.Len(); i++ {
			item := pbill_items.Get(i)
			item.Get_rowbeg()
			item.Get_tableno()
		}
	}

	// Prepare intent for Package_untill.WDoc_bill
	{
		intent := orm.Package_untill.WDoc_bill.NewIntent(refBill.ID())
		intent.Set_close_datetime(time.Now().UnixMicro())
	}

	// Prepare intent for Package_air.WSingleton_NextNumbers
	{
		nextNumberValue, nextNumberOk := orm.Package_air.WSingleton_NextNumbers.QueryValue()
		var nextNumber int32
		var intent orm.Intent_WSingleton_air_NextNumbers
		if !nextNumberOk {
			nextNumber = 1
			intent = orm.Package_air.WSingleton_NextNumbers.NewIntent()
		} else {
			nextNumber = nextNumberValue.Get_NextPBillNumber()
			intent = nextNumberValue.NewIntent()
		}
		intent.Set_NextPBillNumber(nextNumber + 1)
	}
}

// nolint revive
func MyProjector() {

	// Query air.PbillDates
	{
		v := orm.Package_air.View_PbillDates.MustGetValue(2019, 12)
		println(v.Get_FirstOffset())
		println(v.Get_LastOffset())
	}

	// Query untill.Articles
	{
		v := orm.Package_untill.CDoc_articles.MustGetValue(orm.ID(12))
		println(v.Get_article_number())
		println(v.Name())
	}

	// Query air.PbillDates and create intents
	{
		{
			v, ok := orm.Package_air.View_PbillDates.QueryValue(2019, 12)
			if ok {
				intent := v.NewIntent()
				// `Set` is a must to execute naming conflicts with NewIntent()
				intent.Set_FirstOffset(1)
				intent.Set_LastOffset(2)
			}
		}
		{
			intent := orm.Package_air.View_PbillDates.NewIntent(2020, 1)
			intent.Set_FirstOffset(20)
			intent.Set_LastOffset(17)
		}
	}
}

func main() {
	Pbill()
}
