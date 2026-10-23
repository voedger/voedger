/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/wneessen/go-mail"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/verifier"
	vvmpkg "github.com/voedger/voedger/pkg/vvm"
)

// shared among all VIT instances
// caches schemas for apps that are not test apps (i.e., not test1_app1, test1_app2, test2_app1, or test2_app2)
var nonTestAppsSchemasCache = &implISchemasCache_nonTestApps{schemas: map[appdef.AppQName]*parser.AppSchemaAST{}}

func NewVIT(tb testing.TB, vitCfg *VITConfig, opts ...IVITOpt) (vit *VIT) {
	tb.Helper()
	require.NotNil(tb, vitCfg)
	if !vitCfg.isShared {
		return initializeVIT(tb, newVit(tb, vitCfg, coreutils.IsCassandraStorage()), opts...)
	}

	// NewVIT calls must be sequential, including for different configurations.
	// Reserve the identity before construction.
	vit, exists := vits[vitCfg]
	if !exists {
		vit = newVITStruct(tb, vitCfg, coreutils.IsCassandraStorage())
		vits[vitCfg] = vit
	}
	closed := vit.serverClosed
	busy := exists && !vit.isFinalized
	if !closed && !busy {
		vit.isFinalized = false
		vit.T = tb
	}
	if closed {
		tb.Fatal("cannot acquire VIT: server is closed")
		return nil
	}
	if busy {
		panic("TearDown() was not called on a previous VIT which used the provided shared config")
	}
	if !exists {
		vit.startVVMAndCreateEntities()
	}
	return initializeVIT(tb, vit, opts...)
}

func initializeVIT(tb testing.TB, vit *VIT, opts ...IVITOpt) *VIT {
	tb.Helper()
	for _, opt := range opts {
		opt.applyVITOpt(vit)
	}

	vit.T = tb
	vit.emailCaptor.checkEmpty(tb)
	vit.currentAfterTest = slices.Clone(vit.configAfterTest)

	// run each test in the next day to mostly prevent previous tests impact and\or workspace initialization
	vit.TimeAdd(day)

	vit.initialGoroutinesNum = runtime.NumGoroutine()

	return vit
}

func newVit(tb testing.TB, vitCfg *VITConfig, useCas bool) *VIT {
	tb.Helper()
	vit := newVITStruct(tb, vitCfg, useCas)
	vit.startVVMAndCreateEntities()
	return vit
}

func newVITStruct(tb testing.TB, cfg *VITConfig, useCas bool) *VIT {
	return &VIT{
		T:                tb,
		effectiveConfig:  cfg,
		isOnSharedConfig: cfg.isShared,
		useCassandra:     useCas,
		mockTime:         testingu.MockTime,
		appWorkspaces:    map[appdef.AppQName]map[string]*AppWorkspace{},
		principals:       map[appdef.AppQName]map[string]*Principal{},
	}
}

func (vit *VIT) startVVMAndCreateEntities() {
	vit.T.Helper()
	defer logger.SetLogLevelWithRestore(logger.LogLevelWarning)()
	completed := false
	defer vit.cleanupOnFailure(&completed)
	preConfig, err := vit.startVVM(vit.effectiveConfig)
	require.NoError(vit.T, err)
	vit.currentAfterTest = slices.Clone(vit.configAfterTest)
	vit.createEntities(preConfig)
	completed = true
}

// startVVM constructs and launches the server directly on VIT. Its callers
// install deferred cleanup before entering, including for panic and Goexit.
func (vit *VIT) startVVM(vitCfg *VITConfig) (preConfig *vitPreConfig, err error) {
	cfg := vvmpkg.NewVVMDefaultConfig()

	// only dynamic ports are used in tests
	cfg.VVMPort = 0
	cfg.MetricsServicePort = 0
	cfg.AdminPort = 0

	cfg.SequencesTrustLevel = istructs.SequencesTrustLevel_0

	cfg.Time = testingu.MockTime
	retainedTime := cfg.Time
	if vit.VVMConfig != nil {
		retainedTime = vit.Time
	}
	cfg.SchemasCache = nonTestAppsSchemasCache
	cfg.BusyProcessorLogMode = vvmpkg.BusyProcessorLogMode_Silent

	emailCaptor := &implIEmailSender_captor{
		emailCaptorCh: make(chan state.EmailMessage, 1), // must be buffered
	}
	cfg.EmailSender = emailCaptor
	vit.VoedgerVM = nil
	vit.VVMConfig = &cfg
	vit.emailCaptor = emailCaptor
	vit.httpClient, vit.currentTestCleanups = nil, nil
	vit.configAfterTest = nil
	vit.vvmProblemCtx = nil
	vit.serverClosed = false

	cfg.KeyspaceIsolationSuffix = provider.NewTestKeyspaceIsolationSuffix()

	preConfig = &vitPreConfig{
		vvmCfg:  &cfg,
		vitApps: vitApps{},
		secrets: map[string][]byte{},
	}

	if vit.useCassandra {
		cfg.StorageFactory = func(timeu.ITime) (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragecas ", fmt.Sprint(cas.DefaultCasParams))
			return cas.Provide(cas.DefaultCasParams)
		}
	}

	for _, opt := range vitCfg.opts {
		opt(preConfig)
	}

	for _, initFunc := range preConfig.initFuncs {
		initFunc()
	}
	vit.configAfterTest = slices.Clone(preConfig.afterTest)

	if vit.storageFactory == nil {
		factory := cfg.StorageFactory
		cfg.StorageFactory = func(now timeu.ITime) (istorage.IAppStorageFactory, error) {
			storage, err := factory(now)
			if err == nil {
				vit.storageFactory = storage
				vit.storageSuffix = cfg.KeyspaceIsolationSuffix
			}
			return storage, err
		}
	} else {
		cfg.StorageFactory = func(timeu.ITime) (istorage.IAppStorageFactory, error) { return vit.storageFactory, nil }
		cfg.KeyspaceIsolationSuffix = vit.storageSuffix
		cfg.Time = retainedTime
	}
	cfg.SecretsReader = &implVITISecretsReader{secrets: preConfig.secrets, underlyingReader: itokensjwt.ProvideTestSecretsReader(cfg.SecretsReader)}

	// eliminate timeouts impact for debugging
	cfg.RouterReadTimeout = int(debugTimeout)
	cfg.RouterWriteTimeout = int(debugTimeout)

	// append retry on WSAECONNREFUSED to in-VVM IFederationWithRetry
	// Otherwise stress tests on Windows are failed due of WSAECONNREFUSED on workspace post-init
	policyOptsForWithRetry := slices.Clone(cfg.PolicyOptsForFederationWithRetry)
	policyOptsForWithRetry = append(policyOptsForWithRetry, withRetryOnConnRefused)
	cfg.PolicyOptsForFederationWithRetry = policyOptsForWithRetry

	vit.VoedgerVM, err = vvmpkg.Provide(&cfg)
	if err != nil {
		return nil, fmt.Errorf("construct VVM: %w", err)
	}

	// register workspace templates
	for _, app := range preConfig.vitApps {
		ep := vit.AppsExtensionPoints[app.name]
		for _, tf := range app.wsTemplateFuncs {
			tf(ep)
		}
	}

	vit.httpClient, _ = httpu.NewIHTTPClient(httpu.WithRetryPolicy(vitHTTPClientRetryPolicy...))

	// get rid of huge amount of logs reporting about workspaces init process
	defer logger.SetLogLevelWithRestore(logger.LogLevelWarning)()

	// launch the server
	// leadership duration - ten years to avoid leadership expiration when time bumps in tests (including 1 day add on each test)
	vit.vvmProblemCtx = vit.Launch(10*365*24*60*60, vvmpkg.DefaultLeadershipAcquisitionDuration)
	select {
	case <-vit.vvmProblemCtx.Done():
		err = vit.shutdownVVM()
		if err == nil {
			err = errors.New("VVM problem context closed during launch")
		}
		return nil, fmt.Errorf("launch VVM: %w", err)
	default:
	}
	return preConfig, nil
}

func (vit *VIT) createEntities(vitPreConfig *vitPreConfig) {
	vit.T.Helper()
	tb := vit.T
	for _, app := range vitPreConfig.vitApps {
		// generate verified value tokens if queried
		//                desiredValue token
		verifiedValues := map[string]string{}
		for desiredValue, vvi := range app.verifiedValuesIntents {
			appTokens := vit.IAppTokensFactory.New(app.name)
			verifiedValuePayload := payloads.VerifiedValuePayload{
				VerificationKind: appdef.VerificationKind_EMail,
				Entity:           vvi.docQName,
				Field:            vvi.fieldName,
				Value:            vvi.desiredValue,
			}
			verifiedValueToken, err := appTokens.IssueToken(verifier.VerifiedValueTokenDuration, &verifiedValuePayload)
			require.NoError(vit.T, err)
			verifiedValues[desiredValue] = verifiedValueToken
		}

		// create logins and workspaces
		for _, login := range app.logins {
			vit.SignUp(login.Name, login.Pwd, login.AppQName,
				WithReqOpt(httpu.WithExpectedCode(http.StatusCreated)),
				WithReqOpt(httpu.WithExpectedCode(http.StatusConflict)),
			)
			prn := vit.SignIn(login)
			appPrincipals, ok := vit.principals[app.name]
			if !ok {
				appPrincipals = map[string]*Principal{}
				vit.principals[app.name] = appPrincipals
			}
			appPrincipals[login.Name] = prn

			createSubjects(vit, prn.Token, login.subjects, login.AppQName, prn.ProfileWSID)

			for doc, dataFactory := range login.docs {
				if !vit.PostProfile(prn, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":%q}}`, doc)).IsEmpty() {
					continue
				}
				data := dataFactory(verifiedValues)
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = doc.String()

				bb, err := json.Marshal(data)
				require.NoError(tb, err)

				vit.PostProfile(prn, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}

		sysToken, err := payloads.GetSystemPrincipalToken(vit, app.name)
		require.NoError(vit.T, err)
		for _, wsd := range app.ws {
			owner := vit.principals[app.name][wsd.ownerLoginName]
			appWorkspaces, ok := vit.appWorkspaces[app.name]
			if !ok {
				appWorkspaces = map[string]*AppWorkspace{}
				vit.appWorkspaces[app.name] = appWorkspaces
			}
			newAppWS := vit.CreateWorkspace(wsd, owner, httpu.WithExpectedCode(http.StatusOK), httpu.WithExpectedCode(http.StatusConflict))
			newAppWS.childs = wsd.childs
			newAppWS.docs = wsd.docs
			newAppWS.subjects = wsd.subjects
			appWorkspaces[wsd.Name] = newAppWS

			handleWSParam(vit, appWorkspaces[wsd.Name], appWorkspaces, verifiedValues, sysToken)
		}
	}
	if vitPreConfig.postInitFunc != nil {
		vitPreConfig.postInitFunc(vit)
	}
	vit.checkVVMProblemCtx()
}

func handleWSParam(vit *VIT, appWS *AppWorkspace, appWorkspaces map[string]*AppWorkspace, verifiedValues map[string]string, token string) {
	for doc, dataFactory := range appWS.docs {
		if !vit.PostWS(appWS, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":%q}}`, doc), httpu.WithAuthorizeBy(token)).IsEmpty() {
			continue
		}
		data := dataFactory(verifiedValues)
		data[appdef.SystemField_ID] = 1
		data[appdef.SystemField_QName] = doc.String()

		bb, err := json.Marshal(data)
		require.NoError(vit.T, err)

		vit.PostWS(appWS, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb), httpu.WithAuthorizeBy(token))
	}

	createSubjects(vit, token, appWS.subjects, appWS.AppQName(), appWS.WSID)

	for _, childWSParams := range appWS.childs {
		vit.InitChildWorkspace(childWSParams, appWS)
		childAppWS := vit.WaitForChildWorkspace(appWS, childWSParams.Name)
		require.Empty(vit.T, childAppWS.WSError)
		childAppWS.childs = childWSParams.childs
		childAppWS.subjects = childWSParams.subjects
		childAppWS.docs = childWSParams.docs
		childAppWS.ownerLoginName = childWSParams.ownerLoginName
		childAppWS.Owner = vit.GetPrincipal(appWS.AppQName(), childWSParams.ownerLoginName)
		appWorkspaces[childWSParams.Name] = childAppWS
		handleWSParam(vit, childAppWS, appWorkspaces, verifiedValues, token)
	}
}

func createSubjects(vit *VIT, token string, subjects []subject, appQName appdef.AppQName, wsid istructs.WSID) {
	for _, subject := range subjects {
		var roles strings.Builder
		for i, role := range subject.roles {
			if i > 0 {
				roles.WriteString(",")
			}
			roles.WriteString(role.String())
		}
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":%q,"Roles":%q,"SubjectKind":%d,"ProfileWSID":%d}}]}`,
			subject.login, roles.String(), subject.subjectKind, vit.principals[appQName][subject.login].ProfileWSID)
		vit.PostApp(appQName, wsid, "c.sys.CUD", body, httpu.WithAuthorizeBy(token))
	}
}

func NewVITLocalCassandra(tb testing.TB, vitCfg *VITConfig, opts ...IVITOpt) (vit *VIT) {
	tb.Helper()
	vit = newVit(tb, vitCfg, true)
	for _, opt := range opts {
		opt.applyVITOpt(vit)
	}

	return vit
}

func (vit *VIT) WS(appQName appdef.AppQName, wsName string) *AppWorkspace {
	appWorkspaces, ok := vit.appWorkspaces[appQName]
	if !ok {
		panic("unknown app " + appQName.String())
	}
	if ws, ok := appWorkspaces[wsName]; ok {
		return ws
	}
	panic("unknown workspace " + wsName)
}

func (vit *VIT) TearDown() {
	vit.T.Helper()
	if vit.isFinalized {
		return
	}
	defer func() { vit.isFinalized = true }()
	defer func() {
		grNum := runtime.NumGoroutine()
		if grNum-vit.initialGoroutinesNum > allowedGoroutinesNumDiff {
			vit.T.Logf("!!! goroutines leak: was %d on VIT setup, now %d after teardown", vit.initialGoroutinesNum, grNum)
		}
	}()
	vit.T.Log("goroutines num before cleanup:", runtime.NumGoroutine())
	if vit.isOnSharedConfig {
		vit.cleanupSharedVIT()
	} else {
		vit.cleanupOwnVIT()
	}
}

func (vit *VIT) MetricsServicePort() int {
	return int(vit.VoedgerVM.MetricsServicePort())
}

func (vit *VIT) GetSystemPrincipal(appQName appdef.AppQName) *Principal {
	vit.T.Helper()
	vit.requireRunning()
	vit.lock.Lock()
	defer vit.lock.Unlock()
	appPrincipals, ok := vit.principals[appQName]
	if !ok {
		appPrincipals = map[string]*Principal{}
		vit.principals[appQName] = appPrincipals
	}
	prn, ok := appPrincipals["___sys"]
	if !ok {
		as, err := vit.BuiltIn(appQName)
		require.NoError(vit.T, err)
		sysToken, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
		require.NoError(vit.T, err)
		prn = &Principal{
			Token:       sysToken,
			ProfileWSID: istructs.NullWSID,
			Name:        "___sys",
			AppQName:    appQName,
			subjectKind: istructs.SubjectKind_User,
		}
		appPrincipals["___sys"] = prn
	}
	return prn
}

func (vit *VIT) GetPrincipal(appQName appdef.AppQName, login string) *Principal {
	vit.T.Helper()
	appPrincipals, ok := vit.principals[appQName]
	if !ok {
		vit.T.Fatalf("unknown app %s", appQName)
	}
	prn, ok := appPrincipals[login]
	if !ok {
		vit.T.Fatalf("unknown login %s", login)
	}
	return prn
}

func (vit *VIT) PostProfile(prn *Principal, funcName string, body string, opts ...httpu.ReqOptFunc) *federation.FuncResponse {
	vit.T.Helper()
	opts = append(opts, httpu.WithDefaultAuthorize(prn.Token))
	return vit.PostApp(prn.AppQName, prn.ProfileWSID, funcName, body, opts...)
}

func (vit *VIT) PostWS(ws *AppWorkspace, funcName string, body string, opts ...httpu.ReqOptFunc) *federation.FuncResponse {
	vit.T.Helper()
	opts = append(opts, httpu.WithDefaultAuthorize(ws.Owner.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

// PostWSSys is PostWS authorized by the System Token
func (vit *VIT) PostWSSys(ws *AppWorkspace, funcName string, body string, opts ...httpu.ReqOptFunc) *federation.FuncResponse {
	vit.T.Helper()
	sysPrn := vit.GetSystemPrincipal(ws.Owner.AppQName)
	opts = append(opts, httpu.WithDefaultAuthorize(sysPrn.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

func (vit *VIT) UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, name string, contentType string, content []byte,
	ownerRecord appdef.QName, ownerRecordField appdef.FieldName, opts ...httpu.ReqOptFunc) (blobID istructs.RecordID) {
	vit.T.Helper()
	vit.requireRunning()
	blobReader := iblobstorage.BLOBReader{
		Name:             name,
		ContentType:      contentType,
		OwnerRecord:      ownerRecord,
		OwnerRecordField: ownerRecordField,
		ReadCloser:       io.NopCloser(bytes.NewReader(content)),
	}
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithRetryPolicy(vitHTTPClientRetryPolicy...)}
	o = append(o, opts...)
	blobID, err := vit.IFederation.UploadBLOB(appQName, wsid, blobReader, o...)
	require.NoError(vit.T, err)
	return blobID
}

func (vit *VIT) SQLQueryRows(ws *AppWorkspace, sqlQuery string, fmtArgs ...any) []map[string]interface{} {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Query":%q},"elements":[{"fields":["Result"]}]}`, fmt.Sprintf(sqlQuery, fmtArgs...))
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body, httpu.WithAuthorizeBy(ws.Owner.Token))
	res := []map[string]interface{}{}
	for _, elem := range resp.Sections[0].Elements {
		m := map[string]interface{}{}
		require.NoError(vit.T, json.Unmarshal([]byte(elem[0][0][0].(string)), &m))
		res = append(res, m)
	}
	return res
}

func (vit *VIT) SQLQuery(ws *AppWorkspace, sqlQuery string, fmtArgs ...any) map[string]interface{} {
	vit.T.Helper()
	return vit.SQLQueryRows(ws, sqlQuery, fmtArgs...)[0]
}

func (vit *VIT) UploadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, name string, contentType string, content []byte, duration iblobstorage.DurationType,
	opts ...httpu.ReqOptFunc) (blobSUUID iblobstorage.SUUID) {
	vit.T.Helper()
	vit.requireRunning()
	blobReader := iblobstorage.BLOBReader{
		Name:        name,
		ContentType: contentType,
		ReadCloser:  io.NopCloser(bytes.NewReader(content)),
	}
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithRetryPolicy(vitHTTPClientRetryPolicy...)}
	o = append(o, opts...)
	blobSUUID, err := vit.IFederation.UploadTempBLOB(appQName, wsid, blobReader, duration, o...)
	require.NoError(vit.T, err)
	return blobSUUID
}

func (vit *VIT) Func(url string, body string, opts ...httpu.ReqOptFunc) *federation.FuncResponse {
	vit.T.Helper()
	vit.requireRunning()
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), httpu.WithDefaultMethod(http.MethodPost)}
	o = append(o, opts...)
	httpResp, err := vit.httpClient.Req(context.Background(), vit.URLStr()+"/"+url, body, o...)
	funcResp, err := federation.HTTPRespToFuncResp(httpResp, err)
	require.NoError(vit.T, err)
	vit.satisfySysErrorExpectations(funcResp, httpResp.Opts)
	return funcResp
}

// blob ReadCloser must be read out by the test
// will be closed by the VIT
func (vit *VIT) ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, ownerRecord appdef.QName, ownerRecordField appdef.FieldName, ownerID istructs.RecordID,
	optFuncs ...httpu.ReqOptFunc) iblobstorage.BLOBReader {
	vit.T.Helper()
	vit.requireRunning()
	o := []httpu.ReqOptFunc{createVITOpts()}
	o = append(o, optFuncs...)
	reader, err := vit.IFederation.ReadBLOB(appQName, wsid, ownerRecord, ownerRecordField, ownerID, o...)
	require.NoError(vit.T, err)
	vit.trackBLOBReader(reader)
	return reader
}

func (vit *VIT) trackBLOBReader(reader iblobstorage.BLOBReader) {
	vit.lock.Lock()
	defer vit.lock.Unlock()
	vit.currentTestCleanups = append(vit.currentTestCleanups, func() { vit.closeBLOBReader(reader) })
}

// blob ReadCloser must be read out by the test
// will be closed by the VIT
func (vit *VIT) ReadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobSUUID iblobstorage.SUUID, optFuncs ...httpu.ReqOptFunc) iblobstorage.BLOBReader {
	vit.T.Helper()
	vit.requireRunning()
	blobReader, err := vit.IFederation.ReadTempBLOB(appQName, wsid, blobSUUID, optFuncs...)
	require.NoError(vit.T, err)
	vit.trackBLOBReader(blobReader)
	return blobReader
}

func (vit *VIT) GET(relativeURL string, opts ...httpu.ReqOptFunc) *httpu.HTTPResponse {
	vit.T.Helper()
	vit.requireRunning()
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), httpu.WithDefaultMethod(http.MethodGet)}
	o = append(o, opts...)
	url := vit.URLStr() + "/" + relativeURL
	res, err := vit.httpClient.Req(context.Background(), url, "", o...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) POST(relativeURL string, body string, opts ...httpu.ReqOptFunc) *httpu.HTTPResponse {
	vit.T.Helper()
	vit.requireRunning()
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), httpu.WithDefaultMethod(http.MethodPost)}
	o = append(o, opts...)
	url := vit.URLStr() + "/" + relativeURL
	httpResp, err := vit.httpClient.Req(context.Background(), url, body, o...)
	require.NoError(vit.T, err)
	vit.checkExpectationsInHTTPResp(httpResp)
	return httpResp
}

func (vit *VIT) checkExpectationsInHTTPResp(httpResp *httpu.HTTPResponse) {
	if len(httpResp.Opts.CustomOpts(vitOptsKey{}).(*vitReqOpts).expectedMessages) == 0 {
		return
	}
	vit.T.Helper()
	var funcResponse *federation.FuncResponse
	require.NoError(vit.T, json.Unmarshal([]byte(httpResp.Body), &funcResponse))
	vit.satisfySysErrorExpectations(funcResponse, httpResp.Opts)
}

func (vit *VIT) satisfySysErrorExpectations(funcResp *federation.FuncResponse, opts httpu.IReqOpts) {
	vitOpts := opts.CustomOpts(vitOptsKey{}).(*vitReqOpts)
	if len(vitOpts.expectedMessages) == 0 {
		return
	}
	if funcResp == nil || funcResp.SysError == nil {
		vit.T.Fatal("expected error messages", vitOpts.expectedMessages, "but no response or no error in response")
	}
	var sysError coreutils.SysError
	if !errors.As(funcResp.SysError, &sysError) {
		require.NoError(vit.T, funcResp.SysError)
	}
	index := 0
	for _, expectedMes := range vitOpts.expectedMessages {
		require.Containsf(vit.T, sysError.Message[index:], expectedMes, "actual message %q, ordered expected %#v", sysError.Message, vitOpts.expectedMessages)
		index = strings.Index(sysError.Message[index:], expectedMes) + len(expectedMes)
	}
}

func (vit *VIT) PostApp(appQName appdef.AppQName, wsid istructs.WSID, funcName string, body string, opts ...httpu.ReqOptFunc) *federation.FuncResponse {
	vit.T.Helper()
	vit.requireRunning()
	url := fmt.Sprintf("%s/api/%s/%d/%s", vit.URLStr(), appQName, wsid, funcName)
	o := []httpu.ReqOptFunc{createVITOpts(), httpu.WithOptsValidator(httpu.DenyGETAndDiscardResponse), httpu.WithDefaultMethod(http.MethodPost)}
	o = append(o, opts...)
	httpResp, err := vit.httpClient.Req(context.Background(), url, body, o...)
	funcResp, err := federation.HTTPRespToFuncResp(httpResp, err)
	require.NoError(vit.T, err)
	vit.satisfySysErrorExpectations(funcResp, httpResp.Opts)
	return funcResp
}

func (vit *VIT) WaitFor(consumer func() *federation.FuncResponse) *federation.FuncResponse {
	vit.T.Helper()
	start := time.Now()
	for time.Since(start) < testTimeout {
		resp := consumer()
		if len(resp.Sections) > 0 {
			return resp
		}
		logger.Info("waiting for projection")
		time.Sleep(100 * time.Millisecond)
	}
	vit.T.Fail()
	return nil
}

func (vit *VIT) RefreshTokens() {
	vit.T.Helper()
	vit.requireRunning()
	for _, appPrns := range vit.principals {
		for _, prn := range appPrns {
			// issue principal token
			principalPayload := payloads.PrincipalPayload{
				Login:       prn.Name,
				SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: prn.ProfileWSID,
			}
			as, err := vit.BuiltIn(prn.AppQName)
			require.NoError(vit.T, err) // notest
			newToken, err := as.AppTokens().IssueToken(authnz.DefaultPrincipalTokenExpiration, &principalPayload)
			require.NoError(vit.T, err)
			prn.Token = newToken
		}
	}
}

func (vit *VIT) NextNumber() int {
	vit.lock.Lock()
	vit.nextNumber++
	res := vit.nextNumber
	vit.lock.Unlock()
	return res
}

func (vit *VIT) Now() time.Time {
	return vit.Time.Now()
}

func (vit *VIT) TimeAdd(dur time.Duration) {
	vit.mockTime.Add(dur)
	vit.RefreshTokens()
}

// SchedulerTimeAdd advances the scheduler's isolated time by the given duration.
// This is used to control job scheduler timing independently from global MockTime.
// Jobs will only fire when this isolated time is advanced past their scheduled time.
func (vit *VIT) SchedulerTimeAdd(dur time.Duration) {
	vit.ISchedulerRunner.SchedulersTime().(testingu.IMockTime).Add(dur)
}

// SchedulerNow returns the current scheduler's isolated time.
// This is the time used by job schedulers, which may differ from global MockTime.
func (vit *VIT) SchedulerNow() time.Time {
	return vit.ISchedulerRunner.SchedulersTime().Now()
}

func (vit *VIT) NextName() string {
	return "name_" + strconv.Itoa(vit.NextNumber())
}

// CaptureEmail waits for and returns the next sent email
// no emails during testEmailsAwaitingTimeout -> test failed
// an email was sent but CaptureEmail is not called -> test will be failed on VIT.TearDown()
func (vit *VIT) CaptureEmail() (msg state.EmailMessage) {
	vit.T.Helper()
	vit.requireRunning()
	tmr := time.NewTimer(getTestEmailsAwaitingTimeout())
	defer tmr.Stop()
	select {
	case msg = <-vit.emailCaptor.emailCaptorCh:
		return msg
	case <-tmr.C:
		vit.T.Fatal("no email messages")
	}
	return msg
}

// sets delay on IAppStorage.Get() in mem implementation
// will be automatically reset to 0 on RestartVVM or TearDown
// need to e.g. investigate slow workspace create, see https://github.com/voedger/voedger/issues/1663
func (vit *VIT) SetMemStorageGetDelay(delay time.Duration) {
	vit.T.Helper()
	vit.requireRunning()
	vit.iterateDelaySetters(func(delaySetter istorage.IStorageDelaySetter) {
		delaySetter.SetTestDelayGet(delay)
		vit.lock.Lock()
		vit.currentTestCleanups = append(vit.currentTestCleanups, func() { delaySetter.SetTestDelayGet(0) })
		vit.lock.Unlock()
	})
}

// sets delay on IAppStorage.Put() in mem implementation
// will be automatically reset to 0 on RestartVVM or TearDown
// need to e.g. investigate slow workspace create, see https://github.com/voedger/voedger/issues/1663
func (vit *VIT) SetMemStoragePutDelay(delay time.Duration) {
	vit.T.Helper()
	vit.requireRunning()
	vit.iterateDelaySetters(func(delaySetter istorage.IStorageDelaySetter) {
		delaySetter.SetTestDelayPut(delay)
		vit.lock.Lock()
		vit.currentTestCleanups = append(vit.currentTestCleanups, func() { delaySetter.SetTestDelayPut(0) })
		vit.lock.Unlock()
	})
}

func (vit *VIT) iterateDelaySetters(cb func(delaySetter istorage.IStorageDelaySetter)) {
	vit.T.Helper()
	for anyAppQName := range vit.VVMAppsBuilder {
		as, err := vit.AppStorage(anyAppQName)
		require.NoError(vit.T, err)
		delaySetter, ok := as.(istorage.IStorageDelaySetter)
		if !ok {
			vit.T.Fatal("IAppStorage implementation is not in-mem")
		}
		cb(delaySetter)
	}
}

func (sr *implVITISecretsReader) ReadSecret(name string) ([]byte, error) {
	if val, ok := sr.secrets[name]; ok {
		return val, nil
	}
	return sr.underlyingReader.ReadSecret(name)
}

func (vit *VIT) EnrichPrincipalToken(prn *Principal, roles []payloads.RoleType) (enrichedToken string) {
	vit.T.Helper()
	var pp payloads.PrincipalPayload
	_, err := vit.ValidateToken(prn.Token, &pp)
	require.NoError(vit.T, err)
	pp.Roles = append(pp.Roles, roles...)
	enrichedToken, err = vit.ITokens.IssueToken(prn.AppQName, authnz.DefaultPrincipalTokenExpiration, &pp)
	require.NoError(vit.T, err)
	return enrichedToken
}

func (vit *VIT) checkVVMProblemCtx() {
	vit.T.Helper()
	select {
	case <-vit.vvmProblemCtx.Done():
		err := vit.shutdownVVM()
		if err == nil {
			err = errors.New("VVM problem context is closed without a shutdown error")
		}
		require.NoError(vit.T, err)
	default:
	}
}

func (c *implIEmailSender_captor) Send(_ string, msg state.EmailMessage, _ ...mail.Option) error {
	c.emailCaptorCh <- msg
	return nil
}

func (c *implIEmailSender_captor) checkEmpty(tb testing.TB) {
	tb.Helper()
	select {
	case _, ok := <-c.emailCaptorCh:
		if ok {
			tb.Log("unexpected email message received")
			tb.Fail()
		}
	default:
	}
}

func (c *implIEmailSender_captor) shutDown() {
	close(c.emailCaptorCh)
}
