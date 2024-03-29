/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package istatetestctx

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/sys/authnz"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type extPackageContext struct {
	ctx        context.Context
	appStructs istructs.IAppStructs
	appDef     appdef.IAppDef
	// engine       iextengine.IExtensionEngine
	io           iextengine.IExtensionIO
	cud          istructs.ICUD
	event        istructs.IPLogEvent
	plogGen      istructs.IIDGenerator
	secretReader isecrets.ISecretReader
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

/*func newModuleURL(path string) (u *url.URL) {
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	u, err = url.Parse("file:///" + filepath.ToSlash(path))
	if err != nil {
		panic(err)
	}
	return
}*/

func NewPackageContext(processorKind int, packagePath string, createWorkspaces ...TestWorkspace) IExtTestContext {
	ctx := &extPackageContext{}
	ctx.ctx = context.Background()
	ctx.secretReader = &secretReader{secrets: make(map[string][]byte)}
	ctx.buildAppDef(packagePath, ".", createWorkspaces...)
	//ctx.buildEngine(packagePath, packageDir, extKind)
	ctx.buildState(processorKind)
	return ctx
}

func (ctx *extPackageContext) WSID() istructs.WSID {
	return ctx.event.Workspace() // TODO: For QP must be different
}

func (ctx *extPackageContext) Arg() istructs.IObject {
	return ctx.event.ArgumentObject() // TODO: For QP must be different
}

/*func (ctx *extPackageContext) Close() {
	ctx.engine.Close(context.Background())
}*/

func (ctx *extPackageContext) buildState(processorKind int) {

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
		ctx.io = state.ProvideAsyncActualizerStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, nil, ctx.secretReader, eventFunc, IntentsLimit, BundlesLimit)
	case ProcKind_CommandProcessor:
		ctx.io = state.ProvideCommandProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, cudFunc, nil, nil, IntentsLimit, nil, argFunc, unloggedArgFunc)
	case ProcKind_QueryProcessor:
		ctx.io = state.ProvideQueryProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, nil, nil, argFunc)
	}
}

/*
func (ctx *extPackageContext) buildEngine(packagePath string, packageDir string, engineKind appdef.ExtensionEngineKind) {
	var extNames []string

	ctx.appDef.Extensions(func(i appdef.IExtension) {
		if i.QName().Pkg() == testPkgAlias && i.Engine() == engineKind {
			extNames = append(extNames, i.Name())
		}
	})

	packages := []iextengine.ExtensionPackage{
		{
			QualifiedName:  packagePath,
			ModuleUrl:      newModuleURL(filepath.Join(packageDir, wasmFilename)),
			ExtensionNames: extNames,
		},
	}

	var factory iextengine.IExtensionEngineFactory
	if engineKind == appdef.ExtensionEngineKind_WASM {
		factory = iextenginewazero.ProvideExtensionEngineFactory(true)
	}
	engines, err := factory.New(ctx.ctx, packages, &iextengine.ExtEngineConfig{}, 1)
	if err != nil {
		panic(err)
	}
	ctx.engine = engines[0]
}*/

//go:embed testsys/*.sql
var fsTestSys embed.FS

func (ctx *extPackageContext) buildAppDef(packagePath string, packageDir string, createWorkspaces ...TestWorkspace) {

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

/*func (ctx *extPackageContext) Invoke(name appdef.FullQName) error {
	return ctx.engine.Invoke(context.Background(), name, ctx.io)
}*/

func (ctx *extPackageContext) PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback) {
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

func (ctx *extPackageContext) PutView(wsid istructs.WSID, entity appdef.FullQName, callback ViewValueCallback) {
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

func (ctx *extPackageContext) PutSecret(name string, secret []byte) {
	ctx.secretReader.(*secretReader).secrets[name] = secret
}

func (ctx *extPackageContext) HasIntent(storage appdef.QName, entity appdef.FullQName, callback HasIntentCallback) bool {
	localPkgName := ctx.appDef.PackageLocalName(entity.PkgPath())
	localEntity := appdef.NewQName(localPkgName, entity.Entity())
	kb, err := ctx.io.KeyBuilder(storage, localEntity)
	if err != nil {
		panic(err)
	}
	intent := ctx.io.FindIntent(kb)
	if intent == nil {
		return false
	}
	bIntens, err := intent.ToBytes()
	if err != nil {
		panic(err)
	}

	vb, err := ctx.io.NewValue(kb)
	if err != nil {
		panic(err)
	}
	callback(kb, vb)
	bVb, err := vb.ToBytes()
	if err != nil {
		panic(err)
	}

	return reflect.DeepEqual(bIntens, bVb)
}
