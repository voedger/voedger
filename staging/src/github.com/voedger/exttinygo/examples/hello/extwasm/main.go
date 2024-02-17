package main

import "extwasm/schemas"

func MyFunc() {

	// Query air.ProformaPrinted
	{
		v := schemas.Air.ProformaPrinted.MustGetValue(schemas.ID(12))
		println(v.Number())
		println(v.BillID())
	}

	// Query air.PbillDates
	{
		v := schemas.Air.PbillDates.MustGetValue(2019, 12)
		println(v.FirstOffset)
		println(v.LastOffset)
	}

	// Query untill.Articles
	{
		v := schemas.Untill.Articles.MustGetValue(schemas.ID(12))
		println(v.Article_number)
		println(v.Name)
	}
}
