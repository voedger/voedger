/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package main

import (
	"air/wasm/orm"
	"testing"
	"time"

	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"
)

func TestPbill(t *testing.T) {
	t.Parallel()

	currentYear := time.Now().UTC().Year()

	t.Run("Singleton NextPBillNumber: insert", func(t *testing.T) {

		test.NewCommandRunner(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			Record(
				orm.Package_untill.WDoc_bill,
				100002,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
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
			RequireSingletonInsert(
				orm.Package_air.WSingleton_NextNumbers,
				`NextPBillNumber`, 1,
			).
			RequireRecordUpdate(
				orm.Package_untill.WDoc_bill,
				100002,
				`close_year`, currentYear,
			).
			Run()
	})

	t.Run("Singleton NextPBillNumber: update", func(t *testing.T) {
		nextNumber := 5

		test.NewCommandRunner(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			Record(
				orm.Package_air.WSingleton_NextNumbers,
				1,
				`NextPBillNumber`, nextNumber,
			).
			Record(
				orm.Package_untill.WDoc_bill,
				100002,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100002,
				`id_untill_users`, 100001,
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
			RequireSingletonUpdate(
				orm.Package_air.WSingleton_NextNumbers,
				`NextPBillNumber`, nextNumber+1,
			).
			RequireRecordUpdate(
				orm.Package_untill.WDoc_bill,
				100002,
				`close_year`, currentYear,
			).
			Run()
	})
}
