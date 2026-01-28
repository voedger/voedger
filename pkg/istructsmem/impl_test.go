/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/isequencer"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
)

/* Test scenario. */
// Someone who introduced himself as «Carlson 哇" 呀呀» was making a purchase at Supermarket # 1234.
// While he was uploading purchases at self-checkout # 762, the store's automated tools took a picture of him,
// calculated his height (1.75 m) and approximate age (33 years).
// All this data, along with the data on the contents of his basket (cookies and jam), came to us in testDataType. Our task:
// - form new sync event width command «test.sales»
// - write it to PLog at offset 10001 and in WLog at offset 1001
// - write the characteristics of this customer to the «test.photos» table into a new record
// - read the data from the PLog and WLog journals, and from the «test.photo» table and from the «main.photoView» view
//

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	// create app configuration
	appConfigs := func() AppConfigsType {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

		saleParamsName := appdef.NewQName("test", "SaleParams")
		saleSecureParamsName := appdef.NewQName("test", "saleSecureArgs")

		saleParamsDoc := wsb.AddODoc(saleParamsName)
		saleParamsDoc.
			AddField("Buyer", appdef.DataKind_string, true).
			AddField("Age", appdef.DataKind_int8, false).
			AddField("Height", appdef.DataKind_float32, false).
			AddField("isHuman", appdef.DataKind_bool, false).
			AddField("Photo", appdef.DataKind_bytes, false)
		saleParamsDoc.
			AddContainer("Basket", appdef.NewQName("test", "Basket"), 1, 1)

		basketRec := wsb.AddORecord(appdef.NewQName("test", "Basket"))
		basketRec.AddContainer("Good", appdef.NewQName("test", "Good"), 0, appdef.Occurs_Unbounded)

		goodRec := wsb.AddORecord(appdef.NewQName("test", "Good"))
		goodRec.
			AddField("Name", appdef.DataKind_string, true).
			AddField("Code", appdef.DataKind_int64, true).
			AddField("Weight", appdef.DataKind_float64, false)

		saleSecureParamsObj := wsb.AddObject(saleSecureParamsName)
		saleSecureParamsObj.
			AddField("password", appdef.DataKind_string, true)

		photosDoc := wsb.AddCDoc(appdef.NewQName("test", "photos"))
		photosDoc.
			AddField("Buyer", appdef.DataKind_string, true).
			AddField("Age", appdef.DataKind_int8, false).
			AddField("Height", appdef.DataKind_float32, false).
			AddField("isHuman", appdef.DataKind_bool, false).
			AddField("Photo", appdef.DataKind_bytes, false)

		qNameCmdTestSale := appdef.NewQName("test", "Sale")
		wsb.AddCommand(qNameCmdTestSale).
			SetUnloggedParam(saleSecureParamsName).
			SetParam(saleParamsName)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		cfg.Resources.Add(NewCommandFunction(qNameCmdTestSale, NullCommandExec))

		return cfgs
	}()

	// gets AppStructProvider and AppStructs
	provider := Provide(appConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)

	app, err := provider.BuiltIn(appName)
	require.NoError(err)

	// Build raw event demo
	// 1. gets event builder
	bld := app.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				HandlingPartition: 55,
				PLogOffset:        10000,
				Workspace:         1234,
				WLogOffset:        1000,
				QName:             appdef.NewQName("test", "Sale"),
				RegisteredAt:      100500,
			},
			Device:   762,
			SyncedAt: 1005001,
		})

	// 2. make command params object
	cmd := bld.ArgumentObjectBuilder()

	cmd.PutRecordID(appdef.SystemField_ID, 1)
	cmd.PutString("Buyer", "Carlson 哇\"呀呀") // to test unicode issues
	cmd.PutInt8("Age", 33)
	cmd.PutFloat32("Height", 1.75)
	cmd.PutBytes("Photo", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 4, 4, 3, 2, 1, 0})

	basket := cmd.ChildBuilder("Basket")
	basket.PutRecordID(appdef.SystemField_ID, 2)

	good := basket.ChildBuilder("Good")
	good.PutRecordID(appdef.SystemField_ID, 3)
	good.PutString("Name", "Biscuits")
	good.PutInt64("Code", 7070)
	good.PutFloat64("Weight", 1.1)

	good = basket.ChildBuilder("Good")
	good.PutRecordID(appdef.SystemField_ID, 4)
	good.PutString("Name", "Jam")
	good.PutInt64("Code", 8080)
	good.PutFloat64("Weight", 2.02)

	security := bld.ArgumentUnloggedObjectBuilder()
	security.PutString("password", "12345")

	// 3. make result cud
	cud := bld.CUDBuilder()
	rec := cud.Create(appdef.NewQName("test", "photos"))
	rec.PutRecordID(appdef.SystemField_ID, 11)
	rec.PutString("Buyer", "Carlson 哇\"呀呀")
	rec.PutInt8("Age", 33)
	rec.PutFloat32("Height", 1.75)
	rec.PutBytes("Photo", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 4, 4, 3, 2, 1, 0})

	// 4. get raw event
	rawEvent, buildErr := bld.BuildRawEvent()
	require.NoError(buildErr)

	// Save raw event to PLog & WLog and save CUD demo
	// 5. save to PLog
	pLogEvent, saveErr := app.Events().PutPlog(rawEvent, buildErr, NewIDGenerator())
	require.NoError(saveErr)
	defer pLogEvent.Release()

	// 6. save to WLog
	err = app.Events().PutWlog(pLogEvent)
	require.NoError(err)

	// 7. save CUD
	err = app.Records().Apply(pLogEvent)
	require.NoError(err)

	// Read event from PLog & PLog and reads CUDs demo
	// 8. read PLog
	var pLogEvent1 istructs.IPLogEvent
	_ = app.Events().ReadPLog(context.Background(), 55, 10000, 1,
		func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
			pLogEvent1 = event
			return nil
		})

	require.NotNil(pLogEvent1)
	defer pLogEvent1.Release()

	// 9. read WLog
	var wLogEvent1 istructs.IWLogEvent
	_ = app.Events().ReadWLog(context.Background(), 1234, 1000, 1,
		func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
			wLogEvent1 = event
			return nil
		})

	require.NotNil(wLogEvent1)
	defer wLogEvent1.Release()
}

func TestBasicUsage_ViewRecords(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	appConfigs := func() AppConfigsType {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		view := wsb.AddView(appdef.NewQName("test", "viewDrinks"))
		view.Key().PartKey().AddField("partitionKey1", appdef.DataKind_int64)
		view.Key().ClustCols().
			AddField("clusteringColumn1", appdef.DataKind_int64).
			AddField("clusteringColumn2", appdef.DataKind_bool).
			AddField("clusteringColumn3", appdef.DataKind_string, constraints.MaxLen(100))
		view.Value().
			AddField("id", appdef.DataKind_int64, true).
			AddField("name", appdef.DataKind_string, true).
			AddField("active", appdef.DataKind_bool, true)

		cfgs := make(AppConfigsType, 1)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		return cfgs
	}

	p := Provide(appConfigs(), iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)
	as, err := p.BuiltIn(appName)
	require.NoError(err)
	viewRecords := as.ViewRecords()
	entries := []entryType{
		newEntry(viewRecords, 1, 100, true, "soda", 1, "Cola"),
		newEntry(viewRecords, 1, 100, true, "soda", 2, "Pepsi"), // dupe!
		newEntry(viewRecords, 1, 100, false, "soda", 3, "Sprite"),
		newEntry(viewRecords, 1, 200, false, "wine", 4, "White wine"),
		newEntry(viewRecords, 1, 200, true, "wine", 5, "Red wine"),
		newEntry(viewRecords, 2, 200, true, "wine", 4, "White wine"),
		newEntry(viewRecords, 2, 200, false, "wine", 5, "Red wine"),
	}
	for _, e := range entries {
		err := viewRecords.Put(e.wsid, e.key, e.value)
		require.NoError(err)
	}
	t.Run("Should read all records by WSID", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", int64(1))
		counter := 0

		_ = viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			return nil
		})

		require.Equal(4, counter)
	})
	t.Run("Should read records by WSID and department", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 1)
		kb.PutInt64("clusteringColumn1", 200)
		counter := 0

		_ = viewRecords.Read(context.Background(), 1, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			return nil
		})

		require.Equal(2, counter)
	})
	t.Run("Should read one record by WSID and department and active", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutInt64("clusteringColumn1", 200)
		kb.PutBool("clusteringColumn2", true)
		counter := 0

		_ = viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			return nil
		})

		require.Equal(1, counter)
	})
	t.Run("Should read one record by WSID and department, active and code ignore wrong clustering columns order reason", func(t *testing.T) {
		kb := viewRecords.KeyBuilder(appdef.NewQName("test", "viewDrinks"))
		kb.PutInt64("partitionKey1", 2)
		kb.PutString("clusteringColumn3", "wine")
		kb.PutBool("clusteringColumn2", true)
		kb.PutInt64("clusteringColumn1", 200)
		counter := 0

		_ = viewRecords.Read(context.Background(), 2, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
			counter++
			return nil
		})

		require.Equal(1, counter)
	})
}

func Test_appStructsType_ObjectBuilder(t *testing.T) {
	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	objName := appdef.NewQName("test", "object")

	appStructs := func() istructs.IAppStructs {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))
		obj := wsb.AddObject(objName)
		obj.AddField("int", appdef.DataKind_int64, true)
		obj.AddContainer("child", objName, 0, appdef.Occurs_Unbounded)

		cfgs := make(AppConfigsType)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)
		app, err := provider.BuiltIn(appName)
		require.NoError(err)

		return app
	}()

	t.Run("Should be ok to build known object", func(t *testing.T) {
		b := appStructs.ObjectBuilder(objName)
		require.NotNil(b)

		b.PutInt64("int", 1)

		o, err := b.Build()
		require.NoError(err)

		require.Equal(objName, o.QName())
		require.EqualValues(1, o.AsInt64("int"))
	})

	t.Run("Should be ok to fill object with children from JSON", func(t *testing.T) {
		b := appStructs.ObjectBuilder(objName)
		require.NotNil(b)

		b.FillFromJSON(map[string]any{
			"int": int64(1),
			"child": []any{
				map[string]any{
					"int": int64(2),
				},
			},
		})

		o, err := b.Build()
		require.NoError(err)

		require.Equal(objName, o.QName())
		require.EqualValues(1, o.AsInt64("int"))

		require.Equal(1, func() int {
			cnt := 0
			for c := range o.Children("child") {
				cnt++
				require.EqualValues(2, c.AsInt64("int"))
			}
			return cnt
		}())
	})

	t.Run("Should be error to build unknown object", func(t *testing.T) {
		b := appStructs.ObjectBuilder(appdef.NewQName("test", "unknown"))
		require.NotNil(b)

		b.PutInt64("int", 1)

		o, err := b.Build()
		require.Nil(o)
		require.Error(err, require.Is(ErrNameNotFoundError), require.Has("test.unknown"))
	})
}

// TestBasicUsage_Resources: Demonstrates basic usage resources
func TestBasicUsage_Resources(t *testing.T) {
	require := require.New(t)

	t.Run("Basic usage NewCommandFunction", func(t *testing.T) {
		funcQName := appdef.NewQName("test", "cmd")

		f := NewCommandFunction(funcQName, NullCommandExec)
		require.Equal(funcQName, f.QName())
		require.Equal(istructs.ResourceKind_CommandFunction, f.Kind())

		// Calls have no effect since we use Null* closures

		err := f.Exec(istructs.ExecCommandArgs{})
		require.NoError(err)

		// Test String()
		log.Println(f)
	})

	t.Run("Basic usage NewQueryFunction", func(t *testing.T) {
		myExecQuery := func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
			// Can use NullExecQuery instead of myExecQuery, it does nothing
			NullQueryExec(ctx, args, callback) // nolint errcheck

			require.NoError(callback(&istructs.NullObject{}))
			return nil
		}

		funcQName := appdef.NewQName("test", "query")

		f := NewQueryFunction(funcQName, myExecQuery)
		require.Equal(funcQName, f.QName())
		require.Equal(istructs.ResourceKind_QueryFunction, f.Kind())
		require.Panics(func() { f.ResultType(istructs.PrepareArgs{}) })

		// Depends on myExecQuery
		err := f.Exec(context.Background(), istructs.ExecQueryArgs{}, func(istructs.IObject) error { return nil })
		require.NoError(err)

		// Test String()
		log.Println(f)
	})
}

// Demonstrates basic usage application
func TestBasicUsage_AppDef(t *testing.T) {
	require := require.New(t)
	test := newTest()

	app := test.AppStructs

	t.Run("I. test top level type (command object)", func(t *testing.T) {
		cmdDoc := appdef.ODoc(app.AppDef().Type, test.saleCmdDocName)

		require.NotNil(cmdDoc)
		require.Equal(appdef.TypeKind_ODoc, cmdDoc.Kind())

		// check fields
		fields := make(map[string]appdef.DataKind)
		for _, f := range cmdDoc.Fields() {
			fields[f.Name()] = f.DataKind()
		}
		require.Len(fields, 7) // 2 system {sys.QName, sys.ID} + 5 user
		require.Equal(appdef.DataKind_string, fields[test.buyerIdent])
		require.Equal(appdef.DataKind_int8, fields[test.ageIdent])
		require.Equal(appdef.DataKind_float32, fields[test.heightIdent])
		require.Equal(appdef.DataKind_bool, fields[test.humanIdent])
		require.Equal(appdef.DataKind_bytes, fields[test.photoIdent])

		for _, c := range cmdDoc.Containers() {
			require.Equal(test.basketIdent, c.Name())
			require.Equal(appdef.NewQName(test.pkgName, test.basketIdent), c.QName())
			t.Run("II. test first level nested type (basket)", func(t *testing.T) {
				rec := appdef.ORecord(app.AppDef().Type, appdef.NewQName(test.pkgName, test.basketIdent))
				require.NotNil(rec)
				require.Equal(appdef.TypeKind_ORecord, rec.Kind())

				for _, c := range rec.Containers() {
					require.Equal(test.goodIdent, c.Name())
					require.Equal(appdef.NewQName(test.pkgName, test.goodIdent), c.QName())

					t.Run("III. test second level nested type (good)", func(t *testing.T) {
						rec := appdef.ORecord(app.AppDef().Type, appdef.NewQName(test.pkgName, test.goodIdent))
						require.NotNil(rec)
						require.Equal(appdef.TypeKind_ORecord, rec.Kind())

						fields := make(map[string]appdef.DataKind)
						for _, f := range rec.Fields() {
							fields[f.Name()] = f.DataKind()
						}
						require.Len(fields, 8) // 4 system {sys.QName, sys.ID, sys.ParentID, sys.Container} + 4 user
						require.Equal(appdef.DataKind_RecordID, fields[test.saleIdent])
						require.Equal(appdef.DataKind_string, fields[test.nameIdent])
						require.Equal(appdef.DataKind_int64, fields[test.codeIdent])
						require.Equal(appdef.DataKind_float64, fields[test.weightIdent])
					})
				}
			})
		}
	})
}

func Test_BasicUsageDescribePackages(t *testing.T) {

	require := require.New(t)

	appName := istructs.AppQName_test1_app1

	app := func() istructs.IAppStructs {
		adb := builder.New()
		adb.AddPackage("structs", "test.com/structs")
		adb.AddPackage("functions", "test.com/functions")
		adb.AddPackage("workspaces", "test.com/workspace")

		wsQName := appdef.NewQName("workspaces", "test")
		docQName := appdef.NewQName("structs", "CDoc")
		recQName := appdef.NewQName("structs", "CRec")
		viewQName := appdef.NewQName("structs", "View")
		cmdQName := appdef.NewQName("functions", "cmd")
		queryQName := appdef.NewQName("functions", "query")
		argQName := appdef.NewQName("structs", "Arg")

		wsb := adb.AddWorkspace(wsQName)
		wsb.AddCDoc(appdef.NewQName("test", "WSDesc"))
		wsb.SetDescriptor(appdef.NewQName("test", "WSDesc"))

		rec := wsb.AddCRecord(recQName)
		rec.AddField("int", appdef.DataKind_int64, false)

		doc := wsb.AddCDoc(docQName)
		doc.AddField("str", appdef.DataKind_string, true)
		doc.AddField("fld", appdef.DataKind_int32, true)
		doc.SetUniqueField("fld")
		un1 := appdef.NewQName("structs", "uniq1")
		doc.AddUnique(un1, []string{"str"})

		doc.AddContainer("rec", recQName, 0, appdef.Occurs_Unbounded)

		view := wsb.AddView(viewQName)
		view.Key().PartKey().AddField("int", appdef.DataKind_int64)
		view.Key().ClustCols().AddField("str", appdef.DataKind_string, constraints.MaxLen(100))
		view.Value().AddField("bool", appdef.DataKind_bool, false)

		arg := wsb.AddObject(argQName)
		arg.AddField("bool", appdef.DataKind_bool, false)

		wsb.AddCommand(cmdQName).
			SetParam(argQName).
			SetResult(docQName)
		wsb.AddQuery(queryQName).
			SetParam(argQName).
			SetResult(appdef.QNameANY)

		cfgs := make(AppConfigsType)
		cfg := cfgs.AddBuiltInAppConfig(appName, adb)
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

		cfg.Resources.Add(NewCommandFunction(cmdQName, NullCommandExec))
		cfg.Resources.Add(NewQueryFunction(queryQName, NullQueryExec))

		cfg.FunctionRateLimits.AddAppLimit(queryQName, istructs.RateLimit{
			Period:                1,
			MaxAllowedPerDuration: 2,
		})
		cfg.FunctionRateLimits.AddWorkspaceLimit(queryQName, istructs.RateLimit{
			Period:                3,
			MaxAllowedPerDuration: 4,
		})

		provider := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)
		app, err := provider.BuiltIn(appName)
		require.NoError(err)

		return app
	}()

	pkgNames := app.DescribePackageNames()
	require.Len(pkgNames, 3)

	for _, name := range pkgNames {
		pkg := app.DescribePackage(name)
		require.NotNil(pkg)

		bytes, err := json.Marshal(pkg)
		require.NoError(err)

		logger.Info("package: ", name)
		logger.Info(string(bytes))
	}
}

func Test_Provide(t *testing.T) {
	require := require.New(t)
	test := newTest()

	t.Run("AppStructs() must error if unknown app name", func(t *testing.T) {
		cfgs := make(AppConfigsType)
		cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, builder.New())
		cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		p := Provide(cfgs, iratesce.TestBucketsFactory, testTokensFactory(), nil, isequencer.SequencesTrustLevel_0, nil)
		require.NotNil(p)

		_, err := p.BuiltIn(appdef.NewAppQName("test1", "unknownApp"))
		require.ErrorIs(err, istructs.ErrAppNotFound)
		require.ErrorContains(err, "test1/unknownApp")
	})

	t.Run("check application ClusterAppID() and AppQName()", func(t *testing.T) {
		provider := Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), simpleStorageProvider(), isequencer.SequencesTrustLevel_0, nil)

		app, err := provider.BuiltIn(test.appName)
		require.NoError(err)

		require.NotNil(app)

		require.Equal(istructs.ClusterAppID_test1_app1, app.ClusterAppID())
		require.Equal(istructs.AppQName_test1_app1, app.AppQName())
	})
}
