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
	"github.com/untillpro/airs-bp3/utils"
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

func NewHIT(t *testing.T, hitCfg *HITConfig, opts ...hitOptFunc) (hit *HIT) {
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
	if !hitCfg.isShared {
		hit = newHit(t, hitCfg, useCas)
	} else {
		ok := false
		if hit, ok = hits[hitCfg]; ok {
			if !hit.isFinalized {
				panic("Teardown() was not called on a previous HIT which used the provided shared config")
			}
			hit.isFinalized = false
		} else {
			hit = newHit(t, hitCfg, useCas)
			hits[hitCfg] = hit
		}
	}

	for _, opt := range opts {
		opt(hit)
	}

	hit.T = t

	// run each test in the next day to mostly prevent previous tests impact and\or workspace initialization
	hit.TimeAdd(day)

	hit.initialGoroutinesNum = runtime.NumGoroutine()

	return hit
}

func newHit(t *testing.T, hitCfg *HITConfig, useCas bool) *HIT {
	cfg := vvm.NewHVMDefaultConfig()

	// only dynamic ports are used in tests
	cfg.HVMPort = 0
	cfg.MetricsServicePort = 0

	cfg.TimeFunc = func() time.Time { return ts.now() }

	emailMessagesChan := make(chan smtptest.Message, 1) // must be buffered
	cfg.ActualizerStateOpts = append(cfg.ActualizerStateOpts, state.WithEmailMessagesChan(emailMessagesChan))

	hitPreConfig := &hitPreConfig{
		hvmCfg:  &cfg,
		hitApps: hitApps{},
	}
	for _, opt := range hitCfg.opts {
		opt(hitPreConfig)
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

	hvm, err := vvm.ProvideHVM(&cfg, 0)
	require.NoError(t, err)

	hit := &HIT{
		HeeusVM:              hvm,
		HVMConfig:            &cfg,
		T:                    t,
		appWorkspaces:        map[istructs.AppQName]map[string]*AppWorkspace{},
		principals:           map[istructs.AppQName]map[string]*Principal{},
		lock:                 sync.Mutex{},
		isOnSharedConfig:     hitCfg.isShared,
		emailMessagesChan:    emailMessagesChan,
		configCleanupsAmount: len(hitPreConfig.cleanups),
	}

	hit.cleanups = append(hit.cleanups, hitPreConfig.cleanups...)

	// запустим сервер
	require.Nil(t, hit.Launch())

	for _, app := range hitPreConfig.hitApps {
		// создадим логины и рабочие области
		for _, login := range app.logins {
			hit.SignUp(login.Name, login.Pwd, login.AppQName)
			prn := hit.SignIn(login)
			appPrincipals, ok := hit.principals[app.name]
			if !ok {
				appPrincipals = map[string]*Principal{}
				hit.principals[app.name] = appPrincipals
			}
			appPrincipals[login.Name] = prn

			for singleton, data := range login.singletons {
				if !hit.PostProfile(prn, "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, singleton)).IsEmpty() {
					continue
				}
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = singleton.String()

				bb, err := json.Marshal(data)
				require.NoError(t, err)

				hit.PostProfile(prn, "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}
		for _, wsd := range app.ws {
			owner := hit.principals[app.name][wsd.ownerLoginName]
			appWorkspaces, ok := hit.appWorkspaces[app.name]
			if !ok {
				appWorkspaces = map[string]*AppWorkspace{}
				hit.appWorkspaces[app.name] = appWorkspaces
			}
			appWorkspaces[wsd.Name] = hit.CreateWorkspace(wsd, owner)

			for singleton, data := range wsd.singletons {
				if !hit.PostWS(appWorkspaces[wsd.Name], "q.sys.Collection", fmt.Sprintf(`{"args":{"Schema":"%s"}}`, singleton)).IsEmpty() {
					continue
				}
				data[appdef.SystemField_ID] = 1
				data[appdef.SystemField_QName] = singleton.String()

				bb, err := json.Marshal(data)
				require.NoError(t, err)

				hit.PostWS(appWorkspaces[wsd.Name], "c.sys.CUD", fmt.Sprintf(`{"cuds":[{"fields":%s}]}`, bb))
			}
		}
	}
	return hit
}

func NewHITLocalCassandra(t *testing.T, hitCfg *HITConfig, opts ...hitOptFunc) (hit *HIT) {
	hit = newHit(t, hitCfg, true)
	for _, opt := range opts {
		opt(hit)
	}

	return hit
}

// WSID as wsid[0] or 1, system owner
// command processor will skip initialization check for that workspace
func (hit *HIT) DummyWS(appQName istructs.AppQName, awsid ...istructs.WSID) *AppWorkspace {
	wsid := istructs.WSID(1)
	if len(awsid) > 0 {
		wsid = awsid[0]
	}
	commandprocessor.AddDummyWS(wsid)
	sysPrn := hit.GetSystemPrincipal(appQName)
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

func (hit *HIT) WS(appQName istructs.AppQName, wsName string) *AppWorkspace {
	appWorkspaces, ok := hit.appWorkspaces[appQName]
	if !ok {
		panic("unknown app " + appQName.String())
	}
	if ws, ok := appWorkspaces[wsName]; ok {
		return ws
	}
	panic("unknown workspace " + wsName)
}

func (hit *HIT) TearDown() {
	hit.T.Helper()
	hit.isFinalized = true
	for _, cleanup := range hit.cleanups {
		cleanup(hit)
	}
	hit.cleanups = hit.cleanups[:hit.configCleanupsAmount]
	grNum := runtime.NumGoroutine()
	if grNum-hit.initialGoroutinesNum > allowedGoroutinesNumDiff {
		hit.T.Logf("!!! goroutines leak: was %d on HIT setup, now %d after teardown", hit.initialGoroutinesNum, grNum)
	}
	select {
	case <-hit.emailMessagesChan:
		hit.T.Log("unexpected email message received")
		hit.T.Fail()
	default:
	}
	if hit.isOnSharedConfig {
		return
	}
	hit.Shutdown()
}

func (hit *HIT) MetricsServicePort() int {
	return int(hit.HeeusVM.MetricsServicePort())
}

func (hit *HIT) GetSystemPrincipal(appQName istructs.AppQName) *Principal {
	hit.T.Helper()
	hit.lock.Lock()
	defer hit.lock.Unlock()
	appPrincipals, ok := hit.principals[appQName]
	if !ok {
		appPrincipals = map[string]*Principal{}
		hit.principals[appQName] = appPrincipals
	}
	prn, ok := appPrincipals["___sys"]
	if !ok {
		as, err := hit.IAppStructsProvider.AppStructs(appQName)
		require.NoError(hit.T, err)
		sysToken, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
		require.NoError(hit.T, err)
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

func (hit *HIT) GetPrincipal(appQName istructs.AppQName, login string) *Principal {
	appPrincipals, ok := hit.principals[appQName]
	if !ok {
		hit.T.Fatalf("unknown app %s", appQName)
	}
	prn, ok := appPrincipals[login]
	if !ok {
		hit.T.Fatalf("unknown login %s", login)
	}
	return prn
}

func (hit *HIT) PostProfile(prn *Principal, funcName string, body string, opts ...utils.ReqOptFunc) *utils.FuncResponse {
	hit.T.Helper()
	opts = append(opts, utils.WithAuthorizeByIfNot(prn.Token))
	return hit.PostApp(prn.AppQName, prn.ProfileWSID, funcName, body, opts...)
}

func (hit *HIT) PostWS(ws *AppWorkspace, funcName string, body string, opts ...utils.ReqOptFunc) *utils.FuncResponse {
	hit.T.Helper()
	opts = append(opts, utils.WithAuthorizeByIfNot(ws.Owner.Token))
	return hit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

// PostWSSys is PostWS authorized by the System Token
func (hit *HIT) PostWSSys(ws *AppWorkspace, funcName string, body string, opts ...utils.ReqOptFunc) *utils.FuncResponse {
	hit.T.Helper()
	sysPrn := hit.GetSystemPrincipal(ws.Owner.AppQName)
	opts = append(opts, utils.WithAuthorizeByIfNot(sysPrn.Token))
	return hit.PostApp(ws.Owner.AppQName, ws.WSID, funcName, body, opts...)
}

func (hit *HIT) PostFree(url string, body string, opts ...utils.ReqOptFunc) *utils.HTTPResponse {
	hit.T.Helper()
	opts = append(opts, utils.WithMethod(http.MethodPost))
	res, err := utils.Req(url, body, opts...)
	require.NoError(hit.T, err)
	return res
}

func (hit *HIT) Post(url string, body string, opts ...utils.ReqOptFunc) *utils.HTTPResponse {
	hit.T.Helper()
	res, err := utils.FederationPOST(hit.FederationURL(), url, body, opts...)
	require.NoError(hit.T, err)
	return res
}

func (hit *HIT) PostApp(appQName istructs.AppQName, wsid istructs.WSID, funcName string, body string, opts ...utils.ReqOptFunc) *utils.FuncResponse {
	hit.T.Helper()
	url := fmt.Sprintf("api/%s/%d/%s", appQName, wsid, funcName)
	res, err := utils.FederationFunc(hit.FederationURL(), url, body, opts...)
	require.NoError(hit.T, err)
	return res
}

func (hit *HIT) Get(url string, opts ...utils.ReqOptFunc) *utils.HTTPResponse {
	hit.T.Helper()
	res, err := utils.FederationReq(hit.FederationURL(), url, "", opts...)
	require.NoError(hit.T, err)
	return res
}

func (hit *HIT) WaitFor(consumer func() *utils.FuncResponse) *utils.FuncResponse {
	hit.T.Helper()
	start := time.Now()
	for time.Since(start) < testTimeout {
		resp := consumer()
		if len(resp.Sections) > 0 {
			return resp
		}
		logger.Info("waiting for projection")
		time.Sleep(100 * time.Millisecond)
	}
	hit.T.Fail()
	return nil
}

func (hit *HIT) refreshTokens() {
	hit.T.Helper()
	for _, appPrns := range hit.principals {
		for _, prn := range appPrns {
			// issue principal token
			principalPayload := payloads.PrincipalPayload{
				Login:       prn.Login.Name,
				SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: istructs.WSID(prn.ProfileWSID),
			}
			as, err := hit.IAppStructsProvider.AppStructs(prn.AppQName)
			require.NoError(hit.T, err) // notest
			newToken, err := as.AppTokens().IssueToken(signupin.DefaultPrincipalTokenExpiration, &principalPayload)
			require.NoError(hit.T, err)
			prn.Token = newToken
		}
	}
}

func (hit *HIT) NextNumber() int {
	hit.lock.Lock()
	hit.nextNumber++
	res := hit.nextNumber
	hit.lock.Unlock()
	return res
}

func (hit *HIT) Now() time.Time {
	return ts.now()
}

func (hit *HIT) SetNow(now time.Time) {
	ts.setCurrentInstant(now)
	hit.refreshTokens()
}

func (hit *HIT) TimeAdd(dur time.Duration) {
	ts.add(dur)
	hit.refreshTokens()
}

func (hit *HIT) NextName() string {
	return "name_" + strconv.Itoa(hit.NextNumber())
}

func (hit *HIT) ExpectEmail() *EmailCaptor {
	ec := &EmailCaptor{ch: make(chan smtptest.Message, 1), hit: hit}
	go func() {
		m := <-hit.emailMessagesChan
		ec.ch <- m
	}()
	return ec
}

// sets `bs` as state of Buckets for `rateLimitName` in `appQName`
// will be automatically restored on hit.TearDown() to the state the Bucket was before MockBuckets() call
func (hit *HIT) MockBuckets(appQName istructs.AppQName, rateLimitName string, bs irates.BucketState) {
	hit.T.Helper()
	as, err := hit.IAppStructsProvider.AppStructs(appQName)
	require.NoError(hit.T, err)
	appBuckets := istructsmem.IBucketsFromIAppStructs(as)
	initialState, err := appBuckets.GetDefaultBucketsState(rateLimitName)
	require.NoError(hit.T, err)
	appBuckets.SetDefaultBucketState(rateLimitName, bs)
	appBuckets.ResetRateBuckets(rateLimitName, bs)
	hit.cleanups = append(hit.cleanups, func(hit *HIT) {
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
	ec.hit.T.Fatal("no email messages")
	return
}
