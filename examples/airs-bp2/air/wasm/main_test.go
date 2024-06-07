package main

import (
	"path/filepath"
	"testing"

	"github.com/voedger/voedger/examples/airs-bp2/air/wasm/orm"
	"github.com/voedger/voedger/pkg/appdef"
	test "github.com/voedger/voedger/pkg/exttinygo/exttinygotests"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/teststate"
)

func TestPbill(t *testing.T) {
	const (
		pkgPath  = "github.com/voedger/voedger/examples/airs-bp2/air"
		testWSID = istructs.WSID(1)
	)

	testAPI := test.NewTestAPI(
		teststate.ProcKind_CommandProcessor,
		pkgPath,
		teststate.TestWorkspace{WorkspaceDescriptor: "RestaurantDescriptor", WSID: testWSID})

	pkgAlias := filepath.Base(pkgPath)
	nextNumber := int32(5)
	testAPI.PutRecords(testWSID, func(cud istructs.ICUD) {
		fc := cud.Create(appdef.NewQName(pkgAlias, "NextNumbers"))
		fc.PutRecordID(appdef.SystemField_ID, 1)
		fc.PutInt32("NextPBillNumber", nextNumber)
	})

	Pbill()

	// TODO: RequireRecordIntent(t, id, fQName test.IFullQName, map[string]any{'NextPBillNumber': nextNumber+1})
	// type IFullQName interface {
	// PkgPath() string
	// Entity() string
	// }
	testAPI.RequireRecordIntent(
		t,
		state.Record,
		orm.Package_air.WSingleton_NextNumbers,
		map[string]any{state.Field_IsSingleton: true},
		map[string]any{`NextPBillNumber`: nextNumber + 1},
	)

	//testAPI.RequireIntent(t, state.Record, appdef.NewFullQName(pkgAlias, "NextNumbers"), func(key istructs.IStateKeyBuilder) {
	//	key.PutBool(state.Field_IsSingleton, true)
	//}).Assert(func(require *require.Assertions, value istructs.IStateValue) {
	//	require.Equal(nextNumber+1, value.AsInt32("NextPBillNumber"))
	//})
	// TODO: check record in pbill as well
}
