/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginetestctx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	test "github.com/voedger/voedger/pkg/iextengine/testctx"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

const testPkg = "github.com/untillpro/airs-bp3/packages/mypkg"
const testWSID = istructs.WSID(1)

func Test_BasicUsage_Command(t *testing.T) {

	require := require.New(t)
	ctx := test.NewPackageContext(
		"./_testdata/basicusage",
		appdef.ExtensionEngineKind_WASM,
		test.ProcKind_CommandProcessor,
		testPkg)

	defer ctx.Close()

	ctx.PutSecret("encryptionKey", []byte("idkfa"))

	ctx.PutEvent(testWSID, appdef.NewFullQName(testPkg, "Order"), func(arg istructs.IObjectBuilder, _ istructs.ICUD) {
		arg.PutRecordID(appdef.SystemField_ID, 1)
		arg.PutInt32("Year", 2023)
		arg.PutInt32("Month", 1)
		arg.PutInt32("Day", 1)
		items := arg.ChildBuilder("Items")
		items.PutRecordID(appdef.SystemField_ID, 2)
		items.PutInt32("Quantity", -1)
		items.PutInt64("SinglePrice", 100)
	})

	require.ErrorContains(ctx.Invoke(appdef.NewFullQName(testPkg, "NewOrder")), "negative order amount")
}

func Test_BasicUsage_Projector(t *testing.T) {

	require := require.New(t)
	ctx := test.NewPackageContext(
		"./_testdata/basicusage",
		appdef.ExtensionEngineKind_WASM,
		test.ProcKind_Actualizer,
		testPkg,
		test.TestWorkspace{WorkspaceDescriptor: "RestaurantDescriptor", WSID: testWSID})

	defer ctx.Close()

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

	require.NoError(ctx.Invoke(appdef.NewFullQName(testPkg, "CalcOrderedItems")))

	require.True(ctx.HasIntent(state.View, appdef.NewFullQName(testPkg, "OrderedItems"), func(key istructs.IStateKeyBuilder, value istructs.IStateValueBuilder) {
		key.PutInt32("Year", 2023)
		key.PutInt32("Month", 1)
		key.PutInt32("Day", 1)
		value.PutInt64("Amount", 200)
	}))

}
