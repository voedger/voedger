/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import (
	"time"

	"air/wasm/orm"
)

// Command
//
//export Pbill
func Pbill() {

	// Query untill.pbill from the ArgumentObject
	{
		pbill := orm.Package_air.Command_Pbill.ArgumentObject()

		//Basic types fields
		pbill.Get_id_untill_users()

		// Container
		pbill_items := pbill.Get_pbill_item()
		for i := 0; i < pbill_items.Len(); i++ {
			item := pbill_items.Get(i)
			item.Get_price()
		}
	}

	// Prepare intent for Package_untill.WDoc_bill
	{
		pbill := orm.Package_air.Command_Pbill.ArgumentObject()

		// Basic types fields
		billID := pbill.Get_id_bill()
		intent := orm.Package_untill.WDoc_bill.Update(billID)
		intent.Set_close_datetime(time.Now().UnixMicro())
	}

	// Prepare intent for Package_air.WSingleton_NextNumbers
	{
		var nextNumber int32
		nextNumberValue, nextNumberOk := orm.Package_air.WSingleton_NextNumbers.Get()
		var intent orm.Intent_WSingleton_air_NextNumbers
		if !nextNumberOk {
			nextNumber = 0
			intent = nextNumberValue.Insert()
		} else {
			intent = nextNumberValue.Update() //orm.Package_air.WSingleton_NextNumbers.Update(nextNumberValue)
			nextNumber = nextNumberValue.Get_NextPBillNumber()
		}
		intent.Set_NextPBillNumber(nextNumber + 1)
	}
}

// TODO: add test for FillPbillDates
// FillPbillDates аргументом является событие вызова команды Pbill
// Берет из события дату и время и смещения результирующее (WLog).
// В проекторе аргументом является событие вызова команды Pbill

// Спросить у Мишы как расширение-проектор будет работать с событием, если этот проектор "сидит" на команде
// Надо откуда-то это событие брать
// После из события вытащить смещение, день и год и делаем NewIntent у которого FQName - это вьюшка из задачи
func FillPbillDates() {
	// Query air.PbillDates
	{
		v := orm.Package_air.View_PbillDates.MustGet(2019, 12)
		println(v.Get_FirstOffset())
		println(v.Get_LastOffset())
	}

	// Query untill.Articles
	{
		v := orm.Package_untill.CDoc_articles.MustGet(orm.ID(12))
		println(v.Get_article_number())
		println(v.Get_name())
	}

	// Query air.PbillDates and create intents
	{
		{
			v, ok := orm.Package_air.View_PbillDates.Get(2019, 12)
			if ok {
				intent := v.Insert()
				// `Set` is a must to execute naming conflicts with NewIntent()
				intent.Set_FirstOffset(1)
				intent.Set_LastOffset(2)
			}
		}
		{
			intent := orm.Package_air.View_PbillDates.Insert(2020, 1)
			intent.Set_FirstOffset(20)
			intent.Set_LastOffset(17)
		}
	}
}

func main() {
	Pbill()
}
