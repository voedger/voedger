/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package main

import (
	"testing"
	"time"

	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"

	"air/wasm/orm"
)

func TestProjectorFillPbillDates(t *testing.T) {
	date := time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)

	t.Run("View View_PbillDates:insert", func(t *testing.T) {
		test.NewProjectorTest(
			t,
			ProjectorFillPbillDates,
		).
			EventQName(orm.Package_air.Command_Pbill).
			EventWLogOffset(123).
			EventArgumentObject(
				orm.Package_untill.ODoc_pbill,
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
				`pdatetime`, date.UnixMicro(),
			).
			EventArgumentObjectRow(`pbill_item`,
				3,
				`sys.ParentID`, 2,
				`id_pbill`, 100000,
				`id_untill_users`, 100001,
				`tableno`, 123,
				`quantity`, 2,
				`price`, 50_00,
			).
			IntentViewInsert(
				orm.Package_air.View_PbillDates,
				`Year`, 2023,
				`DayOfYear`, 9,
				`FirstOffset`, 123,
				`LastOffset`, 123,
			).
			Run()
	})

	t.Run("View View_PbillDates:update", func(t *testing.T) {
		test.NewProjectorTest(
			t,
			ProjectorFillPbillDates,
		).
			EventQName(orm.Package_air.Command_Pbill).
			EventWLogOffset(123).
			StateView(
				orm.Package_air.View_PbillDates,
				110012,
				`Year`, 2023,
				`DayOfYear`, 9,
				`FirstOffset`, 10,
				`LastOffset`, 10,
			).
			EventArgumentObject(
				orm.Package_untill.ODoc_pbill,
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
				`pdatetime`, date.UnixMicro(),
			).
			EventArgumentObjectRow(`pbill_item`,
				3,
				`sys.ParentID`, 2,
				`id_pbill`, 100000,
				`id_untill_users`, 100001,
				`tableno`, 123,
				`quantity`, 2,
				`price`, 50_00,
			).
			IntentViewUpdate(
				orm.Package_air.View_PbillDates,
				110012,
				`Year`, 2023,
				`DayOfYear`, 9,
				`FirstOffset`, 10,
				`LastOffset`, 123,
			).
			Run()
	})
}

func TestPbill(t *testing.T) {
	date := time.Now()

	t.Run("Singleton.NextPBillNumber:insert", func(t *testing.T) {
		test.NewCommandTest(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			StateRecord(
				orm.Package_untill.WDoc_bill,
				100002,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
				`pdatetime`, date.UnixMicro(),
			).
			ArgumentObjectRow(`pbill_item`,
				3,
				`sys.ParentID`, 2,
				`id_pbill`, 100000,
				`id_untill_users`, 100001,
				`tableno`, 123,
				`quantity`, 2,
				`price`, 50_00,
			).
			IntentSingletonInsert(
				orm.Package_air.WSingleton_NextNumbers,
				`NextPBillNumber`, 1,
			).
			IntentRecordUpdate(
				orm.Package_untill.WDoc_bill,
				100002,
				`close_year`, date.Year(),
			).
			Run()
	})

	t.Run("Singleton.NextPBillNumber:update", func(t *testing.T) {
		nextNumber := 5

		test.NewCommandTest(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			StateSingletonRecord(
				orm.Package_air.WSingleton_NextNumbers,
				`NextPBillNumber`, nextNumber,
			).
			StateRecord(
				orm.Package_untill.WDoc_bill,
				100002,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
				`pdatetime`, date.UnixMicro(),
			).
			ArgumentObjectRow(`pbill_item`,
				3,
				`sys.ParentID`, 2,
				`id_pbill`, 100000,
				`id_untill_users`, 100001,
				`tableno`, 123,
				`quantity`, 2,
				`price`, 50_00,
			).
			IntentSingletonUpdate(
				orm.Package_air.WSingleton_NextNumbers,
				`NextPBillNumber`, nextNumber+1,
			).
			IntentRecordUpdate(
				orm.Package_untill.WDoc_bill,
				100002,
				`close_year`, date.Year(),
			).
			Run()
	})
}

func TestProjectorODoc(t *testing.T) {
	date := time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)
	t.Run("ProjectorODoc", func(t *testing.T) {
		test.NewProjectorTest(
			t,
			ProjectorODoc,
		).
			EventWLogOffset(123).
			EventQName(orm.Package_air.Command_CmdForProformaPrinted).
			EventArgumentObject(
				orm.Package_air.ODoc_ProformaPrinted,
				1,
				`Number`, 100002,
				`UserID`, 100001,
				`Timestamp`, date.UnixMicro(),
			).
			IntentViewInsert(
				orm.Package_air.View_ProformaPrintedDocs,
				`Year`, 2023,
				`DayOfYear`, 9,
				`FirstOffset`, 123,
				`LastOffset`, 123,
			).
			Run()
	})
}

func TestProjectorApplySalesMetrics(t *testing.T) {
	date := time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)

	t.Run("ProjectorApplySalesMetrics", func(t *testing.T) {
		test.NewProjectorTest(
			t,
			ProjectorApplySalesMetrics,
		).
			EventWLogOffset(123).
			EventQName(orm.Package_air.Command_Pbill).
			EventArgumentObject(
				orm.Package_untill.ODoc_pbill,
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
				`pdatetime`, date.UnixMicro(),
			).
			EventArgumentObjectRow(`pbill_item`,
				3,
				`sys.ParentID`, 2,
				`id_pbill`, 100000,
				`id_untill_users`, 100001,
				`tableno`, 123,
				`quantity`, 2,
				`price`, 50_00,
			).
			IntentViewInsert(
				orm.Package_air.View_PbillDates,
				`Year`, 2023,
				`DayOfYear`, 9,
				`FirstOffset`, 123,
				`LastOffset`, 123,
			).
			Run()
	})
}

func TestProjectorNewVideoRecord(t *testing.T) {
	t.Run("ProjectorNewVideoRecord", func(t *testing.T) {
		test.NewProjectorTest(
			t,
			ProjectorNewVideoRecord,
		).
			EventWLogOffset(123).
			EventQName(orm.Package_sys.Command_CUD).
			EventCUD(
				orm.Package_air.CDoc_VideoRecords,
				1,
				`Name`, `record 1`,
				`Length`, 125,
				`Date`, time.Date(
					2023,
					1,
					9,
					0,
					0,
					0,
					0,
					time.UTC,
				).UnixMicro(),
			).
			EventCUD(
				orm.Package_air.CDoc_VideoRecords,
				2,
				`Name`, `record 2`,
				`Length`, 15,
				`Date`, time.Date(
					2023,
					1,
					9,
					0,
					0,
					0,
					0,
					time.UTC,
				).UnixMicro(),
			).
			IntentViewInsert(
				orm.Package_air.View_VideoRecordArchive,
				`Year`, 2023,
				`Month`, 1,
				`Day`, 9,
				`TotalLength`, 140,
			).
			Run()
	})
}
