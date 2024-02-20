/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import exttinygo "github.com/voedger/exttinygo"

const (
	FName_air_ProformaPrinted_Number    = "Number"
	FName_air_ProformaPrinted_UserID    = "UserID"
	FName_air_ProformaPrinted_Timestamp = "Timestamp"
	FName_air_ProformaPrinted_BillID    = "BillID"
	FName_air_PbillDates_Year           = "Year"
	FName_air_PbillDates_DayOfYear      = "DayOfYear"
	FName_air_PbillDates_FirstOffset    = "FirstOffset"
	FName_air_PbillDates_LastOffset     = "LastOffset"
)

var Package_air = struct {
	ODoc_ProformaPrinted ODoc_air_ProformaPrinted
	View_PbillDates      View_air_PbillDates
}{
	ODoc_ProformaPrinted: ODoc_air_ProformaPrinted{
		Type: Type{qname: "air.ProformaPrinted"},
		ValueNames: struct {
			Number    string
			UserID    string
			Timestamp string
			BillID    string
		}{
			Number:    "Number",
			UserID:    "UserID",
			Timestamp: "Timestamp",
			BillID:    "BillID",
		},
	},
	View_PbillDates: View_air_PbillDates{
		Type: Type{qname: "untill.PbillDates"},
		KeyNames: struct {
			Year      string
			DayOfYear string
		}{
			Year:      "Year",
			DayOfYear: "DayOfYear",
		},
		ValueNames: struct {
			FirstOffset string
			LastOffset  string
		}{
			FirstOffset: "FirstOffset",
			LastOffset:  "LastOffset",
		},
	},
}

/*
TABLE ProformaPrinted INHERITS ODoc (

	Number int32 NOT NULL,
	UserID ref(untill.untill_users) NOT NULL,
	Timestamp int64 NOT NULL,
	BillID ref(untill.bill) NOT NULL

);
*/
type ODoc_air_ProformaPrinted struct {
	Type
	ValueNames struct {
		Number    string
		UserID    string
		Timestamp string
		BillID    string
	}
}

/*
VIEW PbillDates (

	Year int32 NOT NULL,
	DayOfYear int32 NOT NULL,
	FirstOffset int64 NOT NULL,
	LastOffset int64 NOT NULL,
	PRIMARY KEY ((Year), DayOfYear)

) AS RESULT OF FillPbillDates;
*/
type View_air_PbillDates struct {
	Type
	KeyNames struct {
		Year      string
		DayOfYear string
	}
	ValueNames struct {
		FirstOffset string
		LastOffset  string
	}
}

func (v *View_air_PbillDates) MustGetValue(year int32, dayOfYear int32) Value_View_untill_PbillDates {
	kb := exttinygo.KeyBuilder(exttinygo.StorageViewRecords, Package_air.View_PbillDates.qname)
	kb.PutInt32(v.KeyNames.Year, year)
	kb.PutInt32(v.KeyNames.DayOfYear, dayOfYear)
	return Value_View_untill_PbillDates{tv: exttinygo.MustGetValue(kb), kb: kb}
}

func (v *View_air_PbillDates) NewIntent(year int32, dayOfYear int32) View_untill_PbillDates_Intent {
	kb := exttinygo.KeyBuilder(exttinygo.StorageViewRecords, Package_air.View_PbillDates.qname)
	kb.PutInt32(v.KeyNames.Year, year)
	kb.PutInt32(v.KeyNames.DayOfYear, dayOfYear)
	return View_untill_PbillDates_Intent{intent: exttinygo.NewValue(kb)}
}

func (v *View_air_PbillDates) QueryValue(year int32, dayOfYear int32) (value Value_View_untill_PbillDates, ok bool) {
	kb := exttinygo.KeyBuilder(exttinygo.StorageViewRecords, Package_air.View_PbillDates.qname)
	kb.PutInt32(v.KeyNames.Year, year)
	kb.PutInt32(v.KeyNames.DayOfYear, dayOfYear)
	ok, tv := exttinygo.QueryValue(kb)
	if !ok {
		return Value_View_untill_PbillDates{}, false
	}
	return Value_View_untill_PbillDates{tv: tv, kb: kb}, true
}

type Value_View_untill_PbillDates struct {
	tv exttinygo.TValue
	kb exttinygo.TKeyBuilder
}

func (v *Value_View_untill_PbillDates) Get_FirstOffset() int32 {
	return v.tv.AsInt32("FirstOffset")
}
func (v *Value_View_untill_PbillDates) Get_LastOffset() int32 {
	return v.tv.AsInt32("LastOffset")
}

func (v *Value_View_untill_PbillDates) NewIntent() View_untill_PbillDates_Intent {
	return View_untill_PbillDates_Intent{intent: exttinygo.NewValue(v.kb)}
}

type View_untill_PbillDates_Intent struct {
	intent exttinygo.TIntent
}

func (i *View_untill_PbillDates_Intent) Set_FirstOffset(value int32) {
	i.intent.PutInt32("FirstOffset", value)
}

func (i *View_untill_PbillDates_Intent) Set_LastOffset(value int32) {
	i.intent.PutInt32("LastOffset", value)
}
