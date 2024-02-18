/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	parser "github.com/voedger/voedger/pkg/parser"
)

func TestEventStorage_Get(t *testing.T) {
	require := require.New(t)
	testQName := appdef.NewQName("main", "Command")

	app := appStructs(
		`APPLICATION test(); 
		WORKSPACE ws1 (
			EXTENSION ENGINE BUILTIN(
				COMMAND Command();
			);
		)
		`,
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1
	wsid := istructs.WSID(1)               // test within workspace 1
	offset := istructs.Offset(123)         // test within offset 1

	eventFunc := func() istructs.IPLogEvent {
		reb := app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				Workspace:         wsid,
				HandlingPartition: partitionNr,
				PLogOffset:        offset,
				QName:             testQName,
			},
		})
		rawEvent, err := reb.BuildRawEvent()
		if err != nil {
			panic(err)
		}
		event, err := app.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
		if err != nil {
			panic(err)
		}
		return event
	}

	s := ProvideAsyncActualizerStateFactory()(context.Background(), app, nil, nil, nil, nil, eventFunc, 0, 0)
	kb, err := s.KeyBuilder(Event, appdef.NullQName)
	require.NoError(err)
	value, err := s.MustExist(kb)
	require.NotNil(value)
	require.NoError(err)

	require.Equal(int64(wsid), value.AsInt64(Field_Workspace))
	require.Equal(testQName, value.AsQName(Field_QName))
	// TODO: test other fields
}

type (
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

//go:embed sql_example_syspkg/*.sql
var sfs embed.FS

func appStructs(appdefSql string, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	appDef := appdef.New()

	fs, err := parser.ParseFile("file1.sql", appdefSql)
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

	err = parser.BuildAppDefs(packages, appDef)
	if err != nil {
		panic(err)
	}

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
	}

	asf := mem.Provide()
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider)
	structs, err := prov.AppStructs(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	return structs
}
