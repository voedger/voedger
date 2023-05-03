/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func (hit *HIT) GetBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobID int64, token string) *BLOB {
	hit.T.Helper()
	resp, err := utils.FederationReq(hit.FederationURL(), fmt.Sprintf(`blob/%s/%d/%d`, appQName.String(), wsid, blobID), "", utils.WithAuthorizeBy(token))
	require.NoError(hit.T, err)
	contentDisposition := resp.HTTPResp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	require.NoError(hit.T, err)
	return &BLOB{
		Content:  []byte(resp.Body),
		Name:     params["filename"],
		MimeType: resp.HTTPResp.Header.Get(coreutils.ContentType),
	}
}

func (hit *HIT) signUp(login Login, wsKindInitData string, opts ...utils.ReqOptFunc) {
	hit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":%q,"ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
		login.Name, login.AppQName.String(), login.subjectKind, wsKindInitData, login.clusterID, login.Pwd)
	hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.CreateLogin", body, opts...)
}

func WithClusterID(clusterID istructs.ClusterID) signUpOptFunc {
	return func(opts *signUpOpts) {
		opts.profileClusterID = clusterID
	}
}

func WithReqOpt(reqOpt utils.ReqOptFunc) signUpOptFunc {
	return func(opts *signUpOpts) {
		opts.reqOpts = append(opts.reqOpts, reqOpt)
	}
}

func (hit *HIT) SignUp(loginName, pwd string, appQName istructs.AppQName, opts ...signUpOptFunc) Login {
	hit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_User, signUpOpts.profileClusterID)
	hit.signUp(login, `{"DisplayName":"User Name"}`, signUpOpts.reqOpts...)
	return login
}

func getSignUpOpts(opts []signUpOptFunc) *signUpOpts {
	res := &signUpOpts{
		profileClusterID: istructs.MainClusterID,
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (hit *HIT) SignUpDevice(loginName, pwd string, appQName istructs.AppQName, opts ...signUpOptFunc) Login {
	hit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_Device, signUpOpts.profileClusterID)
	hit.signUp(login, "{}", signUpOpts.reqOpts...)
	return login
}

func (hit *HIT) GetCDocLoginID(login Login) int64 {
	hit.T.Helper()
	as, err := hit.IAppStructsProvider.AppStructs(istructs.AppQName_sys_registry)
	require.NoError(hit.T, err) // notest
	appWSID := coreutils.GetAppWSID(login.PseudoProfileWSID, as.WSAmount())
	body := fmt.Sprintf(`{"args":{"query":"select CDocLoginID from sys.LoginIdx where AppWSID = %d and AppIDLoginHash = '%s/%s'"}, "elements":[{"fields":["Result"]}]}`,
		appWSID, login.AppQName, signupin.GetLoginHash(login.Name))
	sys := hit.GetSystemPrincipal(istructs.AppQName_sys_registry)
	resp := hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.SqlQuery", body, utils.WithAuthorizeBy(sys.Token))
	m := map[string]interface{}{}
	require.NoError(hit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	return int64(m["CDocLoginID"].(float64))
}

func (hit *HIT) GetCDocWSKind(ws *AppWorkspace) (cdoc map[string]interface{}, id int64) {
	hit.T.Helper()
	return hit.getCDoc(ws.Owner.AppQName, ws.Kind, ws.WSID)
}

func (hit *HIT) getCDoc(appQName istructs.AppQName, qName appdef.QName, wsid istructs.WSID) (cdoc map[string]interface{}, id int64) {
	hit.T.Helper()
	body := bytes.NewBufferString(fmt.Sprintf(`{"args":{"Schema":"%s"},"elements":[{"fields":["sys.ID"`, qName))
	fields := []string{}
	as, err := hit.IAppStructsProvider.AppStructs(appQName)
	require.NoError(hit.T, err)
	cdocDef := as.AppDef().Def(qName)
	cdocDef.Fields(func(field appdef.IField) {
		switch field.Name() {
		case appdef.SystemField_ID, appdef.SystemField_QName, appdef.SystemField_IsActive:
			return
		}
		body.WriteString(fmt.Sprintf(`,"%s"`, field.Name()))
		fields = append(fields, field.Name())
	})
	body.WriteString("]}]}")
	sys := hit.GetSystemPrincipal(appQName)
	resp := hit.PostApp(appQName, wsid, "q.sys.Collection", body.String(), utils.WithAuthorizeBy(sys.Token))
	if len(resp.Sections) == 0 {
		hit.T.Fatalf("no CDoc<%s> at workspace id %d", qName.String(), wsid)
	}
	id = int64(resp.SectionRow()[0].(float64))
	cdoc = map[string]interface{}{}
	for i, fieldName := range fields {
		cdoc[fieldName] = resp.SectionRow()[i+1]
	}
	return
}

func (hit *HIT) GetCDocChildWorkspace(ws *AppWorkspace) (cdoc map[string]interface{}, id int64) {
	hit.T.Helper()
	return hit.getCDoc(ws.Owner.AppQName, authnz.QNameCDocChildWorkspace, ws.Owner.ProfileWSID)
}

func (hit *HIT) waitForWorkspace(wsName string, owner *Principal, respGetter func(owner *Principal, body string) *utils.FuncResponse) (ws *AppWorkspace) {
	const (
		// respect linter
		tmplNameIdx   = 3
		tmplParamsIdx = 4
		wsidIdx       = 5
		wsErrIdx      = 6
	)
	deadline := time.Now().Add(workspaceInitAwaitTimeout)
	logger.Verbose("workspace", wsName, "awaiting started")
	for time.Now().Before(deadline) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": "%s"
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
		require.NoError(hit.T, err)
		if len(wsError) > 0 {
			hit.T.Fatal(wsError)
		}
		return &AppWorkspace{
			WorkspaceDescriptor: WorkspaceDescriptor{
				WSParams: WSParams{
					Name:           resp.SectionRow()[0].(string),
					Kind:           wsKind,
					InitDataJSON:   resp.SectionRow()[2].(string),
					TemplateName:   resp.SectionRow()[tmplNameIdx].(string),
					TemplateParams: resp.SectionRow()[tmplParamsIdx].(string),
					ClusterID:      istructs.MainClusterID,
				},
				WSID:    wsid,
				WSError: wsError,
			},
			Owner: owner,
		}
	}
	hit.T.Fatalf("workspace %s is not initialized in an acceptable time", wsName)
	return ws
}

func (hit *HIT) WaitForWorkspace(wsName string, owner *Principal) (ws *AppWorkspace) {
	return hit.waitForWorkspace(wsName, owner, func(owner *Principal, body string) *utils.FuncResponse {
		return hit.PostProfile(owner, "q.sys.QueryChildWorkspaceByName", body)
	})
}

func (hit *HIT) WaitForChildWorkspace(parentWS *AppWorkspace, wsName string, owner *Principal) (ws *AppWorkspace) {
	return hit.waitForWorkspace(wsName, owner, func(owner *Principal, body string) *utils.FuncResponse {
		return hit.PostWS(parentWS, "q.sys.QueryChildWorkspaceByName", body)
	})
}

func DoNotFailOnTimeout() signInOptFunc {
	return func(opts *signInOpts) {
		opts.failOnTimeout = false
	}
}

func (hit *HIT) SignIn(login Login, optFuncs ...signInOptFunc) (prn *Principal) {
	hit.T.Helper()
	opts := &signInOpts{
		failOnTimeout: true,
	}
	for _, opt := range optFuncs {
		opt(opts)
	}
	deadline := time.Now().Add(workspaceInitAwaitTimeout)
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
		resp := hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.IssuePrincipalToken", body)
		profileWSID := istructs.WSID(resp.SectionRow()[1].(float64))
		wsError := resp.SectionRow()[2].(string)
		token := resp.SectionRow()[0].(string)
		if profileWSID == 0 && len(wsError) == 0 {
			time.Sleep(workspaceQueryDelay)
			continue
		}
		require.Empty(hit.T, wsError)
		require.NotEmpty(hit.T, token)
		return &Principal{
			Login:       login,
			Token:       token,
			ProfileWSID: profileWSID,
		}
	}
	if opts.failOnTimeout {
		hit.T.Fatal("user profile is not initialized in an acceptable time")
	}
	return nil
}

func (hit *HIT) InitChildWorkspace(wsd WSParams, owner *Principal) {
	hit.T.Helper()
	body := fmt.Sprintf(`{
		"args": {
			"WSName": "%s",
			"WSKind": "%s",
			"WSKindInitializationData": %q,
			"TemplateName": "%s",
			"TemplateParams": %q,
			"WSClusterID": %d
		}
	}`, wsd.Name, wsd.Kind.String(), wsd.InitDataJSON, wsd.TemplateName, wsd.TemplateParams, wsd.ClusterID)

	hit.PostProfile(owner, "c.sys.InitChildWorkspace", body)
}

func (hit *HIT) CreateWorkspace(wsp WSParams, owner *Principal) *AppWorkspace {
	hit.InitChildWorkspace(wsp, owner)
	ws := hit.WaitForWorkspace(wsp.Name, owner)
	require.Empty(hit.T, ws.WSError)
	return ws
}

// will be finalized automatically on hit.TearDown()
func (hit *HIT) SubscribeForN10n(ws *AppWorkspace, viewQName appdef.QName) chan int64 {
	n10n := make(chan int64)
	params := url.Values{}
	query := fmt.Sprintf(`{"SubjectLogin":"test_%d","ProjectionKey":[{"App":"%s","Projection":"%s","WS":%d}]}`,
		ws.WSID, ws.Owner.AppQName, viewQName, ws.WSID)
	params.Add("payload", query)
	httpResp, err := utils.FederationReq(hit.FederationURL(), fmt.Sprintf("n10n/channel?%s", params.Encode()), "",
		utils.WithLongPolling())
	require.NoError(hit.T, err)

	scanner := bufio.NewScanner(httpResp.HTTPResp.Body)
	scanner.Split(ScanSSE)

	// lets's wait for channelID
	if !scanner.Scan() {
		if !hit.T.Failed() {
			hit.T.Fatal("failed to get channelID on n10n subscription")
		}
	}
	messages := strings.Split(scanner.Text(), "\n")
	require.Equal(hit.T, "event: channelId", messages[0])
	require.True(hit.T, strings.HasPrefix(messages[1], "data: "))
	channelIDStr := strings.TrimPrefix(messages[1], "data: ")

	go func() {
		defer close(n10n)
		for scanner.Scan() {
			if httpResp.HTTPResp.Request.Context().Err() != nil {
				return
			}
			messages := strings.Split(scanner.Text(), "\n")
			if strings.TrimPrefix(messages[0], "event: ") == "channelId" {
				continue
			}
			offset, err := strconv.Atoi(strings.TrimPrefix(messages[1], "data: "))
			if err != nil {
				panic(err)
			}
			n10n <- int64(offset)
		}
	}()
	hit.lock.Lock() // need to lock because the hit instance is used in different goroutines in e.g. Test_Race_RestaurantIntenseUsage()
	hit.cleanups = append(hit.cleanups, func(hit *HIT) {
		body := fmt.Sprintf(`
			{
				"Channel": "%s",
				"ProjectionKey":[
					{
						"App": "%s",
						"Projection":"%s",
						"WS":%d
					}
				]
			}
		`, channelIDStr, ws.Owner.AppQName, viewQName, ws.WSID)
		params := url.Values{}
		params.Add("payload", string(body))
		hit.Get(fmt.Sprintf("n10n/unsubscribe?%s", params.Encode()))
		httpResp.HTTPResp.Body.Close()
		for range n10n {
		}
	})
	hit.lock.Unlock()
	return n10n
}

func IsCassandraStorage() bool {
	_, ok := os.LookupEnv("CASSANDRA_TESTS_ENABLED")
	return ok
}

func (hit *HIT) MetricsRequest(opts ...utils.ReqOptFunc) (resp string) {
	hit.T.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", hit.HeeusVM.MetricsServicePort())
	res, err := utils.Req(url, "", opts...)
	require.NoError(hit.T, err)
	return res.Body
}

func NewLogin(name, pwd string, appQName istructs.AppQName, subjectKind istructs.SubjectKindType, clusterID istructs.ClusterID) Login {
	pseudoWSID := utils.GetPseudoWSID(name, istructs.MainClusterID)
	return Login{name, pwd, pseudoWSID, appQName, subjectKind, clusterID, map[appdef.QName]map[string]interface{}{}}
}
