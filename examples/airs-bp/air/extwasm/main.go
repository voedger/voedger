/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import "extwasm/orm"

// Command
func UpdateArticle() {
	// Query untill.Articles
	{
		// v := orm.Package_untill.Articles.MustGetValue(orm.ID(12))
		// intent := v.NewIntent()

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
