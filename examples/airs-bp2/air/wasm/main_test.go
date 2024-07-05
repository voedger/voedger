package main

import (
	"testing"
	"time"

	"github.com/voedger/voedger/examples/airs-bp2/air/wasm/orm"
	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"
)

func TestPbill(t *testing.T) {
	// test singletone insert
	{
		require := test.NewCommandRunner(
			t,
			orm.Package_air.Command_Pbill,
			Pbill,
		).
			Record( // не превращает raw id-шники в real id-шники
				orm.Package_air.WSingleton_NextNumbers,
				1,
				`NextPBillNumber`, int32(6),
			).
			Record( // не превращает raw id-шники в real id-шники
				orm.Package_untill.WDoc_bill,
				100000,
				`id_untill_users`, 100001,
				`tableno`, 1,
				`table_part`, `a`,
				`proforma`, 1,
				`working_day`, `monday`,
			).
			ArgumentObject(
				1,
				`tips`, 20_00,
				`id_bill`, 100000,
				`id_untill_users`, 100001,
				`pdatetime`, time.Now().UnixMicro(),
				`working_day`, `monday`,
			).
			ArgumentObjectRow(`pbill_item`,
				1,
				`sys.ParentID`, 1,
				`id_untill_users`, 100001,
				`rowbeg`, 1,
				`tableno`, 123,
				`rowbeg`, 1,
				`kind`, 1,
				`quantity`, 2,
				`price`, 50_00,
			).
			Run() // Run() набивает тест стейт, запускает его и возвращает ICommandRequire

		// ICommandRequire ...

		//testCtx := test.NewProjectContextBuilder(
		//	t,
		//	orm.Package_air.Command_Pbill,
		//	Pbill,
		//).CUD().ArgumentObject().ArgumentObjectRow(`pbill_item`, `id_bill`, int64(1)).Build()

		////
		//testCtx.PutArgument(
		//	`id_bill`, int64(1),
		//	`id_bill`, int64(1),
		//	`id_bill`, int64(1),
		//).AddRow(`pbill_item`, `id_bill`, int64(1))

		// test context assessment
		// implicitly calls PBill
		//require := testCtx.NewRequire()

		// check that the next number is inserted
		nextNumber := int32(1)
		require.SingletonInsert(
			orm.Package_air.WSingleton_NextNumbers,
			`NextPBillNumber`, nextNumber,
		)
	}

	// TODO: if intent was created but we didn't expect it, it should be an error
	// (DO NOT USE RequireNoIntent())

	////test singletone update
	//{
	//	testCtx := test.NewContext(
	//		t,
	//		orm.Package_air.Command_Pbill,
	//		Pbill,
	//	)
	//
	//	// test context assessment
	//	nextNumber := int32(5)
	//	testCtx.PutRecord(
	//		orm.Package_air.WSingleton_NextNumbers,
	//		`NextPBillNumber`, nextNumber,
	//	)
	//
	//	require := testCtx.NewRequire()
	//
	//	// check that the next number is updated
	//	require.SingletonUpdate(
	//		orm.Package_air.WSingleton_NextNumbers,
	//		`NextPBillNumber`, nextNumber+1,
	//	)
	//}
}
