package schemas

type Field struct {
	Name string
	Type string
}

var Air = struct {
	ProformaPrinted Air_ProformaPrinted
}{
	ProformaPrinted: Air_ProformaPrinted{
		Entity: Entity{QName: "air.ProformaPrinted"},
		Fields: struct {
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
}

var Untill = struct {
	PbillDates Untill_PbillDates
}{
	PbillDates: Untill_PbillDates{
		Entity: Entity{QName: "untill.PbillDates"},
		Key: struct {
			Year      string
			DayOfYear string
		}{
			Year:      "Year",
			DayOfYear: "DayOfYear",
		},
		Fields: struct {
			FirstOffset string
			LastOffset  string
		}{
			FirstOffset: "FirstOffset",
			LastOffset:  "LastOffset",
		},
	},
}
