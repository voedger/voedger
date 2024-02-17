package main

import "extwasm/schemas"

func MyFunc() {

	// Query Untill_ProformaPrinted
	{
		v := schemas.Air.ProformaPrinted.MustGetValue(schemas.ID(12))
		println(v.Number())
		println(v.BillID())
	}

	// Query PbillDates
	{
		v := schemas.Untill.PbillDates.MustGetValue(2019, 12)
		println(v.FirstOffset)
		println(v.LastOffset)
	}
}
