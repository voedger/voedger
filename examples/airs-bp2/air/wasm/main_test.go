package main

import (
	"air/wasm/orm"
	"testing"

	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"
)

func TestPbill(t *testing.T) {
	// TODO: if intent was created but we didn't expect it, it should be an error

	// test singletone insert
	{
		require := test.NewCommandRunner(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			Record( // не превращает raw id-шники в real id-шники
				orm.Package_untill.WDoc_bill,
				1,
				`id_untill_users`, 100001,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100000,
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
			Run()

		// check that the next number is inserted
		nextNumber := int32(1)
		require.SingletonInsert(
			orm.Package_air.WSingleton_NextNumbers,
			`NextPBillNumber`, nextNumber,
		)
	}

	{
		nextNumber := int32(5)

		require := test.NewCommandRunner(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			Record( // не превращает raw id-шники в real id-шники
				orm.Package_air.WSingleton_NextNumbers,
				1,
				`NextPBillNumber`, nextNumber,
			).
			Record( // не превращает raw id-шники в real id-шники
				orm.Package_untill.WDoc_bill,
				1,
				`id_untill_users`, 100001,
				`tableno`, 1,
			).
			ArgumentObject(
				2,
				`id_bill`, 100000,
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
			Run() // Run() набивает тест стейт, запускает его и возвращает ICommandRequire

		// check that the next number is inserted
		require.SingletonUpdate(
			orm.Package_air.WSingleton_NextNumbers,
			`NextPBillNumber`, nextNumber+1,
		)
	}
}
