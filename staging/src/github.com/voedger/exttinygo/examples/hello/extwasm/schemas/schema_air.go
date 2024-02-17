package schemas

var Air = struct {
	ProformaPrinted Air_ProformaPrinted
	PbillDates      Air_PbillDates
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
	PbillDates: Air_PbillDates{
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
