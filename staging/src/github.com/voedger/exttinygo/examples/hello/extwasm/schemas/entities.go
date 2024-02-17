package schemas

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
	Fields struct {
		Number    string
		UserID    string
		Timestamp string
		BillID    string
	}
}

type Air_ProformaPrinted_Value struct {
	Number    int32
	UserID    ID
	Timestamp int64
	BillID    ID
}

func (pp *Air_ProformaPrinted) MustGetValue(id ID) Air_ProformaPrinted_Value {
	return Air_ProformaPrinted_Value{}
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
type Untill_PbillDates struct {
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

type Untill_PbillDates_Value struct {
	FirstOffset int32
	LastOffset  int32
}

func (pp *Untill_PbillDates) MustGetValue(year int32, dayOfYear int32) Untill_PbillDates_Value {
	kb := exttinygo.KeyBuilder("table", "qq")
	_ = kb
	// exttinygo.MustGetValue()
	return Untill_PbillDates_Value{}
}
