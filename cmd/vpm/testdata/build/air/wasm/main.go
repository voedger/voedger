/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import (
	"time"

	ext "github.com/voedger/voedger/pkg/exttinygo"

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

		intent.Set_close_year(int32(time.Now().UTC().Year()))
	}

	// Prepare intent for Package_air.WSingleton_NextNumbers
	{
		var nextNumber int32
		var intent orm.Intent_WSingleton_air_NextNumbers

		nextNumberValue, nextNumberOk := orm.Package_air.WSingleton_NextNumbers.Get()
		if !nextNumberOk {
			nextNumber = 0
			intent = nextNumberValue.Insert()
		} else {
			intent = nextNumberValue.Update()
			nextNumber = nextNumberValue.Get_NextPBillNumber()
		}

		intent.Set_NextPBillNumber(nextNumber + 1)
	}
}

func ProjectorFillPbillDates() {
	projector := orm.Package_air.Projector_FillPbillDates()
	if arg, ok := projector.Arg_untill_pbill(); ok {
		offs := projector.Event().WLogOffset
		// get pbill datetime
		pbillDatetime := time.UnixMicro(arg.Get_pdatetime())
		// extract year and day of year from pbill datetime
		year := pbillDatetime.Year()
		dayOfYear := pbillDatetime.Day()

		var intent orm.Intent_View_air_PbillDates
		val, ok := orm.Package_air.View_PbillDates.Get(int32(year), int32(dayOfYear))
		if !ok {
			intent = val.Insert()
			intent.Set_FirstOffset(offs)
		} else {
			intent = val.Update()
		}

		intent.Set_LastOffset(offs)

	}
}

func ProjectorNewVideoRecord() {
	projector := orm.Package_air.Projector_ProjectorNewVideoRecord()

	years := make(map[int]map[int]map[int]int64)
	for cud := range projector.CUDs_air_VideoRecords() {
		date := time.UnixMicro(cud.Get_Date())
		year := date.Year()
		month := int(date.Month())
		day := date.Day()

		months, ok := years[year]
		if !ok {
			months = make(map[int]map[int]int64)
			years[year] = months
		}

		days, ok := months[month]
		if !ok {
			days = make(map[int]int64)
			months[month] = days
		}

		days[day] += cud.Get_Length()
	}

	for year, months := range years {
		for month, days := range months {
			for day, length := range days {
				var intent orm.Intent_View_air_VideoRecordArchive
				val, ok := orm.Package_air.View_VideoRecordArchive.Get(int32(year), int32(month), int32(day))
				if !ok {
					intent = val.Insert()
				} else {
					intent = val.Update()
				}
				intent.Set_TotalLength(length)
			}
		}
	}
}

func ProjectorApplySalesMetrics() {
	projector := orm.Package_air.Projector_ApplySalesMetrics()
	if arg, ok := projector.Cmd_Pbill().Arg(); ok {
		offs := projector.Cmd_Pbill().Event().WLogOffset

		pbillDatetime := time.UnixMicro(arg.Get_pdatetime())
		// extract year and day of year from pbill datetime
		year := pbillDatetime.Year()
		dayOfYear := pbillDatetime.Day()

		var intent orm.Intent_View_air_PbillDates
		val, ok := orm.Package_air.View_PbillDates.Get(int32(year), int32(dayOfYear))
		if !ok {
			intent = val.Insert()
			intent.Set_FirstOffset(offs)
		} else {
			intent = val.Update()
		}

		intent.Set_LastOffset(offs)
	}
}

func ProjectorODoc() {
	projector := orm.Package_air.Projector_ProjectorODoc()
	if odoc, ok := projector.ODoc(); ok {
		if !odoc.Is(orm.Package_air.ODoc_ProformaPrinted) {
			return
		}

		offs := odoc.Event().WLogOffset

		date := time.UnixMicro(odoc.AsInt64("Timestamp"))
		// extract year and day of year from pbill datetime
		year := date.Year()
		dayOfYear := date.Day()

		var intent orm.Intent_View_air_ProformaPrintedDocs

		val, ok := orm.Package_air.View_ProformaPrintedDocs.Get(int32(year), int32(dayOfYear))
		if !ok {
			intent = val.Insert()
			intent.Set_FirstOffset(offs)
		} else {
			intent = val.Update()
		}

		intent.Set_LastOffset(offs)

	}
}

func FillPbillDates() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	offs := event.AsInt64("WLogOffset")
	arg := event.AsValue("ArgumentObject")
	// get pbill datetime
	pbillDatetime := time.UnixMicro(arg.AsInt64("pdatetime"))
	// extract year and day of year from pbill datetime
	year := pbillDatetime.Year()
	dayOfYear := pbillDatetime.Day()

	var intent orm.Intent_View_air_PbillDates
	val, ok := orm.Package_air.View_PbillDates.Get(int32(year), int32(dayOfYear))
	if !ok {
		intent = val.Insert()
		intent.Set_FirstOffset(offs)
	} else {
		intent = val.Update()
	}

	intent.Set_LastOffset(offs)
}

func main() {
	Pbill()
}
