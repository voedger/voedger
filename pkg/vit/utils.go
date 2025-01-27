/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istorage"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/vvm"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func (vit *VIT) GetBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobID istructs.RecordID, token string) *BLOB {
	vit.T.Helper()
	blobReader, err := vit.IFederation.ReadBLOB(appQName, wsid, blobID, coreutils.WithAuthorizeBy(token))
	require.NoError(vit.T, err)
	blobContent, err := io.ReadAll(blobReader)
	require.NoError(vit.T, err)
	return &BLOB{
		Content:  blobContent,
		Name:     blobReader.Name,
		MimeType: blobReader.MimeType,
	}
}

func (vit *VIT) signUp(login Login, wsKindInitData string, opts ...coreutils.ReqOptFunc) {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":%q,"ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
		login.Name, login.AppQName.String(), login.subjectKind, wsKindInitData, login.clusterID, login.Pwd)
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.CreateLogin", body, opts...)
}

func WithClusterID(clusterID istructs.ClusterID) signUpOptFunc {
	return func(opts *signUpOpts) {
		opts.profileClusterID = clusterID
	}
}

func WithReqOpt(reqOpt coreutils.ReqOptFunc) signUpOptFunc {
	return func(opts *signUpOpts) {
		opts.reqOpts = append(opts.reqOpts, reqOpt)
	}
}

func (vit *VIT) SignUp(loginName, pwd string, appQName appdef.AppQName, opts ...signUpOptFunc) Login {
	vit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_User, signUpOpts.profileClusterID)
	vit.signUp(login, `{"DisplayName":"User Name"}`, signUpOpts.reqOpts...)
	return login
}

func getSignUpOpts(opts []signUpOptFunc) *signUpOpts {
	res := &signUpOpts{
		profileClusterID: istructs.CurrentClusterID(),
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (vit *VIT) SignUpDevice(loginName, pwd string, appQName appdef.AppQName, opts ...signUpOptFunc) Login {
	vit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_Device, signUpOpts.profileClusterID)
	vit.signUp(login, "{}", signUpOpts.reqOpts...)
	return login
}

func (vit *VIT) GetCDocLoginID(login Login) int64 {
	vit.T.Helper()
	as, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_sys_registry)
	require.NoError(vit.T, err) // notest
	appWSID := coreutils.GetAppWSID(login.PseudoProfileWSID, as.NumAppWorkspaces())
	body := fmt.Sprintf(`{"args":{"Query":"select CDocLoginID from registry.LoginIdx where AppWSID = %d and AppIDLoginHash = '%s/%s'"}, "elements":[{"fields":["Result"]}]}`,
		appWSID, login.AppQName, registry.GetLoginHash(login.Name))
	sys := vit.GetSystemPrincipal(istructs.AppQName_sys_registry)
	resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.SqlQuery", body, coreutils.WithAuthorizeBy(sys.Token))
	m := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	return int64(m["CDocLoginID"].(float64))
}

func (vit *VIT) GetCDocWSKind(ws *AppWorkspace) (cdoc map[string]interface{}, id int64) {
	vit.T.Helper()
	return vit.getCDoc(ws.Owner.AppQName, ws.Kind, ws.WSID)
}

func (vit *VIT) getCDoc(appQName appdef.AppQName, qName appdef.QName, wsid istructs.WSID) (cdoc map[string]interface{}, id int64) {
	vit.T.Helper()
	body := bytes.NewBufferString(fmt.Sprintf(`{"args":{"Schema":"%s"},"elements":[{"fields":["sys.ID"`, qName))
	fields := []string{}
	as, err := vit.IAppStructsProvider.BuiltIn(appQName)
	require.NoError(vit.T, err)
	if doc := appdef.CDoc(as.AppDef().Type, qName); doc != nil {
		for _, field := range doc.Fields() {
			if field.IsSys() {
				continue
			}
			body.WriteString(fmt.Sprintf(`,"%s"`, field.Name()))
			fields = append(fields, field.Name())
		}
	}
	body.WriteString("]}]}")
	sys := vit.GetSystemPrincipal(appQName)
	resp := vit.PostApp(appQName, wsid, "q.sys.Collection", body.String(), coreutils.WithAuthorizeBy(sys.Token))
	if len(resp.Sections) == 0 {
		vit.T.Fatalf("no CDoc<%s> at workspace id %d", qName.String(), wsid)
	}
	id = int64(resp.SectionRow()[0].(float64))
	cdoc = map[string]interface{}{}
	for i, fieldName := range fields {
		cdoc[fieldName] = resp.SectionRow()[i+1]
	}
	return
}

func (vit *VIT) GetCDocChildWorkspace(ws *AppWorkspace) (cdoc map[string]interface{}, id int64) {
	vit.T.Helper()
	return vit.getCDoc(ws.Owner.AppQName, authnz.QNameCDocChildWorkspace, ws.Owner.ProfileWSID)
}

func (vit *VIT) waitForWorkspace(wsName string, owner *Principal, respGetter func(owner *Principal, body string) *coreutils.FuncResponse, expectWSInitErrorChunks ...string) (ws *AppWorkspace) {
	const (
		// respect linter
		tmplNameIdx   = 3
		tmplParamsIdx = 4
		wsidIdx       = 5
		wsErrIdx      = 6
	)
	deadline := time.Now().Add(getWorkspaceInitAwaitTimeout())
	logger.Verbose("workspace", wsName, "awaiting started")
	for time.Now().Before(deadline) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": %q
				},
				"elements":[
					{
						"fields":["WSName", "WSKind", "WSKindInitializationData", "TemplateName", "TemplateParams", "WSID", "WSError"]
					}
				]
			}`, wsName)

		resp := respGetter(owner, body)
		wsid := istructs.WSID(resp.SectionRow()[wsidIdx].(float64))
		wsError := resp.SectionRow()[wsErrIdx].(string)
		if wsid == 0 && len(wsError) == 0 {
			time.Sleep(workspaceQueryDelay)
			continue
		}
		wsKind, err := appdef.ParseQName(resp.SectionRow()[1].(string))
		require.NoError(vit.T, err)

		if len(expectWSInitErrorChunks) > 0 {
			tempWSError := wsError
			for _, errChunk := range expectWSInitErrorChunks {
				if strings.Contains(wsError, errChunk) {
					tempWSError = tempWSError[:strings.Index(tempWSError, errChunk)+len(errChunk)]
					continue
				}
				vit.T.Fatalf(`expected ws init error template is [%s] but is "%s"`, strings.Join(expectWSInitErrorChunks, ", "), wsError)
			}
		} else {
			require.Empty(vit.T, wsError)
		}

		return &AppWorkspace{
			WorkspaceDescriptor: WorkspaceDescriptor{
				WSParams: WSParams{
					Name:           resp.SectionRow()[0].(string),
					Kind:           wsKind,
					InitDataJSON:   resp.SectionRow()[2].(string),
					TemplateName:   resp.SectionRow()[tmplNameIdx].(string),
					TemplateParams: resp.SectionRow()[tmplParamsIdx].(string),
					ClusterID:      istructs.CurrentClusterID(),
					ownerLoginName: owner.Name,
				},
				WSID:    wsid,
				WSError: wsError,
			},
			Owner: owner,
		}
	}
	vit.T.Fatalf("workspace %s is not initialized in an acceptable time", wsName)
	return ws
}

func (vit *VIT) WaitForProfile(cdocLoginID istructs.RecordID, login string, appQName appdef.AppQName, expectWSInitErrorChunks ...string) (profileWSID istructs.WSID) {
	vit.T.Helper()
	deadline := time.Now().Add(getWorkspaceInitAwaitTimeout())
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
	queryCDocLoginBody := fmt.Sprintf(`{"args":{"Query":"select * from registry.Login.%d"},"elements":[{"fields":["Result"]}]}`, cdocLoginID)
	sysToken, err := payloads.GetSystemPrincipalToken(vit.ITokens, istructs.AppQName_sys_registry)
	require.NoError(vit.T, err)
	for time.Now().Before(deadline) {
		resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.SqlQuery", queryCDocLoginBody, coreutils.WithAuthorizeBy(sysToken))
		m := map[string]interface{}{}
		require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		wsError := m["WSError"].(string)
		if len(wsError) > 0 {
			if len(expectWSInitErrorChunks) > 0 {
				tempWSErr := wsError
				for _, errChunk := range expectWSInitErrorChunks {
					errChunkIdx := strings.Index(tempWSErr, errChunk)
					if errChunkIdx == 0 {
						vit.T.Fatalf("expected error should contain [%s] but is %s", strings.Join(expectWSInitErrorChunks, ","), wsError)
					}
					tempWSErr = tempWSErr[errChunkIdx+len(errChunk):]
				}
				return istructs.WSID(m["WSID"].(float64))
			}
			vit.T.Fatalf("profile init error: %s", wsError)
		}
		if m["WSID"].(float64) > 0 {
			if len(expectWSInitErrorChunks) > 0 {
				vit.T.Fatalf("profile init error should contain [%s] but inited with no error", strings.Join(expectWSInitErrorChunks, ","))
			}
			return istructs.WSID(m["WSID"].(float64))
		}
	}
	panic("profile init await taimout")
}

func (vit *VIT) WaitForWorkspace(wsName string, owner *Principal, expectWSInitErrorChunks ...string) (ws *AppWorkspace) {
	return vit.waitForWorkspace(wsName, owner, func(owner *Principal, body string) *coreutils.FuncResponse {
		return vit.PostProfile(owner, "q.sys.QueryChildWorkspaceByName", body)
	}, expectWSInitErrorChunks...)
}

func (vit *VIT) WaitForChildWorkspace(parentWS *AppWorkspace, wsName string) (ws *AppWorkspace) {
	return vit.waitForWorkspace(wsName, parentWS.Owner, func(owner *Principal, body string) *coreutils.FuncResponse {
		return vit.PostWS(parentWS, "q.sys.QueryChildWorkspaceByName", body)
	})
}

func DoNotFailOnTimeout() signInOptFunc {
	return func(opts *signInOpts) {
		opts.failOnTimeout = false
	}
}

func (vit *VIT) SignIn(login Login, optFuncs ...signInOptFunc) (prn *Principal) {
	vit.T.Helper()
	opts := &signInOpts{
		failOnTimeout: true,
	}
	for _, opt := range optFuncs {
		opt(opts)
	}
	deadline := time.Now().Add(getWorkspaceInitAwaitTimeout())
	for time.Now().Before(deadline) {
		body := fmt.Sprintf(`
			{
				"args": {
					"Login": "%s",
					"Password": "%s",
					"AppName": "%s"
				},
				"elements":[
					{
						"fields":["PrincipalToken", "WSID", "WSError"]
					}
				]
			}`, login.Name, login.Pwd, login.AppQName.String())
		resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body)
		profileWSID := istructs.WSID(resp.SectionRow()[1].(float64))
		wsError := resp.SectionRow()[2].(string)
		token := resp.SectionRow()[0].(string)
		if profileWSID == 0 && len(wsError) == 0 {
			time.Sleep(workspaceQueryDelay)
			continue
		}
		require.Empty(vit.T, wsError)
		require.NotEmpty(vit.T, token)
		return &Principal{
			Login:       login,
			Token:       token,
			ProfileWSID: profileWSID,
		}
	}
	if opts.failOnTimeout {
		vit.T.Fatal("user profile is not initialized in an acceptable time")
	}
	return nil
}

// owner could be *vit.Principal or *vit.AppWorkspace
func (vit *VIT) InitChildWorkspace(wsd WSParams, ownerIntf interface{}, opts ...coreutils.ReqOptFunc) {
	vit.T.Helper()
	body := fmt.Sprintf(`{
		"args": {
			"WSName": %q,
			"WSKind": "%s",
			"WSKindInitializationData": %q,
			"TemplateName": "%s",
			"TemplateParams": %q,
			"WSClusterID": %d
		}
	}`, wsd.Name, wsd.Kind.String(), wsd.InitDataJSON, wsd.TemplateName, wsd.TemplateParams, wsd.ClusterID)

	switch owner := ownerIntf.(type) {
	case *Principal:
		vit.PostProfile(owner, "c.sys.InitChildWorkspace", body, opts...)
	case *AppWorkspace:
		vit.PostWS(owner, "c.sys.InitChildWorkspace", body, opts...)
	default:
		panic("ownerIntf could be vit.*Principal or vit.*AppWorkspace only")
	}
}

func SimpleWSParams(wsName string) WSParams {
	return WSParams{
		Name:         wsName,
		Kind:         QNameApp1_TestWSKind,
		ClusterID:    istructs.CurrentClusterID(),
		InitDataJSON: `{"IntFld": 42}`,
	}
}

func (vit *VIT) CreateWorkspace(wsp WSParams, owner *Principal, opts ...coreutils.ReqOptFunc) *AppWorkspace {
	vit.InitChildWorkspace(wsp, owner, opts...)
	ws := vit.WaitForWorkspace(wsp.Name, owner)
	require.Empty(vit.T, ws.WSError)
	return ws
}

func (vit *VIT) SubscribeForN10n(ws *AppWorkspace, projectionQName appdef.QName) federation.OffsetsChan {
	vit.T.Helper()
	return vit.SubscribeForN10nProjectionKey(in10n.ProjectionKey{
		App:        ws.AppQName(),
		Projection: projectionQName,
		WS:         ws.WSID,
	})
}

// will be unsubscribed automatically on vit.TearDown()
func (vit *VIT) SubscribeForN10nProjectionKey(pk in10n.ProjectionKey) federation.OffsetsChan {
	vit.T.Helper()
	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(pk)
	vit.lock.Lock() // need to lock because the vit instance is used in different goroutines in e.g. Test_Race_RestaurantIntenseUsage()
	vit.cleanups = append(vit.cleanups, func(vit *VIT) {
		unsubscribe()
	})
	vit.lock.Unlock()
	return offsetsChan
}

func (vit *VIT) SubscribeForN10nUnsubscribe(pk in10n.ProjectionKey) (offsetsChan federation.OffsetsChan, unsubscribe func()) {
	vit.T.Helper()
	offsetsChan, unsubscribe, err := vit.IFederation.N10NSubscribe(pk)
	require.NoError(vit.T, err)
	return offsetsChan, unsubscribe
}

func (vit *VIT) MetricsRequest(client coreutils.IHTTPClient, opts ...coreutils.ReqOptFunc) (resp string) {
	vit.T.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", vit.VoedgerVM.MetricsServicePort())
	res, err := client.Req(url, "", opts...)
	require.NoError(vit.T, err)
	return res.Body
}

func (vit *VIT) GetAny(entity string, ws *AppWorkspace) istructs.RecordID {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Query":"select DocID from sys.CollectionView where PartKey = 1 and DocQName = '%s'"},"elements":[{"fields":["Result"]}]}`, entity)
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	if len(resp.Sections) == 0 {
		vit.T.Fatalf("no %s at workspace id %d", entity, ws.WSID)
	}
	data := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &data))
	return istructs.RecordID(data["DocID"].(float64))
}

func NewLogin(name, pwd string, appQName appdef.AppQName, subjectKind istructs.SubjectKindType, clusterID istructs.ClusterID) Login {
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, name, istructs.CurrentClusterID())
	return Login{name, pwd, pseudoWSID, appQName, subjectKind, clusterID, map[appdef.QName]func(verifiedValues map[string]string) map[string]interface{}{}, []subject{}}
}

func TestDeadline() time.Time {
	deadline := time.Now().Add(5 * time.Second)
	if coreutils.IsDebug() {
		deadline = deadline.Add(time.Hour)
	}
	return deadline
}

func getTestEmailsAwaitingTimeout() time.Duration {
	if coreutils.IsDebug() {
		return math.MaxInt
	}
	return testEmailsAwaitingTimeout
}

func getWorkspaceInitAwaitTimeout() time.Duration {
	if coreutils.IsDebug() {
		// so long for Test_Race_RestaurantIntenseUsage with -race
		return math.MaxInt
	}
	return defaultWorkspaceAwaitTimeout
}

func DummyWS(wsKind appdef.QName, wsid istructs.WSID, ownerPrn *Principal) *AppWorkspace {
	return &AppWorkspace{
		WorkspaceDescriptor: WorkspaceDescriptor{
			WSParams: WSParams{
				Kind:      wsKind,
				ClusterID: istructs.CurrentClusterID(),
			},
			WSID: wsid,
		},
		Owner: ownerPrn,
	}
}

// calls testBeforeRestart() then stops then VIT, then launches new VIT on the same config but with storage from previous VIT
// then calls testAfterRestart() with the new VIT
// cfg must be owned
func TestRestartPreservingStorage(t *testing.T, cfg *VITConfig, testBeforeRestart, testAfterRestart func(t *testing.T, vit *VIT)) {
	require.False(t, cfg.isShared, "storage restart could be done on Own VIT Config only")
	var sharedStorageFactory istorage.IAppStorageFactory
	suffix := t.Name() + uuid.NewString()
	cfg.opts = append(cfg.opts, WithVVMConfig(func(cfg *vvm.VVMConfig) {
		if sharedStorageFactory == nil {
			var err error
			sharedStorageFactory, err = cfg.StorageFactory()
			require.NoError(t, err)
		}
		cfg.KeyspaceNameSuffix = suffix
		cfg.StorageFactory = func() (istorage.IAppStorageFactory, error) {
			return sharedStorageFactory, nil
		}
	}))
	func() {
		vit := NewVIT(t, cfg)
		defer vit.TearDown()
		testBeforeRestart(t, vit)
	}()
	vit := NewVIT(t, cfg)
	defer vit.TearDown()
	testAfterRestart(t, vit)
}
