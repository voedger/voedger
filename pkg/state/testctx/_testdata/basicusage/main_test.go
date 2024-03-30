/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

const testPkg = "github.com/untillpro/airs-bp3/packages/mypkg"
const testWSID = istructs.WSID(1)

func Test_CalcOrderedItems(t *testing.T) {

	// Construct test context
	ctx := test.NewPackageContext(
		test.ProcKind_Actualizer,
		testPkg,
		test.TestWorkspace{WorkspaceDescriptor: "RestaurantDescriptor", WSID: testWSID})

	// Fill state
	ctx.PutSecret("encryptionKey", []byte("idkfa"))
	ctx.PutView(testWSID, appdef.NewFullQName(testPkg, "OrderedItems"), func(key istructs.IKeyBuilder, value istructs.IValueBuilder) {
		key.PutInt32("Year", 2023)
		key.PutInt32("Month", 1)
		key.PutInt32("Day", 1)
		value.PutInt64("Amount", 100)
	})

	ctx.PutEvent(testWSID, appdef.NewFullQName(testPkg, "Order"), func(arg istructs.IObjectBuilder, _ istructs.ICUD) {
		arg.PutRecordID(appdef.SystemField_ID, 1)
		arg.PutInt32("Year", 2023)
		arg.PutInt32("Month", 1)
		arg.PutInt32("Day", 1)
		items := arg.ChildBuilder("Items")
		items.PutRecordID(appdef.SystemField_ID, 2)
		items.PutInt32("Quantity", 1)
		items.PutInt64("SinglePrice", 100)
	})

	// Call the extension
	CalcOrderedItems()

	// Check the intent
	ctx.RequireIntent(t, state.View, appdef.NewFullQName(testPkg, "OrderedItems"), func(key istructs.IStateKeyBuilder) {
		key.PutInt32("Year", 2023)
		key.PutInt32("Month", 1)
		key.PutInt32("Day", 1)
	}).Equal(func(value istructs.IStateValueBuilder) {
		value.PutInt64("Amount", 200)
	})

}
