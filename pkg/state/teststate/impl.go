/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import (
	"context"
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istorage/mem"
	wsdescutil "github.com/voedger/voedger/pkg/utils/testwsdesc"

	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type testState struct {
	state.IState

	ctx                   context.Context
	appStructs            istructs.IAppStructs
	appDef                appdef.IAppDef
	cud                   istructs.ICUD
	event                 istructs.IPLogEvent
	plogGen               istructs.IIDGenerator
	wsOffsets             map[istructs.WSID]istructs.Offset
	plogOffset            istructs.Offset
	secretReader          isecrets.ISecretReader
	httpHandler           HttpHandlerFunc
	federationCmdHandler  state.FederationCommandHandler
	federationBlobHandler state.FederationBlobHandler
	uniquesHandler        state.UniquesHandler
	principals            []iauthnz.Principal
	token                 string
	queryWsid             istructs.WSID
	queryName             appdef.FullQName
	processorKind         int
	readObjects           []istructs.IObject
	queryObject           istructs.IObject
}

func NewTestState(processorKind int, packagePath string, createWorkspaces ...TestWorkspace) ITestState {
	ts := &testState{}
	ts.ctx = context.Background()
	ts.processorKind = processorKind
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
	if ctx.processorKind == ProcKind_QueryProcessor {
		return ctx.queryWsid
	}
	return ctx.event.Workspace()
}

func (ctx *testState) GetReadObjects() []istructs.IObject {
	return ctx.readObjects
}

func (ctx *testState) Arg() istructs.IObject {
	if ctx.queryObject != nil {
		return ctx.queryObject
	}
	if ctx.event == nil {
		panic("no current event")
	}
	return ctx.event.ArgumentObject()
}

func (ctx *testState) ResultBuilder() istructs.IObjectBuilder {
	if ctx.event == nil {
		panic("no current event")
	}
	qname := ctx.event.QName()
	command := ctx.appDef.Command(qname)
	if command == nil {
		panic(fmt.Sprintf("%v is not a command", qname))
	}
	return ctx.appStructs.ObjectBuilder(command.Result().QName())
}

func (ctx *testState) Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error) {
	if ctx.httpHandler == nil {
		panic("http handler not set")
	}
	req := HttpRequest{
		Timeout: timeout,
		Method:  method,
		URL:     url,
		Body:    body,
		Headers: headers,
	}
	resp, err := ctx.httpHandler(req)
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.Status, resp.Body, resp.Headers, nil
}

func (ctx *testState) PutQuery(wsid istructs.WSID, name appdef.FullQName, argb QueryArgBuilderCallback) {
	ctx.queryWsid = wsid
	ctx.queryName = name

	if argb != nil {
		localPkgName := ctx.appDef.PackageLocalName(ctx.queryName.PkgPath())
		query := ctx.appDef.Query(appdef.NewQName(localPkgName, ctx.queryName.Entity()))
		if query == nil {
			panic(fmt.Sprintf("query not found: %v", ctx.queryName))
		}
		ab := ctx.appStructs.ObjectBuilder(query.Param().QName())
		argb(ab)
		qo, err := ab.Build()
		if err != nil {
			panic(err)
		}
		ctx.queryObject = qo
	}
}

func (ctx *testState) PutRequestSubject(principals []iauthnz.Principal, token string) {
	ctx.principals = principals
	ctx.token = token
}

func (ctx *testState) PutFederationCmdHandler(emu state.FederationCommandHandler) {
	ctx.federationCmdHandler = emu
}

func (ctx *testState) PutFederationBlobHandler(emu state.FederationBlobHandler) {
	ctx.federationBlobHandler = emu
}

func (ctx *testState) PutUniquesHandler(emu state.UniquesHandler) {
	ctx.uniquesHandler = emu
}

func (ctx *testState) emulateUniquesHandler(entity appdef.QName, wsid istructs.WSID, data map[string]interface{}) (istructs.RecordID, error) {
	if ctx.uniquesHandler == nil {
		panic("uniques handler not set")
	}
	return ctx.uniquesHandler(entity, wsid, data)
}

func (ctx *testState) emulateFederationCmd(owner, appname string, wsid istructs.WSID, command appdef.QName, body string) (statusCode int, newIDs map[string]int64, result string, err error) {
	if ctx.federationCmdHandler == nil {
		panic("federation command handler not set")
	}
	return ctx.federationCmdHandler(owner, appname, wsid, command, body)
}

func (ctx *testState) emulateFederationBlob(owner, appname string, wsid istructs.WSID, blobId int64) ([]byte, error) {
	if ctx.federationBlobHandler == nil {
		panic("federation blob handler not set")
	}
	return ctx.federationBlobHandler(owner, appname, wsid, blobId)
}

func (ctx *testState) buildState(processorKind int) {

	appFunc := func() istructs.IAppStructs { return ctx.appStructs }
	eventFunc := func() istructs.IPLogEvent { return ctx.event }
	partitionIDFunc := func() istructs.PartitionID { return TestPartition }
	cudFunc := func() istructs.ICUD { return ctx.cud }
	commandPrepareArgs := func() istructs.CommandPrepareArgs {
		return istructs.CommandPrepareArgs{
			PrepareArgs: istructs.PrepareArgs{
				Workpiece:      nil,
				ArgumentObject: ctx.Arg(),
				WSID:           ctx.WSID(),
				Workspace:      nil,
			},
			ArgumentUnloggedObject: nil,
		}
	}
	argFunc := func() istructs.IObject { return ctx.Arg() }
	unloggedArgFunc := func() istructs.IObject { return nil }
	wlogOffsetFunc := func() istructs.Offset { return ctx.event.WLogOffset() }
	wsidFunc := func() istructs.WSID {
		return ctx.WSID()
	}
	resultBuilderFunc := func() istructs.IObjectBuilder {
		return ctx.ResultBuilder()
	}
	principalsFunc := func() []iauthnz.Principal {
		return ctx.principals
	}
	tokenFunc := func() string {
		return ctx.token
	}
	execQueryArgsFunc := func() istructs.PrepareArgs {
		return istructs.PrepareArgs{
			Workpiece:      nil,
			ArgumentObject: ctx.Arg(),
			WSID:           ctx.WSID(),
			Workspace:      nil,
		}
	}
	qryResultBuilderFunc := func() istructs.IObjectBuilder {
		localPkgName := ctx.appDef.PackageLocalName(ctx.queryName.PkgPath())
		query := ctx.appDef.Query(appdef.NewQName(localPkgName, ctx.queryName.Entity()))
		if query == nil {
			panic(fmt.Sprintf("query not found: %v", ctx.queryName))
		}
		return ctx.appStructs.ObjectBuilder(query.Result().QName())
	}
	execQueryCallback := func() istructs.ExecQueryCallback {
		return func(o istructs.IObject) error {
			ctx.readObjects = append(ctx.readObjects, o)
			return nil
		}
	}

	switch processorKind {
	case ProcKind_Actualizer:
		ctx.IState = state.ProvideAsyncActualizerStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, nil, ctx.secretReader, eventFunc, nil, nil,
			IntentsLimit, BundlesLimit, state.WithCustomHttpClient(ctx), state.WithFedearationCommandHandler(ctx.emulateFederationCmd), state.WithUniquesHandler(ctx.emulateUniquesHandler), state.WithFederationBlobHandler(ctx.emulateFederationBlob))
	case ProcKind_CommandProcessor:
		ctx.IState = state.ProvideCommandProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, cudFunc, principalsFunc, tokenFunc,
			IntentsLimit, resultBuilderFunc, commandPrepareArgs, argFunc, unloggedArgFunc, wlogOffsetFunc, state.WithUniquesHandler(ctx.emulateUniquesHandler))
	case ProcKind_QueryProcessor:
		ctx.IState = state.ProvideQueryProcessorStateFactory()(ctx.ctx, appFunc, partitionIDFunc, wsidFunc, ctx.secretReader, principalsFunc, tokenFunc, nil,
			execQueryArgsFunc, argFunc, qryResultBuilderFunc, nil, execQueryCallback,
			state.WithCustomHttpClient(ctx), state.WithFedearationCommandHandler(ctx.emulateFederationCmd), state.WithUniquesHandler(ctx.emulateUniquesHandler), state.WithFederationBlobHandler(ctx.emulateFederationBlob))
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
	`, packagePath, TestPkgAlias, TestPkgAlias))
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

	// TODO: obtain app name from packages
	// appName := appSchema.AppQName()

	appName := istructs.AppQName_test1_app1

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
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	ctx.appDef.Extensions(func(i appdef.IExtension) {
		if i.QName().Pkg() == TestPkgAlias {
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
	structs, err := prov.BuiltIn(appName)
	if err != nil {
		panic(err)
	}
	ctx.appStructs = structs
	ctx.plogGen = istructsmem.NewIDGenerator()
	ctx.wsOffsets = make(map[istructs.WSID]istructs.Offset)

	for _, ws := range createWorkspaces {
		err = wsdescutil.CreateCDocWorkspaceDescriptorStub(ctx.appStructs, TestPartition, ws.WSID, appdef.NewQName(TestPkgAlias, ws.WorkspaceDescriptor), ctx.nextPLogOffs(), ctx.nextWSOffs(ws.WSID))
		if err != nil {
			panic(err)
		}
	}
}

func (ctx *testState) nextPLogOffs() istructs.Offset {
	ctx.plogOffset += 1
	return ctx.plogOffset
}

func (ctx *testState) nextWSOffs(ws istructs.WSID) istructs.Offset {
	offs, ok := ctx.wsOffsets[ws]
	if !ok {
		offs = istructs.Offset(0)
	}
	offs += 1
	ctx.wsOffsets[ws] = offs
	return offs
}

func (ctx *testState) PutHttpHandler(handler HttpHandlerFunc) {
	ctx.httpHandler = handler

}

func (ctx *testState) PutRecords(wsid istructs.WSID, cb NewRecordsCallback) (wLogOffs istructs.Offset, newRecordIds []istructs.RecordID) {
	return ctx.PutEvent(wsid, appdef.NewFullQName(istructs.QNameCommandCUD.Pkg(), istructs.QNameCommandCUD.Entity()), func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD) {
		cb(cudBuilder)
	})
}

func (ctx *testState) GetRecord(wsid istructs.WSID, id istructs.RecordID) istructs.IRecord {
	var rec istructs.IRecord
	rec, err := ctx.appStructs.Records().Get(wsid, false, id)
	if err != nil {
		panic(err)
	}
	return rec
}

func (ctx *testState) PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback) (wLogOffs istructs.Offset, newRecordIds []istructs.RecordID) {
	var localPkgName string
	if name.PkgPath() == appdef.SysPackage {
		localPkgName = name.PkgPath()
	} else {
		localPkgName = ctx.appDef.PackageLocalName(name.PkgPath())
	}
	wLogOffs = ctx.nextWSOffs(wsid)
	reb := ctx.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: TestPartition,
			QName:             appdef.NewQName(localPkgName, name.Entity()),
			WLogOffset:        wLogOffs,
			PLogOffset:        ctx.nextPLogOffs(),
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

	err = ctx.appStructs.Events().PutWlog(event)
	if err != nil {
		panic(err)
	}

	newRecordIds = make([]istructs.RecordID, 0)
	err = ctx.appStructs.Records().Apply2(event, func(r istructs.IRecord) {
		newRecordIds = append(newRecordIds, r.ID())
	})

	if err != nil {
		panic(err)
	}

	ctx.event = event
	return wLogOffs, newRecordIds
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
	if ia.vb == nil {
		require.Fail(ia.t, "expected intent to exist")
	}
}

func (ia *intentAssertions) Assert(cb IntentAssertionsCallback) {
	if ia.vb == nil {
		require.Fail(ia.t, "expected intent to exist")
		return
	}
	value := ia.vb.BuildValue()
	if value == nil {
		require.Fail(ia.t, "value builder does not support Assert operation")
		return
	}
	cb(require.New(ia.t), value)
}

func (ia *intentAssertions) Equal(vbc ValueBuilderCallback) {
	if ia.vb == nil {
		panic("intent not found")
	}

	vb, err := ia.ctx.IState.NewValue(ia.kb)
	if err != nil {
		panic(err)
	}
	vbc(vb)

	if !ia.vb.Equal(vb) {
		require.Fail(ia.t, "Expected intents to be equal")
	}
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
