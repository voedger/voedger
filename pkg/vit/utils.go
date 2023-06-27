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
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/authnz/signupin"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func (vit *VIT) GetBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobID int64, token string) *BLOB {
	vit.T.Helper()
	resp, err := coreutils.FederationReq(vit.IFederation.URL(), fmt.Sprintf(`blob/%s/%d/%d`, appQName.String(), wsid, blobID), "", coreutils.WithAuthorizeBy(token))
	require.NoError(vit.T, err)
	contentDisposition := resp.HTTPResp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	require.NoError(vit.T, err)
	return &BLOB{
		Content:  []byte(resp.Body),
		Name:     params["filename"],
		MimeType: resp.HTTPResp.Header.Get(coreutils.ContentType),
	}
}

func (vit *VIT) signUp(login Login, wsKindInitData string, opts ...coreutils.ReqOptFunc) {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":%q,"ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
		login.Name, login.AppQName.String(), login.subjectKind, wsKindInitData, login.clusterID, login.Pwd)
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.CreateLogin", body, opts...)
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

func (vit *VIT) SignUp(loginName, pwd string, appQName istructs.AppQName, opts ...signUpOptFunc) Login {
	vit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_User, signUpOpts.profileClusterID)
	vit.signUp(login, `{"DisplayName":"User Name"}`, signUpOpts.reqOpts...)
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

func (vit *VIT) SignUpDevice(loginName, pwd string, appQName istructs.AppQName, opts ...signUpOptFunc) Login {
	vit.T.Helper()
	signUpOpts := getSignUpOpts(opts)
	login := NewLogin(loginName, pwd, appQName, istructs.SubjectKind_Device, signUpOpts.profileClusterID)
	vit.signUp(login, "{}", signUpOpts.reqOpts...)
	return login
}

func (vit *VIT) GetCDocLoginID(login Login) int64 {
	vit.T.Helper()
	as, err := vit.IAppStructsProvider.AppStructs(istructs.AppQName_sys_registry)
	require.NoError(vit.T, err) // notest
	appWSID := coreutils.GetAppWSID(login.PseudoProfileWSID, as.WSAmount())
	body := fmt.Sprintf(`{"args":{"query":"select CDocLoginID from sys.LoginIdx where AppWSID = %d and AppIDLoginHash = '%s/%s'"}, "elements":[{"fields":["Result"]}]}`,
		appWSID, login.AppQName, signupin.GetLoginHash(login.Name))
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

func (vit *VIT) getCDoc(appQName istructs.AppQName, qName appdef.QName, wsid istructs.WSID) (cdoc map[string]interface{}, id int64) {
	vit.T.Helper()
	body := bytes.NewBufferString(fmt.Sprintf(`{"args":{"Schema":"%s"},"elements":[{"fields":["sys.ID"`, qName))
	fields := []string{}
	as, err := vit.IAppStructsProvider.AppStructs(appQName)
	require.NoError(vit.T, err)
	if def := as.AppDef().CDoc(qName); def != nil {
		def.Fields(func(field appdef.IField) {
			switch field.Name() {
			case appdef.SystemField_ID, appdef.SystemField_QName, appdef.SystemField_IsActive:
				return
			}
			body.WriteString(fmt.Sprintf(`,"%s"`, field.Name()))
			fields = append(fields, field.Name())
		})
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

func (vit *VIT) waitForWorkspace(wsName string, owner *Principal, respGetter func(owner *Principal, body string) *coreutils.FuncResponse) (ws *AppWorkspace) {
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
		require.NoError(vit.T, err)
		if len(wsError) > 0 {
			vit.T.Fatal(wsError)
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
	vit.T.Fatalf("workspace %s is not initialized in an acceptable time", wsName)
	return ws
}

func (vit *VIT) WaitForWorkspace(wsName string, owner *Principal) (ws *AppWorkspace) {
	return vit.waitForWorkspace(wsName, owner, func(owner *Principal, body string) *coreutils.FuncResponse {
		return vit.PostProfile(owner, "q.sys.QueryChildWorkspaceByName", body)
	})
}

func (vit *VIT) WaitForChildWorkspace(parentWS *AppWorkspace, wsName string, owner *Principal) (ws *AppWorkspace) {
	return vit.waitForWorkspace(wsName, owner, func(owner *Principal, body string) *coreutils.FuncResponse {
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
		resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.IssuePrincipalToken", body)
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

func (vit *VIT) InitChildWorkspace(wsd WSParams, owner *Principal) {
	vit.T.Helper()
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

	vit.PostProfile(owner, "c.sys.InitChildWorkspace", body)
}

func (vit *VIT) CreateWorkspace(wsp WSParams, owner *Principal) *AppWorkspace {
	vit.InitChildWorkspace(wsp, owner)
	ws := vit.WaitForWorkspace(wsp.Name, owner)
	require.Empty(vit.T, ws.WSError)
	return ws
}

// will be finalized automatically on vit.TearDown()
func (vit *VIT) SubscribeForN10n(ws *AppWorkspace, viewQName appdef.QName) chan int64 {
	n10n := make(chan int64)
	params := url.Values{}
	query := fmt.Sprintf(`{"SubjectLogin":"test_%d","ProjectionKey":[{"App":"%s","Projection":"%s","WS":%d}]}`,
		ws.WSID, ws.Owner.AppQName, viewQName, ws.WSID)
	params.Add("payload", query)
	httpResp, err := coreutils.FederationReq(vit.IFederation.URL(), fmt.Sprintf("n10n/channel?%s", params.Encode()), "",
		coreutils.WithLongPolling())
	require.NoError(vit.T, err)

	scanner := bufio.NewScanner(httpResp.HTTPResp.Body)
	scanner.Split(ScanSSE)

	// lets's wait for channelID
	if !scanner.Scan() {
		if !vit.T.Failed() {
			vit.T.Fatal("failed to get channelID on n10n subscription")
		}
	}
	messages := strings.Split(scanner.Text(), "\n")
	require.Equal(vit.T, "event: channelId", messages[0])
	require.True(vit.T, strings.HasPrefix(messages[1], "data: "))
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
	vit.lock.Lock() // need to lock because the vit instance is used in different goroutines in e.g. Test_Race_RestaurantIntenseUsage()
	vit.cleanups = append(vit.cleanups, func(vit *VIT) {
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
		vit.Get(fmt.Sprintf("n10n/unsubscribe?%s", params.Encode()))
		httpResp.HTTPResp.Body.Close()
		for range n10n {
		}
	})
	vit.lock.Unlock()
	return n10n
}

func (vit *VIT) MetricsRequest(opts ...coreutils.ReqOptFunc) (resp string) {
	vit.T.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", vit.VoedgerVM.MetricsServicePort())
	res, err := coreutils.Req(url, "", opts...)
	require.NoError(vit.T, err)
	return res.Body
}

func NewLogin(name, pwd string, appQName istructs.AppQName, subjectKind istructs.SubjectKindType, clusterID istructs.ClusterID) Login {
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, name, istructs.MainClusterID)
	return Login{name, pwd, pseudoWSID, appQName, subjectKind, clusterID, map[appdef.QName]map[string]interface{}{}}
}
