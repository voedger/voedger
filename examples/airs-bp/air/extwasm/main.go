/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import "extwasm/orm"

// Command
func Pbill() {

	// Query untill.pbill from the ArgumentObject
	{
		pbill := orm.Package_air.Command_Pbill.ArgumentObject()

		// Basic types fields
		pbill.Get_id_bill()
		pbill.Get_id_untill_users()

		// Container
		pbill_items := pbill.Get_pbill_item()
		for i := 0; i < pbill_items.Len(); i++ {
			item := pbill_items.Get(i)
			item.Get_rowbeg()
			item.Get_tableno()
		}
	}
}

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
