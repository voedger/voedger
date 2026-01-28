/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestEventStorage_Get(t *testing.T) {
	require := require.New(t)
	testQName := appdef.NewQName("main", "Command")

	app := appStructs(
		`APPLICATION test();
		WORKSPACE ws1 (
			DESCRIPTOR ();
			TABLE t1 INHERITS sys.CDoc (
				x int32
			);
			TYPE CommandParam(
				i int32
			);
			EXTENSION ENGINE BUILTIN(
				COMMAND Command(CommandParam);
			);
		)
		`,
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1)
	wsid := istructs.WSID(1)
	offset := istructs.Offset(123)
	tQname := appdef.NewQName("main", "t1")

	reb := app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: partitionNr,
			PLogOffset:        offset,
			QName:             testQName,
		},
	})
	argb := reb.ArgumentObjectBuilder()
	argb.PutInt32("i", 1)
	_, err := argb.Build()
	require.NoError(err)

	cud := reb.CUDBuilder()
	rw := cud.Create(tQname)
	rw.PutRecordID(appdef.SystemField_ID, 1)
	rw.PutInt32("x", 1)

	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		panic(err)
	}

	event, err := app.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		panic(err)
	}

	eventFunc := func() istructs.IPLogEvent {
		return event
	}

	storage := NewEventStorage(eventFunc)

	kb := storage.NewKeyBuilder(appdef.NullQName, nil)

	value, err := storage.(state.IWithGet).Get(kb)
	require.NotNil(value)
	require.NoError(err)

	require.Equal(int64(wsid), value.AsInt64(sys.Storage_Event_Field_Workspace))
	require.Equal(int64(0), value.AsInt64(sys.Storage_Event_Field_RegisteredAt))
	require.Equal(int64(0), value.AsInt64(sys.Storage_Event_Field_SyncedAt))
	require.Equal(int64(0), value.AsInt64(sys.Storage_Event_Field_Offset))
	require.Equal(int64(0), value.AsInt64(sys.Storage_Event_Field_WLogOffset))
	require.Equal(int64(0), value.AsInt64(sys.Storage_Event_Field_DeviceID))
	require.Equal(testQName, value.AsQName(sys.Storage_Event_Field_QName))
	require.False(value.AsBool(sys.Storage_Event_Field_Synced))

	v := value.AsValue(sys.Storage_Event_Field_ArgumentObject)
	require.NotNil(v)
	require.Equal(int32(1), v.AsInt32("i"))

	c := value.AsValue(sys.Storage_Event_Field_CUDs)
	require.NotNil(c)
	require.Equal(1, c.Length())
	cud1 := c.GetAsValue(0)
	require.NotNil(cud1)
	require.Equal(int32(1), cud1.AsInt32("x"))
	require.Equal(tQname, cud1.AsQName("sys.QName"))

}

type (
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

//go:embed sql_example_syspkg/*.vsql
var sfs embed.FS

func appStructs(appdefSQL string, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	fs, err := parser.ParseFile("file1.vsql", appdefSQL)
	if err != nil {
		panic(err)
	}

	pkg, err := parser.BuildPackageSchema("test/main", []*parser.FileSchemaAST{fs})
	if err != nil {
		panic(err)
	}

	pkgSys, err := parser.ParsePackageDir(appdef.SysPackage, sfs, "sql_example_syspkg")
	if err != nil {
		panic(err)
	}

	packages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		pkgSys,
		pkg,
	})
	if err != nil {
		panic(err)
	}

	// TODO: obtain app name from packages
	// appName := packages.AppQName()
	// require.Equal(t, istructs.AppQName_test1_app1, packages.AppQName())

	appName := istructs.AppQName_test1_app1

	appDef := builder.New()

	err = parser.BuildAppDefs(packages, appDef)
	if err != nil {
		panic(err)
	}

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, appDef)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
	}

	asf := mem.Provide(testingu.MockTime)
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider,
		isequencer.SequencesTrustLevel_0,
		nil,
	)
	structs, err := prov.BuiltIn(appName)
	if err != nil {
		panic(err)
	}
	return structs
}
