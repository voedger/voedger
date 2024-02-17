package schemas

// nolint
import exttinygo "github.com/voedger/exttinygo"

/*
TABLE ProformaPrinted INHERITS ODoc (

	Number int32 NOT NULL,
	UserID ref(untill.untill_users) NOT NULL,
	Timestamp int64 NOT NULL,
	BillID ref(untill.bill) NOT NULL

);
*/
type Air_ProformaPrinted struct {
	Entity
	// Do we need this for air development?
	Fields struct {
		Number    string
		UserID    string
		Timestamp string
		BillID    string
	}
}

type Air_ProformaPrinted_Value struct{ tv exttinygo.TValue }

func (v *Air_ProformaPrinted_Value) Number() int32 {
	return v.tv.AsInt32("Number")
}
func (v *Air_ProformaPrinted_Value) UserID() int64 {
	return v.tv.AsInt64("UserID")
}
func (v *Air_ProformaPrinted_Value) Timestamp() int64 {
	return v.tv.AsInt64("Timestamp")
}
func (v *Air_ProformaPrinted_Value) BillID() int64 {
	return v.tv.AsInt64("BillID")
}

func (pp *Air_ProformaPrinted) MustGetValue(id ID) Air_ProformaPrinted_Value {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecords, Air.ProformaPrinted.QName)
	return Air_ProformaPrinted_Value{tv: exttinygo.MustGetValue(kb)}
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
type Air_PbillDates struct {
	Entity
	Key struct {
		Year      string
		DayOfYear string
	}
	Fields struct {
		FirstOffset string
		LastOffset  string
	}
}

type Untill_PbillDates_Value struct{ tv exttinygo.TValue }

func (v *Untill_PbillDates_Value) FirstOffset() int32 {
	return v.tv.AsInt32("FirstOffset")
}
func (v *Untill_PbillDates_Value) LastOffset() int32 {
	return v.tv.AsInt32("LastOffset")
}

func (pp *Air_PbillDates) MustGetValue(year int32, dayOfYear int32) Untill_PbillDates_Value {
	kb := exttinygo.KeyBuilder(exttinygo.StorageViewRecords, Air.PbillDates.QName)
	kb.PutInt32("Year", year)
	kb.PutInt32("DayOfYear", dayOfYear)
	return Untill_PbillDates_Value{tv: exttinygo.MustGetValue(kb)}
}
