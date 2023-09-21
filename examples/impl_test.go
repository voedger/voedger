/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package examples

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iextengine"
	wasm "github.com/voedger/voedger/pkg/iextenginewazero"
)

const (
	extUpdateMockTableStatus = "updateMockTableStatus"
	extUpdateTableStatus     = "updateTableStatus"
)

const (
	modeOrder = iota
	modeBill
)

//go:embed sys/*.sql
var sfs embed.FS

//go:embed vrestaurant/*.sql
var fsvRestaurant embed.FS

var limits = iextengine.ExtensionLimits{
	ExecutionInterval: 100 * time.Second,
}

func Test_BasicUsageMockWasmExt(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()

	moduleURL := testModuleURL("./vrestaurant/extwasm/ext.wasm")
	extEngine, err := wasm.ExtEngineWazeroFactory(ctx, moduleURL, []string{extUpdateMockTableStatus}, iextengine.ExtEngineConfig{})
	require.Nil(err)

	extEngine.SetLimits(limits)

	//
	// Invoke Order
	//

	var order = &mockIo{}
	mockmode = modeOrder

	err = extEngine.Invoke(ctx, extUpdateMockTableStatus, order)
	require.NoError(err)
	require.Equal(int(1), len(order.intents))
	v := order.intents[0].value.(*mockValueBuilder)
	require.Equal(int32(1560), v.items["Amount"])
	require.Equal(int32(1), v.items["Status"])

	//
	// Invoke Payment
	//
	var bill = &mockIo{}
	mockmode = modeBill
	err = extEngine.Invoke(ctx, extUpdateMockTableStatus, bill)
	require.NoError(err)

	require.Equal(1, len(bill.intents))
	b := bill.intents[0].value.(*mockValueBuilder)
	require.Equal(int32(0), b.items["Amount"])
	require.Equal(int32(0), b.items["Status"])
}
/*
func Test_BasicUsageWasmExt(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()

	moduleURL := testModuleURL("./vrestaurant/extwasm/ext.wasm")
	extEngine, err := wasm.ExtEngineWazeroFactory(ctx, moduleURL, []string{extUpdateTableStatus}, iextengine.ExtEngineConfig{})
	require.NoError(err)
	extEngine.SetLimits(limits)

	// Init BO for Ordering
	InitTestBO()

	// Create Order in storage
	CreateTestOrder()
	//
	// Invoke Order
	//
	var order = &mockIo{}
	require.NoError(extensions["updateTableStatus"].Invoke(order))
	require.Equal(1, len(order.intents))
	v := order.intents[0].value.(*mockValueBuilder)
	require.Equal(1, v.items["Status"])

	// Init BO for Payment
	CreateTestBill()

	//
	// Invoke Payment
	//
	var bill = &mockIo{}
	require.NoError(extensions["updateTableStatus"].Invoke(bill))
	require.Equal(1, len(order.intents))
	v = order.intents[0].value.(*mockValueBuilder)
	require.Equal(0, v.items["Status"])

}
*/
/*
	func getSysPackageAST() *parser.PackageSchemaAST {
		pkgSys, err := parser.ParsePackageDir(appdef.SysPackage, sfs, "sys")
		if err != nil {
			panic(err)
		}
		return pkgSys
	}

func Test_VRestaurantBasic(t *testing.T) {

		require := require.New(t)

		vRestaurantPkgAST, err := parser.ParsePackageDir("github.com/examples/vrestaurant", fsvRestaurant, "vrestaurant")
		require.NoError(err)

		packages, err := parser.MergePackageSchemas([]*parser.PackageSchemaAST{
			getSysPackageAST(),
			vRestaurantPkgAST,
		})
		require.NoError(err)

		builder := appdef.New()
		err = parser.BuildAppDefs(packages, builder)
		require.NoError(err)

		// table
		cdoc := builder.Def(appdef.NewQName("vrestaurant", "TablePlan"))
		require.NotNil(cdoc)
		require.Equal(appdef.DefKind_CDoc, cdoc.Kind())
		require.Equal(appdef.DataKind_RecordID, cdoc.(appdef.IFields).Field("Picture").DataKind())

		cdoc = builder.Def(appdef.NewQName("vrestaurant", "Client"))
		require.NotNil(cdoc)

		cdoc = builder.Def(appdef.NewQName("vrestaurant", "POSUser"))
		require.NotNil(cdoc)

		cdoc = builder.Def(appdef.NewQName("vrestaurant", "Department"))
		require.NotNil(cdoc)

		cdoc = builder.Def(appdef.NewQName("vrestaurant", "Article"))
		require.NotNil(cdoc)

		// child table
		crec := builder.Def(appdef.NewQName("vrestaurant", "TableItem"))
		require.NotNil(crec)
		require.Equal(appdef.DefKind_CRecord, crec.Kind())
		require.Equal(appdef.DataKind_int32, crec.(appdef.IFields).Field("Tableno").DataKind())

		// view
		view := builder.View(appdef.NewQName("vrestaurant", "SalesPerDay"))
		require.NotNil(view)
		require.Equal(appdef.DefKind_ViewRecord, view.Kind())
	}
*/
