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
	"github.com/voedger/voedger/pkg/appdef/builder"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys"

	"github.com/voedger/voedger/pkg/coreutils"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
)

type testState struct {
	state.IState

	ctx                   context.Context
	appStructs            istructs.IAppStructs
	appDef                appdef.IAppDef
	cud                   istructs.ICUD
	ipLogEvent            istructs.IPLogEvent
	plogGen               istructs.IIDGenerator
	wsOffsets             map[istructs.WSID]istructs.Offset
	plogOffset            istructs.Offset
	secretReader          isecrets.ISecretReader
	httpHandler           HTTPHandlerFunc
	federationCmdHandler  state.FederationCommandHandler
	federationBlobHandler state.FederationBlobHandler
	uniquesHandler        state.UniquesHandler
	emailSender           state.IEmailSender
	principals            []iauthnz.Principal
	token                 string
	queryWsid             istructs.WSID
	queryName             appdef.FullQName
	processorKind         int
	readObjects           []istructs.IObject
	queryObject           istructs.IObject

	testData map[string]any
	// argumentType and argumentObject are to pass to argument
	argumentType   appdef.FullQName
	argumentObject map[string]any
	t              *testing.T
	commandWSID    istructs.WSID
	origin         string
}

func NewTestState(processorKind int, packagePath string, createWorkspaces ...TestWorkspace) ITestState {
	ts := &testState{}
	ts.ctx = context.Background()
	ts.processorKind = processorKind
	ts.secretReader = &secretReader{secrets: make(map[string][]byte)}
	ts.buildAppDef(packagePath, "..", createWorkspaces...)
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

func (ts *testState) WSID() istructs.WSID {
	switch ts.processorKind {
	case ProcKind_QueryProcessor:
		return ts.queryWsid
	case ProcKind_CommandProcessor:
		// for command processor kind first look for WSID in event field
		if ts.ipLogEvent != nil {
			return ts.ipLogEvent.Workspace()
		}

		return ts.commandWSID
	default:
		if ts.ipLogEvent != nil {
			return ts.ipLogEvent.Workspace()
		}

		return istructs.WSID(0)
	}
}

func (ts *testState) GetReadObjects() []istructs.IObject {
	return ts.readObjects
}

func (ts *testState) Arg() istructs.IObject {
	if ts.t != nil {
		ts.t.Helper()
	}

	if ts.queryObject != nil {
		return ts.queryObject
	}

	if ts.testData != nil && ts.testData[sys.Storage_CommandContext_Field_ArgumentObject] != nil {
		localPkgName := ts.appDef.PackageLocalName(ts.argumentType.PkgPath())
		localQName := appdef.NewQName(localPkgName, ts.argumentType.Entity())

		ob := ts.appStructs.ObjectBuilder(localQName)
		ob.FillFromJSON(ts.testData[sys.Storage_CommandContext_Field_ArgumentObject].(map[string]any))

		obj, err := ob.Build()
		require.NoError(ts.t, err)

		return obj
	}

	if ts.ipLogEvent == nil {
		panic("no current event")
	}

	return ts.ipLogEvent.ArgumentObject()
}

func (ts *testState) ResultBuilder() istructs.IObjectBuilder {
	if ts.ipLogEvent == nil {
		panic("no current event")
	}
	qname := ts.ipLogEvent.QName()
	command := appdef.Command(ts.appDef.Type, qname)
	if command == nil {
		panic(fmt.Sprintf("%v is not a command", qname))
	}
	return ts.appStructs.ObjectBuilder(command.Result().QName())
}

func (ts *testState) Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error) {
	if ts.httpHandler == nil {
		panic("http handler not set")
	}
	req := HTTPRequest{
		Timeout: timeout,
		Method:  method,
		URL:     url,
		Body:    body,
		Headers: headers,
	}
	resp, err := ts.httpHandler(req)
	if err != nil {
		return 0, nil, nil, err
	}
	return resp.Status, resp.Body, resp.Headers, nil
}

func (ts *testState) PutQuery(wsid istructs.WSID, name appdef.FullQName, argb QueryArgBuilderCallback) {
	ts.queryWsid = wsid
	ts.queryName = name

	if argb != nil {
		localPkgName := ts.appDef.PackageLocalName(ts.queryName.PkgPath())
		query := appdef.Query(ts.appDef.Type, appdef.NewQName(localPkgName, ts.queryName.Entity()))
		if query == nil {
			panic(fmt.Sprintf("query not found: %v", ts.queryName))
		}
		ab := ts.appStructs.ObjectBuilder(query.Param().QName())
		argb(ab)
		qo, err := ab.Build()
		if err != nil {
			panic(err)
		}
		ts.queryObject = qo
	}
}

func (ts *testState) PutRequestSubject(principals []iauthnz.Principal, token string) {
	ts.principals = principals
	ts.token = token
}

func (ts *testState) PutFederationCmdHandler(emu state.FederationCommandHandler) {
	ts.federationCmdHandler = emu
}

func (ts *testState) PutFederationBlobHandler(emu state.FederationBlobHandler) {
	ts.federationBlobHandler = emu
}

func (ts *testState) PutUniquesHandler(emu state.UniquesHandler) {
	ts.uniquesHandler = emu
}

func (ts *testState) PutEmailSender(emu state.IEmailSender) {
	ts.emailSender = emu
}

func (ts *testState) emulateUniquesHandler(entity appdef.QName, wsid istructs.WSID, data map[string]interface{}) (istructs.RecordID, error) {
	if ts.uniquesHandler == nil {
		panic("uniques handler not set")
	}
	return ts.uniquesHandler(entity, wsid, data)
}

func (ts *testState) emulateFederationCmd(owner, appname string, wsid istructs.WSID, command appdef.QName, body string) (statusCode int, newIDs map[string]istructs.RecordID, result string, err error) {
	if ts.federationCmdHandler == nil {
		panic("federation command handler not set")
	}
	return ts.federationCmdHandler(owner, appname, wsid, command, body)
}

func (ts *testState) emulateFederationBlob(owner, appname string, wsid istructs.WSID, ownerRecord appdef.QName, ownerRecordField appdef.FieldName,
	ownerID istructs.RecordID) ([]byte, error) {
	if ts.federationBlobHandler == nil {
		panic("federation blob handler not set")
	}
	return ts.federationBlobHandler(owner, appname, wsid, ownerRecord, ownerRecordField, ownerID)
}

func (ts *testState) buildState(processorKind int) {

	appFunc := func() istructs.IAppStructs { return ts.appStructs }
	eventFunc := func() istructs.IPLogEvent { return ts.ipLogEvent }
	partitionIDFunc := func() istructs.PartitionID { return TestPartition }
	cudFunc := func() istructs.ICUD { return ts.cud }
	commandPrepareArgs := func() istructs.CommandPrepareArgs {
		return istructs.CommandPrepareArgs{
			PrepareArgs: istructs.PrepareArgs{
				Workpiece:      nil,
				ArgumentObject: ts.Arg(),
				WSID:           ts.WSID(),
				Workspace:      nil,
			},
			ArgumentUnloggedObject: nil,
		}
	}
	argFunc := func() istructs.IObject { return ts.Arg() }
	unloggedArgFunc := func() istructs.IObject { return nil }
	wlogOffsetFunc := func() istructs.Offset {
		if ts.ipLogEvent != nil {
			return ts.ipLogEvent.WLogOffset()
		}

		return istructs.Offset(0)
	}
	originFunc := func() string {
		return ts.origin
	}
	wsidFunc := func() istructs.WSID {
		return ts.WSID()
	}
	resultBuilderFunc := func() istructs.IObjectBuilder {
		return ts.ResultBuilder()
	}
	principalsFunc := func() []iauthnz.Principal {
		return ts.principals
	}
	tokenFunc := func() string {
		return ts.token
	}
	execQueryArgsFunc := func() istructs.PrepareArgs {
		return istructs.PrepareArgs{
			Workpiece:      nil,
			ArgumentObject: ts.Arg(),
			WSID:           ts.WSID(),
			Workspace:      nil,
		}
	}
	qryResultBuilderFunc := func() istructs.IObjectBuilder {
		localPkgName := ts.appDef.PackageLocalName(ts.queryName.PkgPath())
		query := appdef.Query(ts.appDef.Type, appdef.NewQName(localPkgName, ts.queryName.Entity()))
		if query == nil {
			panic(fmt.Sprintf("query not found: %v", ts.queryName))
		}
		return ts.appStructs.ObjectBuilder(query.Result().QName())
	}
	execQueryCallback := func() istructs.ExecQueryCallback {
		return func(o istructs.IObject) error {
			ts.readObjects = append(ts.readObjects, o)
			return nil
		}
	}

	switch processorKind {
	case ProcKind_Actualizer:
		state := state.StateOpts{
			CustomHTTPClient:         ts,
			FederationCommandHandler: ts.emulateFederationCmd,
			UniquesHandler:           ts.emulateUniquesHandler,
			FederationBlobHandler:    ts.emulateFederationBlob,
		}
		ts.IState = stateprovide.ProvideAsyncActualizerStateFactory()(ts.ctx, appFunc, partitionIDFunc, wsidFunc, nil, ts.secretReader, eventFunc, nil, nil,
			IntentsLimit, BundlesLimit, state, ts.emailSender)
	case ProcKind_CommandProcessor:
		state := state.StateOpts{
			UniquesHandler: ts.emulateUniquesHandler,
		}
		ts.IState = stateprovide.ProvideCommandProcessorStateFactory()(ts.ctx, appFunc, partitionIDFunc, wsidFunc, ts.secretReader, cudFunc, principalsFunc, tokenFunc,
			IntentsLimit, resultBuilderFunc, commandPrepareArgs, argFunc, unloggedArgFunc, wlogOffsetFunc, state, originFunc)
	case ProcKind_QueryProcessor:
		state := state.StateOpts{
			CustomHTTPClient:         ts,
			FederationCommandHandler: ts.emulateFederationCmd,
			UniquesHandler:           ts.emulateUniquesHandler,
			FederationBlobHandler:    ts.emulateFederationBlob,
		}
		ts.IState = stateprovide.ProvideQueryProcessorStateFactory()(ts.ctx, appFunc, partitionIDFunc, wsidFunc, ts.secretReader, principalsFunc, tokenFunc, nil,
			execQueryArgsFunc, argFunc, qryResultBuilderFunc, nil, execQueryCallback, state)
	}
}

//go:embed testsys/*.sql
var fsTestSys embed.FS

func (ts *testState) buildAppDef(packagePath string, packageDir string, createWorkspaces ...TestWorkspace) {

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

	app, err := parser.FindApplication(pkgAst)
	if err != nil {
		panic(err)
	}

	packagesAST := []*parser.PackageSchemaAST{pkgAst, sysPackageAST}

	var dummyAppPkgAST *parser.PackageSchemaAST
	if app == nil {
		PackageName = "tstpkg"
		dummyAppFileAST, err := parser.ParseFile("dummy.sql", fmt.Sprintf(`
			IMPORT SCHEMA '%s' AS %s;
			APPLICATION test(
				USE %s;
			);
		`, packagePath, PackageName, PackageName))
		if err != nil {
			panic(err)
		}
		dummyAppPkgAST, err = parser.BuildPackageSchema(packagePath+"_app", []*parser.FileSchemaAST{dummyAppFileAST})
		if err != nil {
			panic(err)
		}
		packagesAST = append(packagesAST, dummyAppPkgAST)
	} else {
		PackageName = parser.ExtractLocalPackageName(packagePath)
	}

	appSchema, err := parser.BuildAppSchema(packagesAST)
	if err != nil {
		panic(err)
	}

	// TODO: obtain app name from packages
	// appName := appSchema.AppQName()

	appName := istructs.AppQName_test1_app1

	adb := builder.New()
	err = parser.BuildAppDefs(appSchema, adb)
	if err != nil {
		panic(err)
	}

	adf, err := adb.Build()
	if err != nil {
		panic(err)
	}

	ts.appDef = adf

	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(appName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	for ext := range appdef.Extensions(ts.appDef.Types()) {
		if ext.QName().Pkg() == PackageName {
			if proj, ok := ext.(appdef.IProjector); ok {
				if proj.Sync() {
					cfg.AddSyncProjectors(istructs.Projector{Name: ext.QName()})
				} else {
					cfg.AddAsyncProjectors(istructs.Projector{Name: ext.QName()})
				}
			} else if cmd, ok := ext.(appdef.ICommand); ok {
				cfg.Resources.Add(istructsmem.NewCommandFunction(cmd.QName(), istructsmem.NullCommandExec))
			} else if q, ok := ext.(appdef.IQuery); ok {
				cfg.Resources.Add(istructsmem.NewCommandFunction(q.QName(), istructsmem.NullCommandExec))
			}
		}
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
	ts.appStructs = structs
	ts.plogGen = istructsmem.NewIDGenerator()
	ts.wsOffsets = make(map[istructs.WSID]istructs.Offset)

	for _, ws := range createWorkspaces {
		err = wsdescutil.CreateCDocWorkspaceDescriptorStub(ts.appStructs, TestPartition, ws.WSID, appdef.NewQName(PackageName, ws.WorkspaceDescriptor), ts.nextPLogOffs(), ts.nextWSOffs(ws.WSID))
		if err != nil {
			panic(err)
		}
	}
}

func (ts *testState) nextPLogOffs() istructs.Offset {
	ts.plogOffset += 1
	return ts.plogOffset
}

func (ts *testState) nextWSOffs(ws istructs.WSID) istructs.Offset {
	offs, ok := ts.wsOffsets[ws]
	if !ok {
		offs = istructs.Offset(0)
	}
	offs += 1
	ts.wsOffsets[ws] = offs
	return offs
}

func (ts *testState) PutHTTPHandler(handler HTTPHandlerFunc) {
	ts.httpHandler = handler
}

func (ts *testState) PutRecords(wsid istructs.WSID, cb NewRecordsCallback) (wLogOffs istructs.Offset, newRecordIds []istructs.RecordID) {
	return ts.PutEvent(wsid, appdef.NewFullQName(istructs.QNameCommandCUD.Pkg(), istructs.QNameCommandCUD.Entity()), func(argBuilder istructs.IObjectBuilder, cudBuilder istructs.ICUD) {
		cb(cudBuilder)
	})
}

func (ts *testState) GetRecord(wsid istructs.WSID, id istructs.RecordID) istructs.IRecord {
	var rec istructs.IRecord
	rec, err := ts.appStructs.Records().Get(wsid, false, id)
	if err != nil {
		panic(err)
	}
	return rec
}

func (ts *testState) PutEvent(wsid istructs.WSID, name appdef.FullQName, cb NewEventCallback) (wLogOffs istructs.Offset, newRecordIds []istructs.RecordID) {
	var localPkgName string
	if name.PkgPath() == appdef.SysPackage {
		localPkgName = name.PkgPath()
	} else {
		localPkgName = ts.appDef.PackageLocalName(name.PkgPath())
	}

	wLogOffs = ts.nextWSOffs(wsid)
	reb := ts.appStructs.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: TestPartition,
			QName:             appdef.NewQName(localPkgName, name.Entity()),
			WLogOffset:        wLogOffs,
			PLogOffset:        ts.nextPLogOffs(),
		},
	})

	if cb != nil {
		ts.cud = reb.CUDBuilder()
		cb(reb.ArgumentObjectBuilder(), ts.cud)
	}

	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		panic(err)
	}

	ipLogEvent, err := ts.appStructs.Events().PutPlog(rawEvent, nil, ts.plogGen)
	if err != nil {
		panic(err)
	}

	err = ts.appStructs.Events().PutWlog(ipLogEvent)
	if err != nil {
		panic(err)
	}

	newRecordIds = make([]istructs.RecordID, 0)
	err = ts.appStructs.Records().Apply2(ipLogEvent, func(r istructs.IRecord) {
		newRecordIds = append(newRecordIds, r.ID())
	})

	if err != nil {
		panic(err)
	}

	ts.ipLogEvent = ipLogEvent

	return wLogOffs, newRecordIds
}

// nolint unusedwrite
func (ts *testState) PutView(wsid istructs.WSID, entity appdef.FullQName, callback ViewValueCallback) {
	localPkgName := ts.appDef.PackageLocalName(entity.PkgPath())
	v := TestViewValue{
		wsid: wsid,
		vr:   ts.appStructs.ViewRecords(),
		Key:  ts.appStructs.ViewRecords().KeyBuilder(appdef.NewQName(localPkgName, entity.Entity())),
		Val:  ts.appStructs.ViewRecords().NewValueBuilder(appdef.NewQName(localPkgName, entity.Entity())),
	}
	callback(v.Key, v.Val)
	err := ts.appStructs.ViewRecords().Put(wsid, v.Key, v.Val)
	if err != nil {
		panic(err)
	}
}

func (ts *testState) PutSecret(name string, secret []byte) {
	ts.secretReader.(*secretReader).secrets[name] = secret
}

type intentAssertions struct {
	t   *testing.T
	kb  istructs.IStateKeyBuilder
	vb  istructs.IStateValueBuilder
	ctx *testState
}

func (ia *intentAssertions) NotExists() {
	if ia.vb != nil {
		require.Fail(ia.t, "expected intent not to exist")
	}
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

	vb, err := ia.ctx.NewValue(ia.kb)
	if err != nil {
		panic(err)
	}
	vbc(vb)

	if !ia.vb.Equal(vb) {
		require.Fail(ia.t, "expected intents to be equal")
	}
}

func (ts *testState) RequireNoIntents(t *testing.T) {
	if ts.IntentsCount() > 0 {
		require.Fail(t, "expected no intents")
	}
}

func (ts *testState) RequireIntent(t *testing.T, storage appdef.QName, entity appdef.FullQName, kbc KeyBuilderCallback) IIntentAssertions {
	localPkgName := ts.appDef.PackageLocalName(entity.PkgPath())
	localEntity := appdef.NewQName(localPkgName, entity.Entity())
	kb, err := ts.KeyBuilder(storage, localEntity)
	if err != nil {
		panic(err)
	}
	kbc(kb)
	return &intentAssertions{
		t:   t,
		kb:  kb,
		vb:  ts.IState.FindIntent(kb),
		ctx: ts,
	}
}
