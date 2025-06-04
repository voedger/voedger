/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/isequencer"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/cas"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/verifier"
	vvmpkg "github.com/voedger/voedger/pkg/vvm"
)

func NewVIT(t testing.TB, vitCfg *VITConfig, opts ...vitOptFunc) (vit *VIT) {
	useCas := coreutils.IsCassandraStorage()
	if !vitCfg.isShared {
		vit = newVit(t, vitCfg, useCas, false)
	} else {
		ok := false
		if vit, ok = vits[vitCfg]; ok {
			if !vit.isFinalized {
				panic("Teardown() was not called on a previous VIT which used the provided shared config")
			}
			vit.isFinalized = false
		} else {
			vit = newVit(t, vitCfg, useCas, false)
			vits[vitCfg] = vit
		}
	}

	for _, opt := range opts {
		opt(vit)
	}

	vit.emailCaptor.checkEmpty(t)

	vit.T = t

	// run each test in the next day to mostly prevent previous tests impact and\or workspace initialization
	vit.TimeAdd(day)

	vit.initialGoroutinesNum = runtime.NumGoroutine()

	return vit
}

func newVit(t testing.TB, vitCfg *VITConfig, useCas bool, vvmLaunchOnly bool) *VIT {
	cfg := vvmpkg.NewVVMDefaultConfig()

	// only dynamic ports are used in tests
	cfg.VVMPort = 0
	cfg.MetricsServicePort = 0

	// [~server.design.sequences/tuc.VVMConfig.ConfigureSequencesTrustLevel~impl]
	cfg.SequencesTrustLevel = isequencer.SequencesTrustLevel_0

	cfg.Time = testingu.MockTime
	if !coreutils.IsTest() {
		cfg.SecretsReader = itokensjwt.ProvideTestSecretsReader(cfg.SecretsReader)
	}

	emailMessagesChan := make(chan smtptest.Message, 1) // must be buffered
	cfg.ActualizerStateOpts = append(cfg.ActualizerStateOpts, state.WithEmailMessagesChan(emailMessagesChan))

	vitPreConfig := &vitPreConfig{
		vvmCfg:  &cfg,
		vitApps: vitApps{},
		secrets: map[string][]byte{},
	}

	if useCas {
		cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragecas ", fmt.Sprint(cas.DefaultCasParams))
			return cas.Provide(cas.DefaultCasParams)
		}
	}

	for _, opt := range vitCfg.opts {
		opt(vitPreConfig)
	}

	for _, initFunc := range vitPreConfig.initFuncs {
		initFunc()
	}

	cfg.SecretsReader = &implVITISecretsReader{secrets: vitPreConfig.secrets, underlyingReader: cfg.SecretsReader}

	// eliminate timeouts impact for debugging
	cfg.RouterReadTimeout = int(debugTimeout)
	cfg.RouterWriteTimeout = int(debugTimeout)
	cfg.SendTimeout = bus.SendTimeout(debugTimeout)

	vvm, err := vvmpkg.Provide(&cfg)
	require.NoError(t, err)

	// register workspace templates
	for _, app := range vitPreConfig.vitApps {
		ep := vvm.AppsExtensionPoints[app.name]
		for _, tf := range app.wsTemplateFuncs {
			tf(ep)
		}
	}

	vit := &VIT{
		VoedgerVM:            vvm,
		VVMConfig:            &cfg,
		T:                    t,
		appWorkspaces:        map[appdef.AppQName]map[string]*AppWorkspace{},
		principals:           map[appdef.AppQName]map[string]*Principal{},
		lock:                 sync.Mutex{},
		isOnSharedConfig:     vitCfg.isShared,
		configCleanupsAmount: len(vitPreConfig.cleanups),
		emailCaptor:          emailMessagesChan,
		mockTime:             testingu.MockTime,
	}
	httpClient, httpClientCleanup := coreutils.NewIHTTPClient()
	vit.httpClient = httpClient

	vit.cleanups = append(vit.cleanups, vitPreConfig.cleanups...)
	vit.cleanups = append(vit.cleanups, func(vit *VIT) { httpClientCleanup() })

	// launch the server
	// leadership duration - ten years to avoid leadership expiration when time bumps in tests (including 1 day add on each test)
	vit.vvmProblemCtx = vit.Launch(10*365*24*60*60, vvmpkg.DefaultLeadershipAcquisitionDuration)
	vit.checkVVMProblemCtx()

	if vvmLaunchOnly {
		return vit
	}

	for _, app := range vitPreConfig.vitApps {
		// generate verified value tokens if queried
		//                desiredValue token
		verifiedValues := map[string]string{}
		for desiredValue, vvi := range app.verifiedValuesIntents {
			appTokens := vvm.IAppTokensFactory.New(app.name)
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
				WithReqOpt(coreutils.WithExpectedCode(http.StatusCreated)),
				WithReqOpt(coreutils.WithExpectedCode(http.StatusConflict)),
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
				if !vit.PostProfile(prn, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, doc)).IsEmpty() {
					continue
				}
				data := dataFactory(verifiedValues)
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = doc.String()

				bb, err := json.Marshal(data)
				require.NoError(t, err)

				vit.PostProfile(prn, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}

		// time.Sleep(10 * time.Second)
		sysToken, err := payloads.GetSystemPrincipalToken(vit, app.name)
		require.NoError(vit.T, err)
		for _, wsd := range app.ws {
			owner := vit.principals[app.name][wsd.ownerLoginName]
			appWorkspaces, ok := vit.appWorkspaces[app.name]
			if !ok {
				appWorkspaces = map[string]*AppWorkspace{}
				vit.appWorkspaces[app.name] = appWorkspaces
			}
			newAppWS := vit.CreateWorkspace(wsd, owner, coreutils.WithExpectedCode(http.StatusOK), coreutils.WithExpectedCode(http.StatusConflict))
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
	return vit
}

func handleWSParam(vit *VIT, appWS *AppWorkspace, appWorkspaces map[string]*AppWorkspace, verifiedValues map[string]string, token string) {
	for doc, dataFactory := range appWS.docs {
		if !vit.PostWS(appWS, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, doc), coreutils.WithAuthorizeBy(token)).IsEmpty() {
			continue
		}
		data := dataFactory(verifiedValues)
		data[appdef.SystemField_ID] = 1
		data[appdef.SystemField_QName] = doc.String()

		bb, err := json.Marshal(data)
		require.NoError(vit.T, err)

		vit.PostWS(appWS, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb), coreutils.WithAuthorizeBy(token))
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
		roles := ""
		for i, role := range subject.roles {
			if i > 0 {
				roles += ","
			}
			roles += role.String()
		}
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"sys.Subject","Login":"%s","Roles":"%s","SubjectKind":%d,"ProfileWSID":%d}}]}`,
			subject.login, roles, subject.subjectKind, vit.principals[appQName][subject.login].ProfileWSID)
		vit.PostApp(appQName, wsid, "c.sys.CUD", body, coreutils.WithAuthorizeBy(token))
	}
}

func NewVITLocalCassandra(tb testing.TB, vitCfg *VITConfig, opts ...vitOptFunc) (vit *VIT) {
	vit = newVit(tb, vitCfg, true, false)
	for _, opt := range opts {
		opt(vit)
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
	vit.isFinalized = true
	for _, cleanup := range vit.cleanups {
		cleanup(vit)
	}
	vit.cleanups = vit.cleanups[:vit.configCleanupsAmount]
	grNum := runtime.NumGoroutine()
	if grNum-vit.initialGoroutinesNum > allowedGoroutinesNumDiff {
		vit.T.Logf("!!! goroutines leak: was %d on VIT setup, now %d after teardown", vit.initialGoroutinesNum, grNum)
	}
	vit.emailCaptor.checkEmpty(vit.T)
	vit.checkVVMProblemCtx()
	if vit.isOnSharedConfig {
		return
	}
	vit.emailCaptor.shutDown()
	require.NoError(vit.T, vit.Shutdown())
}

func (vit *VIT) MetricsServicePort() int {
	return int(vit.VoedgerVM.MetricsServicePort())
}

func (vit *VIT) GetSystemPrincipal(appQName appdef.AppQName) *Principal {
	vit.T.Helper()
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
			Login: Login{
				Name:        "___sys",
				AppQName:    appQName,
				subjectKind: istructs.SubjectKind_User,
			},
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

func (vit *VIT) PostProfile(prn *Principal, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	opts = append(opts, coreutils.WithDefaultAuthorize(prn.Token))
	return vit.PostApp(prn.AppQName, prn.ProfileWSID, funcName, body, opts...)
}

func (vit *VIT) PostWS(ws *AppWorkspace, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	opts = append(opts, coreutils.WithDefaultAuthorize(ws.Owner.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

// PostWSSys is PostWS authorized by the System Token
func (vit *VIT) PostWSSys(ws *AppWorkspace, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	sysPrn := vit.GetSystemPrincipal(ws.Owner.AppQName)
	opts = append(opts, coreutils.WithDefaultAuthorize(sysPrn.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

func (vit *VIT) SqlQueryRows(ws *AppWorkspace, sqlQuery string, fmtArgs ...any) []map[string]interface{} {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Query":"%s"},"elements":[{"fields":["Result"]}]}`, fmt.Sprintf(sqlQuery, fmtArgs...))
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.WithAuthorizeBy(ws.Owner.Token))
	res := []map[string]interface{}{}
	for _, elem := range resp.Sections[0].Elements {
		m := map[string]interface{}{}
		require.NoError(vit.T, json.Unmarshal([]byte(elem[0][0][0].(string)), &m))
		res = append(res, m)
	}
	return res
}

func (vit *VIT) SqlQuery(ws *AppWorkspace, sqlQuery string, fmtArgs ...any) map[string]interface{} {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Query":"%s"},"elements":[{"fields":["Result"]}]}`, fmt.Sprintf(sqlQuery, fmtArgs...))
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.WithAuthorizeBy(ws.Owner.Token))
	if len(resp.Sections[0].Elements) > 1 {
		vit.T.Fatal("sql query returned few rows. Use SqlQueryRows")
	}
	res := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.Sections[0].Elements[0][0][0][0].(string)), &res))
	return res
}

func (vit *VIT) UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobName string, blobMimeType string, blobContent []byte,
	opts ...coreutils.ReqOptFunc) (blobID istructs.RecordID) {
	vit.T.Helper()
	blobSUUID := vit.UploadTempBLOB(appQName, wsid, blobName, blobMimeType, blobContent, 0, opts...)
	if len(blobSUUID) == 0 {
		return istructs.NullRecordID
	}
	blobIDUint64, err := strconv.ParseUint(string(blobSUUID), utils.DecimalBase, utils.BitSize64)
	require.NoError(vit.T, err)
	return istructs.RecordID(blobIDUint64)
}

func (vit *VIT) UploadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobName string, blobMimeType string, blobContent []byte, duration iblobstorage.DurationType,
	opts ...coreutils.ReqOptFunc) (blobSUUID iblobstorage.SUUID) {
	vit.T.Helper()
	blobReader := iblobstorage.BLOBReader{
		DescrType: iblobstorage.DescrType{
			Name:     blobName,
			MimeType: blobMimeType,
		},
		ReadCloser: io.NopCloser(bytes.NewReader(blobContent)),
	}
	blobSUUID, err := vit.IFederation.UploadTempBLOB(appQName, wsid, blobReader, duration, opts...)
	require.NoError(vit.T, err)
	return blobSUUID
}

func (vit *VIT) Func(url string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	res, err := vit.IFederation.Func(url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

// blob ReadCloser must be read out by the test
// will be closed by the VIT
func (vit *VIT) ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobID istructs.RecordID, optFuncs ...coreutils.ReqOptFunc) iblobstorage.BLOBReader {
	vit.T.Helper()
	reader, err := vit.IFederation.ReadBLOB(appQName, wsid, blobID, optFuncs...)
	require.NoError(vit.T, err)
	vit.registerBLOBReaderCleanup(reader)
	return reader
}

func (vit *VIT) registerBLOBReaderCleanup(reader iblobstorage.BLOBReader) {
	vit.cleanups = append(vit.cleanups, func(vit *VIT) {
		if reader.ReadCloser != nil {
			buf := make([]byte, 1)
			_, err := reader.Read(buf)
			if errors.Is(err, io.EOF) {
				return
			}
			require.NoError(vit.T, err)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(vit.T, err)
			defer reader.Close()
			vit.T.Fatal("BLOB reader is not read out")
		}
	})
}

// blob ReadCloser must be read out by the test
// will be closed by the VIT
func (vit *VIT) ReadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobSUUID iblobstorage.SUUID, optFuncs ...coreutils.ReqOptFunc) iblobstorage.BLOBReader {
	vit.T.Helper()
	blobReader, err := vit.IFederation.ReadTempBLOB(appQName, wsid, blobSUUID, optFuncs...)
	require.NoError(vit.T, err)
	vit.registerBLOBReaderCleanup(blobReader)
	return blobReader
}

func (vit *VIT) POST(relativeURL string, body string, opts ...coreutils.ReqOptFunc) *coreutils.HTTPResponse {
	vit.T.Helper()
	opts = append(opts, coreutils.WithDefaultMethod(http.MethodPost))
	url := vit.URLStr() + "/" + relativeURL
	res, err := vit.httpClient.Req(url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) PostApp(appQName appdef.AppQName, wsid istructs.WSID, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	url := fmt.Sprintf("api/%s/%d/%s", appQName, wsid, funcName)
	res, err := vit.IFederation.Func(url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) WaitFor(consumer func() *coreutils.FuncResponse) *coreutils.FuncResponse {
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

func (vit *VIT) refreshTokens() {
	vit.T.Helper()
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
	vit.refreshTokens()
}

func (vit *VIT) NextName() string {
	return "name_" + strconv.Itoa(vit.NextNumber())
}

// sets `bs` as state of Buckets for `rateLimitName` in `appQName`
// will be automatically restored on vit.TearDown() to the state the Bucket was before MockBuckets() call
func (vit *VIT) MockBuckets(appQName appdef.AppQName, rateLimitName appdef.QName, bs irates.BucketState) {
	vit.T.Helper()
	as, err := vit.BuiltIn(appQName)
	require.NoError(vit.T, err)
	appBuckets := istructsmem.IBucketsFromIAppStructs(as)
	initialState, err := appBuckets.GetDefaultBucketsState(rateLimitName)
	require.NoError(vit.T, err)
	appBuckets.SetDefaultBucketState(rateLimitName, bs)
	appBuckets.ResetRateBuckets(rateLimitName, bs)
	vit.cleanups = append(vit.cleanups, func(vit *VIT) {
		appBuckets.SetDefaultBucketState(rateLimitName, initialState)
		appBuckets.ResetRateBuckets(rateLimitName, initialState)
	})
}

// CaptureEmail waits for and returns the next sent email
// no emails during testEmailsAwaitingTimeout -> test failed
// an email was sent but CaptureEmail is not called -> test will be failed on VIT.TearDown()
func (vit *VIT) CaptureEmail() (msg smtptest.Message) {
	vit.T.Helper()
	tmr := time.NewTimer(getTestEmailsAwaitingTimeout())
	select {
	case msg = <-vit.emailCaptor:
		return msg
	case <-tmr.C:
		vit.T.Fatal("no email messages")
	}
	return
}

// sets delay on IAppStorage.Get() in mem implementation
// will be automatically reset to 0 on TearDown
// need to e.g. investigate slow workspace create, see https://github.com/voedger/voedger/issues/1663
func (vit *VIT) SetMemStorageGetDelay(delay time.Duration) {
	vit.T.Helper()
	vit.iterateDelaySetters(func(delaySetter istorage.IStorageDelaySetter) {
		delaySetter.SetTestDelayGet(delay)
		vit.cleanups = append(vit.cleanups, func(vit *VIT) {
			delaySetter.SetTestDelayGet(0)
		})
	})
}

// sets delay on IAppStorage.Put() in mem implementation
// will be automatically reset to 0 on TearDown
// need to e.g. investigate slow workspace create, see https://github.com/voedger/voedger/issues/1663
func (vit *VIT) SetMemStoragePutDelay(delay time.Duration) {
	vit.T.Helper()
	vit.iterateDelaySetters(func(delaySetter istorage.IStorageDelaySetter) {
		delaySetter.SetTestDelayPut(delay)
		vit.cleanups = append(vit.cleanups, func(vit *VIT) {
			delaySetter.SetTestDelayPut(0)
		})
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

func (ec emailCaptor) checkEmpty(t testing.TB) {
	select {
	case _, ok := <-ec:
		if ok {
			t.Log("unexpected email message received")
			t.Fail()
		}
	default:
	}
}

func (ec emailCaptor) shutDown() {
	close(ec)
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
	select {
	case <-vit.vvmProblemCtx.Done():
		require.NoError(vit.T, vit.Shutdown())
		vit.T.Fatal("vvmProblemCtx is closed but no error on vvm.Shutdown()")
	default:
	}
}
