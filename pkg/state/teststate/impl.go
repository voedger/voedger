/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage/mem"

	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type testState struct {
	state.IState

	ctx          context.Context
	appStructs   istructs.IAppStructs
	appDef       appdef.IAppDef
	cud          istructs.ICUD
	event        istructs.IPLogEvent
	plogGen      istructs.IIDGenerator
	secretReader isecrets.ISecretReader
}

func NewTestState(processorKind int, packagePath string, createWorkspaces ...TestWorkspace) ITestState {
	ts := &testState{}
	ts.ctx = context.Background()
	ts.secretReader = &secretReader{secrets: make(map[string][]byte)}
	ts.buildAppDef(packagePath, ".", createWorkspaces...)
	ts.buildState(processorKind)
	return ts
}

type secretReader struct {
	secrets map[string][]byte
}

func (s *secretReader) ReadSecret(name string) (bb []byte, err error) {
	if bb, ok := s.secrets[name]; ok {
		return bb, nil
	}
	return nil, fmt.Errorf("secret not found: %s", name)
}

func (ctx *testState) WSID() istructs.WSID {
	return ctx.event.Workspace() // TODO: For QP must be different
}

func (ctx *testState) Arg() istructs.IObject {
	return ctx.event.ArgumentObject() // TODO: For QP must be different
}

func (ctx *testState) buildState(processorKind int) {

	appFunc := func() istructs.IAppStructs { return ctx.appStructs }
	eventFunc := func() istructs.IPLogEvent { return ctx.event }
	partitionIDFunc := func() istructs.PartitionID { return testPartition }
	cudFunc := func() istructs.ICUD { return ctx.cud }
	argFunc := func() istructs.IObject { return ctx.Arg() }
	unloggedArgFunc := func() istructs.IObject { return nil }
	wsidFunc := func() istructs.WSID {
		return ctx.WSID()
	}

	switch processorKind {
	case ProcKind_Actualizer:
		ctx.IState = state.ProvideAsyncActualizerStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, nil, ctx.secretReader, eventFunc, IntentsLimit, BundlesLimit)
	case ProcKind_CommandProcessor:
		ctx.IState = state.ProvideCommandProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, cudFunc, nil, nil, IntentsLimit, nil, argFunc, unloggedArgFunc)
	case ProcKind_QueryProcessor:
		ctx.IState = state.ProvideQueryProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, nil, nil, argFunc)
	}
}

//go:embed testsys/*.sql
var fsTestSys embed.FS

func (ctx *testState) buildAppDef(packagePath string, packageDir string, createWorkspaces ...TestWorkspace) {

	absPath, err := filepath.Abs(packageDir)
	if err != nil {
		panic(err)
	}

	pkgAst, err := parser.ParsePackageDir(packagePath, coreutils.NewPathReader(absPath), "")
	if err != nil {
		panic(err)
	}
	sysPackageAST, err := parser.ParsePackageDir(appdef.SysPackage, fsTestSys, "testsys")
	if err != nil {
		panic(err)
	}
	dummyAppFileAST, err := parser.ParseFile("dummy.sql", fmt.Sprintf(`
		IMPORT SCHEMA '%s' AS %s;
		APPLICATION test(
			USE %s;
		);
	`, packagePath, testPkgAlias, testPkgAlias))
	if err != nil {
		panic(err)
	}
	dummyAppPkgAST, err := parser.BuildPackageSchema(packagePath+"_app", []*parser.FileSchemaAST{dummyAppFileAST})
	if err != nil {
		panic(err)
	}

	packagesAST := []*parser.PackageSchemaAST{pkgAst, dummyAppPkgAST, sysPackageAST}

	appSchema, err := parser.BuildAppSchema(packagesAST)
	if err != nil {
		panic(err)
	}

	adb := appdef.New()
	err = parser.BuildAppDefs(appSchema, adb)
	if err != nil {
		panic(err)
	}

	adf, err := adb.Build()
	if err != nil {
		panic(err)
	}

	ctx.appDef = adf

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, adb)
	cfg.Resources.Add(istructsmem.NewCommandFunction(newWorkspaceCmd, istructsmem.NullCommandExec))
	ctx.appDef.Extensions(func(i appdef.IExtension) {
		if i.QName().Pkg() == testPkgAlias {
			if proj, ok := i.(appdef.IProjector); ok {
				if proj.Sync() {
					cfg.AddSyncProjectors(istructs.Projector{Name: i.QName()})
				} else {
					cfg.AddAsyncProjectors(istructs.Projector{Name: i.QName()})
				}
			} else if cmd, ok := i.(appdef.ICommand); ok {
				cfg.Resources.Add(istructsmem.NewCommandFunction(cmd.QName(), istructsmem.NullCommandExec))
			} else if q, ok := i.(appdef.IQuery); ok {
				cfg.Resources.Add(istructsmem.NewCommandFunction(q.QName(), istructsmem.NullCommandExec))
			}
		}
	})

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
	ctx.appStructs = structs
	ctx.plogGen = istructsmem.NewIDGenerator()

	for _, ws := range createWorkspaces {
		rebWs := ctx.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				Workspace:         istructs.WSID(ws.WSID),
				HandlingPartition: testPartition,
				QName:             newWorkspaceCmd,
			},
		})
		cud := rebWs.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
		cud.PutRecordID(appdef.SystemField_ID, istructs.RecordID(1))
		cud.PutQName("WSKind", appdef.NewQName(testPkgAlias, ws.WorkspaceDescriptor))
		rawWsEvent, err := rebWs.BuildRawEvent()
		if err != nil {
			panic(err)
		}
		wsEvent, err := ctx.appStructs.Events().PutPlog(rawWsEvent, nil, ctx.plogGen)
		if err != nil {
			panic(err)
		}
		err = ctx.appStructs.Records().Apply(wsEvent)
		if err != nil {
			panic(err)
		}
	}

}

func (ctx *testState) PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback) {
	localPkgName := ctx.appDef.PackageLocalName(name.PkgPath())
	reb := ctx.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: testPartition,
			//			PLogOffset:        offset + 1,
			QName: appdef.NewQName(localPkgName, name.Entity()),
		},
	})
	if cb != nil {
		ctx.cud = reb.CUDBuilder()
		cb(reb.ArgumentObjectBuilder(), ctx.cud)
	}
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		panic(err)
	}
	event, err := ctx.appStructs.Events().PutPlog(rawEvent, nil, ctx.plogGen)
	if err != nil {
		panic(err)
	}
	ctx.event = event
}

func (ctx *testState) PutView(wsid istructs.WSID, entity appdef.FullQName, callback ViewValueCallback) {
	localPkgName := ctx.appDef.PackageLocalName(entity.PkgPath())
	v := TestViewValue{
		wsid: wsid,
		vr:   ctx.appStructs.ViewRecords(),
		Key:  ctx.appStructs.ViewRecords().KeyBuilder(appdef.NewQName(localPkgName, entity.Entity())),
		Val:  ctx.appStructs.ViewRecords().NewValueBuilder(appdef.NewQName(localPkgName, entity.Entity())),
	}
	callback(v.Key, v.Val)
	err := ctx.appStructs.ViewRecords().Put(wsid, v.Key, v.Val)
	if err != nil {
		panic(err)
	}
}

func (ctx *testState) PutSecret(name string, secret []byte) {
	ctx.secretReader.(*secretReader).secrets[name] = secret
}

type intentAssertions struct {
	t   *testing.T
	kb  istructs.IStateKeyBuilder
	vb  istructs.IStateValueBuilder
	ctx *testState
}

func (ia *intentAssertions) Exists() {
	require.NotNil(ia.t, ia.vb, "Expected intent to exist")
}

func (ia *intentAssertions) Equal(vbc ValueBuilderCallback) {
	if ia.vb == nil {
		panic("intent not found")
	}
	bIntens, err := ia.vb.ToBytes()
	if err != nil {
		panic(err)
	}

	vb, err := ia.ctx.IState.NewValue(ia.kb)
	if err != nil {
		panic(err)
	}
	vbc(vb)
	bVb, err := vb.ToBytes()
	if err != nil {
		panic(err)
	}

	require.True(ia.t, reflect.DeepEqual(bIntens, bVb), "Expected intents to be equal")

}

func (ctx *testState) RequireIntent(t *testing.T, storage appdef.QName, entity appdef.FullQName, kbc KeyBuilderCallback) IIntentAssertions {
	localPkgName := ctx.appDef.PackageLocalName(entity.PkgPath())
	localEntity := appdef.NewQName(localPkgName, entity.Entity())
	kb, err := ctx.IState.KeyBuilder(storage, localEntity)
	if err != nil {
		panic(err)
	}
	kbc(kb)
	return &intentAssertions{
		t:   t,
		kb:  kb,
		vb:  ctx.IState.FindIntent(kb),
		ctx: ctx,
	}
}
