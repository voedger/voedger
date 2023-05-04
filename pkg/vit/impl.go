/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "embed"

	istoragecas "github.com/heeus/core-istoragecas"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

//go:embed slowtests.txt
var slowTestsPaths string

func NewVIT(t *testing.T, vitCfg *VITConfig, opts ...vitOptFunc) (vit *VIT) {
	if coreutils.SkipSlowTests() {
		pc, _, _, _ := runtime.Caller(1)
		callerFunc := runtime.FuncForPC(pc)
		for _, p := range strings.Split(slowTestsPaths, "\n") {
			fn := callerFunc.Name()
			if fn == p {
				t.Skip("slow test skipped")
			}
		}
	}
	useCas := IsCassandraStorage()
	if !vitCfg.isShared {
		vit = newVit(t, vitCfg, useCas)
	} else {
		ok := false
		if vit, ok = vits[vitCfg]; ok {
			if !vit.isFinalized {
				panic("Teardown() was not called on a previous VIT which used the provided shared config")
			}
			vit.isFinalized = false
		} else {
			vit = newVit(t, vitCfg, useCas)
			vits[vitCfg] = vit
		}
	}

	for _, opt := range opts {
		opt(vit)
	}

	vit.T = t

	// run each test in the next day to mostly prevent previous tests impact and\or workspace initialization
	vit.TimeAdd(day)

	vit.initialGoroutinesNum = runtime.NumGoroutine()

	return vit
}

func newVit(t *testing.T, vitCfg *VITConfig, useCas bool) *VIT {
	cfg := vvm.NewVVMDefaultConfig()

	// only dynamic ports are used in tests
	cfg.VVMPort = 0
	cfg.MetricsServicePort = 0

	cfg.TimeFunc = func() time.Time { return ts.now() }

	emailMessagesChan := make(chan smtptest.Message, 1) // must be buffered
	cfg.ActualizerStateOpts = append(cfg.ActualizerStateOpts, state.WithEmailMessagesChan(emailMessagesChan))

	vitPreConfig := &vitPreConfig{
		vvmCfg:  &cfg,
		vitApps: vitApps{},
	}
	for _, opt := range vitCfg.opts {
		opt(vitPreConfig)
	}

	// eliminate timeouts impact for debugging
	cfg.RouterReadTimeout = int(debugTimeout)
	cfg.RouterWriteTimeout = int(debugTimeout)
	cfg.BusTimeout = vvm.BusTimeout(debugTimeout)

	if useCas {
		cfg.StorageFactory = func() (provider istorage.IAppStorageFactory, err error) {
			logger.Info("using istoragecas ", fmt.Sprint(vvm.DefaultCasParams))
			return istoragecas.Provide(vvm.DefaultCasParams)
		}
	}

	hvm, err := vvm.ProvideVVM(&cfg, 0)
	require.NoError(t, err)

	vit := &VIT{
		HeeusVM:              hvm,
		VVMConfig:            &cfg,
		T:                    t,
		appWorkspaces:        map[istructs.AppQName]map[string]*AppWorkspace{},
		principals:           map[istructs.AppQName]map[string]*Principal{},
		lock:                 sync.Mutex{},
		isOnSharedConfig:     vitCfg.isShared,
		emailMessagesChan:    emailMessagesChan,
		configCleanupsAmount: len(vitPreConfig.cleanups),
	}

	vit.cleanups = append(vit.cleanups, vitPreConfig.cleanups...)

	// запустим сервер
	require.Nil(t, vit.Launch())

	for _, app := range vitPreConfig.vitApps {
		// создадим логины и рабочие области
		for _, login := range app.logins {
			vit.SignUp(login.Name, login.Pwd, login.AppQName)
			prn := vit.SignIn(login)
			appPrincipals, ok := vit.principals[app.name]
			if !ok {
				appPrincipals = map[string]*Principal{}
				vit.principals[app.name] = appPrincipals
			}
			appPrincipals[login.Name] = prn

			for singleton, data := range login.singletons {
				if !vit.PostProfile(prn, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, singleton)).IsEmpty() {
					continue
				}
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = singleton.String()

				bb, err := json.Marshal(data)
				require.NoError(t, err)

				vit.PostProfile(prn, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}
		for _, wsd := range app.ws {
			owner := vit.principals[app.name][wsd.ownerLoginName]
			appWorkspaces, ok := vit.appWorkspaces[app.name]
			if !ok {
				appWorkspaces = map[string]*AppWorkspace{}
				vit.appWorkspaces[app.name] = appWorkspaces
			}
			appWorkspaces[wsd.Name] = vit.CreateWorkspace(wsd, owner)

			for singleton, data := range wsd.singletons {
				if !vit.PostWS(appWorkspaces[wsd.Name], "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, singleton)).IsEmpty() {
					continue
				}
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = singleton.String()

				bb, err := json.Marshal(data)
				require.NoError(t, err)

				vit.PostWS(appWorkspaces[wsd.Name], "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}
	}
	return vit
}

func NewVITLocalCassandra(t *testing.T, vitCfg *VITConfig, opts ...vitOptFunc) (vit *VIT) {
	vit = newVit(t, vitCfg, true)
	for _, opt := range opts {
		opt(vit)
	}

	return vit
}

// WSID as wsid[0] or 1, system owner
// command processor will skip initialization check for that workspace
func (vit *VIT) DummyWS(appQName istructs.AppQName, awsid ...istructs.WSID) *AppWorkspace {
	wsid := istructs.WSID(1)
	if len(awsid) > 0 {
		wsid = awsid[0]
	}
	commandprocessor.AddDummyWS(wsid)
	sysPrn := vit.GetSystemPrincipal(appQName)
	return &AppWorkspace{
		WorkspaceDescriptor: WorkspaceDescriptor{
			WSParams: WSParams{
				Kind:      appdef.NullQName,
				ClusterID: istructs.MainClusterID,
			},
			WSID: wsid,
		},
		Owner: sysPrn,
	}
}

func (vit *VIT) WS(appQName istructs.AppQName, wsName string) *AppWorkspace {
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
	select {
	case <-vit.emailMessagesChan:
		vit.T.Log("unexpected email message received")
		vit.T.Fail()
	default:
	}
	if vit.isOnSharedConfig {
		return
	}
	vit.Shutdown()
}

func (vit *VIT) MetricsServicePort() int {
	return int(vit.HeeusVM.MetricsServicePort())
}

func (vit *VIT) GetSystemPrincipal(appQName istructs.AppQName) *Principal {
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
		as, err := vit.IAppStructsProvider.AppStructs(appQName)
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

func (vit *VIT) GetPrincipal(appQName istructs.AppQName, login string) *Principal {
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
	opts = append(opts, coreutils.WithAuthorizeByIfNot(prn.Token))
	return vit.PostApp(prn.AppQName, prn.ProfileWSID, funcName, body, opts...)
}

func (vit *VIT) PostWS(ws *AppWorkspace, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	opts = append(opts, coreutils.WithAuthorizeByIfNot(ws.Owner.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

// PostWSSys is PostWS authorized by the System Token
func (vit *VIT) PostWSSys(ws *AppWorkspace, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	sysPrn := vit.GetSystemPrincipal(ws.Owner.AppQName)
	opts = append(opts, coreutils.WithAuthorizeByIfNot(sysPrn.Token))
	return vit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

func (vit *VIT) PostFree(url string, body string, opts ...coreutils.ReqOptFunc) *coreutils.HTTPResponse {
	vit.T.Helper()
	opts = append(opts, coreutils.WithMethod(http.MethodPost))
	res, err := coreutils.Req(url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) Post(url string, body string, opts ...coreutils.ReqOptFunc) *coreutils.HTTPResponse {
	vit.T.Helper()
	res, err := coreutils.FederationPOST(vit.VVMAPI.FederationURL(), url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) PostApp(appQName istructs.AppQName, wsid istructs.WSID, funcName string, body string, opts ...coreutils.ReqOptFunc) *coreutils.FuncResponse {
	vit.T.Helper()
	url := fmt.Sprintf("api/%s/%d/%s", appQName, wsid, funcName)
	res, err := coreutils.FederationFunc(vit.VVMAPI.FederationURL(), url, body, opts...)
	require.NoError(vit.T, err)
	return res
}

func (vit *VIT) Get(url string, opts ...coreutils.ReqOptFunc) *coreutils.HTTPResponse {
	vit.T.Helper()
	res, err := coreutils.FederationReq(vit.VVMAPI.FederationURL(), url, "", opts...)
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
				Login:       prn.Login.Name,
				SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: istructs.WSID(prn.ProfileWSID),
			}
			as, err := vit.IAppStructsProvider.AppStructs(prn.AppQName)
			require.NoError(vit.T, err) // notest
			newToken, err := as.AppTokens().IssueToken(signupin.DefaultPrincipalTokenExpiration, &principalPayload)
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
	return ts.now()
}

func (vit *VIT) SetNow(now time.Time) {
	ts.setCurrentInstant(now)
	vit.refreshTokens()
}

func (vit *VIT) TimeAdd(dur time.Duration) {
	ts.add(dur)
	vit.refreshTokens()
}

func (vit *VIT) NextName() string {
	return "name_" + strconv.Itoa(vit.NextNumber())
}

func (vit *VIT) ExpectEmail() *EmailCaptor {
	ec := &EmailCaptor{ch: make(chan smtptest.Message, 1), vit: vit}
	go func() {
		m := <-vit.emailMessagesChan
		ec.ch <- m
	}()
	return ec
}

// sets `bs` as state of Buckets for `rateLimitName` in `appQName`
// will be automatically restored on vit.TearDown() to the state the Bucket was before MockBuckets() call
func (vit *VIT) MockBuckets(appQName istructs.AppQName, rateLimitName string, bs irates.BucketState) {
	vit.T.Helper()
	as, err := vit.IAppStructsProvider.AppStructs(appQName)
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

func (ts *timeService) now() time.Time {
	ts.m.Lock()
	res := ts.currentInstant
	ts.m.Unlock()
	return res
}

func (ts *timeService) add(dur time.Duration) {
	ts.m.Lock()
	ts.currentInstant = ts.currentInstant.Add(dur)
	ts.m.Unlock()
}

func (ts *timeService) setCurrentInstant(now time.Time) {
	ts.m.Lock()
	ts.currentInstant = now
	ts.m.Unlock()
}

func ScanSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (ec *EmailCaptor) Capture() (m smtptest.Message) {
	tmr := time.NewTimer(testEmailsAwaitingTimeout)
	select {
	case m = <-ec.ch:
		return m
	case <-tmr.C:
	}
	ec.vit.T.Fatal("no email messages")
	return
}
