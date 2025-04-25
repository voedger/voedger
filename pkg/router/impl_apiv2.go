/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func (s *httpService) registerHandlersV2() {
	// create: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{pkg}.{table}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/docs/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table),
		corsHandler(requestHandlerV2_table(s.requestSender, processors.APIPath_Docs, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("create")

	// update, deactivate, read single doc: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/docs/{%s}.{%s}/{%s:[0-9]+}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table,
		URLPlaceholder_id),
		corsHandler(requestHandlerV2_table(s.requestSender, processors.APIPath_Docs, s.numsAppsWorkspaces))).
		Methods(http.MethodPatch, http.MethodDelete, http.MethodGet).Name("update or read single")

	// read collection: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/cdocs/{pkg}.{table}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/cdocs/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table),
		corsHandler(requestHandlerV2_table(s.requestSender, processors.APIPath_CDocs, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("read collection")

	// execute cmd: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/commands/{pkg}.{command}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/commands/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_command),
		corsHandler(requestHandlerV2_extension(s.requestSender, processors.APIPath_Commands, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("exec cmd")

	// execute query: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/queries/{pkg}.{query}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/queries/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_query),
		corsHandler(requestHandlerV2_extension(s.requestSender, processors.APIPath_Queries, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("exec query")

	// view: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/views/{pkg}.{view}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/views/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_view),
		corsHandler(requestHandlerV2_view(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("view")

	// blobs: create /api/v2/apps/{owner}/{app}/workspaces/{wsid}/blobs
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/blobs",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
		corsHandler(requestHandlerV2_blobs())).
		Methods(http.MethodPost).Name("blobs read")

	// blobs: replace, upload, download, update meta: /api/v2/apps/{owner}/{app}/workspaces/{wsid}/blobs/{blobId}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s:[0-9]+}/blobs/{%s:[a-zA-Z0-9-_]+}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_blobIDOrSUUID),
		corsHandler(requestHandlerV2_blobs())).
		Methods(http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete).Name("blobs anything but read")

	// schemas: get workspace schema: /api/v2/apps/{owner}/{app}/schemas
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/schemas",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_schemas(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas")

	// schemas, workspace roles: get workspace schema: /api/v2/apps/{owner}/{app}/schemas/{pkg}.{workspace}/roles
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/schemas/{%s}.{%s}/roles",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspace),
		corsHandler(requestHandlerV2_schemas_wsRoles(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas, workspace roles")

	// schemas, workspace role: get workspace schema: /api/v2/apps/{owner}/{app}/schemas/{pkg}.{workspace}/roles/{pkg}.{role}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/schemas/{%s}.{%s}/roles/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspace, URLPlaceholder_rolePkg, URLPlaceholder_role),
		corsHandler(requestHandlerV2_schemas_wsRole(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas, workspace role")

	// auth/login: /api/v2/apps/{owner}/{app}/auth/login
	// [~server.apiv2.auth/cmp.routerLoginPathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/auth/login",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_auth_login(s.federation, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("auth login")

	// auth/login: /api/v2/apps/{owner}/{app}/auth/login
	// [~server.apiv2.auth/cmp.routerRefreshHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/auth/refresh",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_auth_refresh(s.iTokens, s.federation, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("auth refresh")

	// create user /api/v2/apps/{owner}/{app}/users
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/users",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_create_user(s.numsAppsWorkspaces, s.iTokens, s.federation))).
		Methods(http.MethodPost).Name("create user")
}

func requestHandlerV2_schemas(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPaths_Schema)
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_schemas_wsRoles(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Schemas_WorkspaceRoles)
		busRequest.WorkspaceQName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_workspace])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

// [~server.apiv2.auth/cmp.provideAuthRefreshHandler~impl]
func requestHandlerV2_auth_refresh(iTokens itokens.ITokens, federation federation.IFederation,
	numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		body := map[string]interface{}{}
		if err := json.Unmarshal(busRequest.Body, &body); err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := bus.GetPrincipalToken(busRequest)
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		if len(token) == 0 {
			ReplyCommonError(rw, "authorization header is empty", http.StatusUnauthorized)
			return
		}

		var principalPayload payloads.PrincipalPayload
		if _, err = iTokens.ValidateToken(token, &principalPayload); err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusUnauthorized)
			return
		}

		url := fmt.Sprintf("api/v2/apps/%s/%s/workspaces/%d/queries/sys.RefreshPrincipalToken", busRequest.AppQName.Owner(), busRequest.AppQName.Name(),
			principalPayload.ProfileWSID)
		resp, err := federation.Query(url, coreutils.WithAuthorizeBy(token))
		if err != nil {
			if funcError, ok := err.(coreutils.FuncError); ok {
				ReplyCommonError(rw, funcError.ToJSON_APIV2(), funcError.HTTPStatus)
			} else {
				ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if resp.IsEmpty() {
			ReplyCommonError(rw, "sys.RefreshPrincipalToken response is empty", http.StatusInternalServerError)
			return
		}

		newToken := resp.QPv2Response.Result()["NewPrincipalToken"].(string)
		if len(newToken) == 0 {
			ReplyCommonError(rw, "sys.RefreshPrincipalToken response contains empty token", http.StatusInternalServerError)
			return
		}

		payload := payloads.PrincipalPayload{}
		gp, err := iTokens.ValidateToken(newToken, &payload)
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		expiresIn := gp.Duration.Seconds()
		json := fmt.Sprintf(`{"PrincipalToken": "%s", "ExpiresIn": %d, "WSID": %d}`, newToken, int(expiresIn), principalPayload.ProfileWSID)
		rw.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
		rw.WriteHeader(http.StatusOK)
		writeResponse(rw, json)
	}
}

// [~server.apiv2.auth/cmp.provideAuthLoginHandler~impl]
// wrong to handle it by QPv2 because 10 simultaneous calls of aut/login -> each would be fail to call registry.IssuePrincipalToken
func requestHandlerV2_auth_login(federation federation.IFederation, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}

		body := map[string]interface{}{}
		if err := json.Unmarshal(busRequest.Body, &body); err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}

		args := coreutils.MapObject(body)
		login, _, err := args.AsString("Login")
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		password, _, err := args.AsString("Password")
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}

		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

		url := fmt.Sprintf(`api/v2/apps/%s/%s/workspaces/%d/queries/registry.IssuePrincipalToken?arg=%s`,
			istructs.SysOwner, istructs.AppQName_sys_registry.Name(), pseudoWSID,
			url.QueryEscape(fmt.Sprintf(`{"Login":"%s", "Password":"%s", "AppName": "%s"}`, login, password, busRequest.AppQName)))
		resp, err := federation.Query(url)
		if err != nil {
			if funcError, ok := err.(coreutils.FuncError); ok {
				ReplyCommonError(rw, funcError.ToJSON_APIV2(), funcError.HTTPStatus)
			} else {
				ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if resp.IsEmpty() {
			ReplyCommonError(rw, "registry.IssuePrincipalToken response is empty", http.StatusInternalServerError)
			return
		}
		token := resp.QPv2Response.Result()["PrincipalToken"].(string)
		wsError := resp.QPv2Response.Result()["WSError"].(string)
		wsid := resp.QPv2Response.Result()["WSID"].(float64)

		if len(wsError) > 0 {
			ReplyCommonError(rw, "the login profile is created with an error: "+wsError, http.StatusInternalServerError)
		}

		expiresIn := authnz.DefaultPrincipalTokenExpiration.Seconds()
		json := fmt.Sprintf(`{"PrincipalToken": "%s","ExpiresIn": %d,"WSID": %d}`, token, int(expiresIn), int(wsid))
		rw.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
		rw.WriteHeader(http.StatusOK)
		writeResponse(rw, json)
	}
}

func requestHandlerV2_create_user(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces,
	iTokens itokens.ITokens, federation federation.IFederation) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		verifiedEmailToken, displayName, pwd, err := parseCreateLoginArgs(string(busRequest.Body))
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		payload := payloads.VerifiedValuePayload{}
		_, err = iTokens.ValidateToken(verifiedEmailToken, &payload)
		if err != nil {
			ReplyCommonError(rw, fmt.Sprintf("VerifiedEmailToken validation failed: %s", err.Error()), http.StatusBadRequest)
			return
		}
		email := payload.Value.(string)
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, email, istructs.CurrentClusterID())
		url := fmt.Sprintf("api/v2/apps/sys/registry/workspaces/%d/commands/registry.CreateEmailLogin", pseudoWSID)
		wsKindInitData := fmt.Sprintf(`{"DisplayName":%q}`, displayName)
		body := fmt.Sprintf(`{"args":{"Email":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":%q,"ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
			verifiedEmailToken, busRequest.AppQName, istructs.SubjectKind_User, wsKindInitData, istructs.CurrentClusterID(), pwd)
		sysToken, err := payloads.GetSystemPrincipalToken(iTokens, istructs.AppQName_sys_registry)
		if err != nil {
			ReplyCommonError(rw, "failed to issue sys token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := federation.Func(url, body,
			coreutils.WithAuthorizeBy(sysToken),
			coreutils.WithMethod(http.MethodPost),
		)
		rw.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
		if err != nil {
			funcErr := err.(coreutils.FuncError)
			rw.WriteHeader(funcErr.HTTPStatus)
			sysError := funcErr.ToJSON_APIV2()
			logger.Error("registry.Createlogin failed:" + sysError)
			if !writeResponse(rw, sysError) {
				logger.Error("failed to send registry.CreateLogin error")
			}
			return
		}
		rw.WriteHeader(http.StatusCreated)
		if !writeResponse(rw, resp.Body) {
			logger.Error("failed to forward registry.CreateLogin response: " + resp.Body)
		}
	}
}

func parseCreateLoginArgs(body string) (verifiedlEmailToken, displayName, pwd string, err error) {
	args := coreutils.MapObject{}
	if err = json.Unmarshal([]byte(body), &args); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal body: %w:\n%s", err, body)
	}
	ok := false
	verifiedlEmailToken, ok, err = args.AsString("VerifiedEmailToken")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("VerifiedEmailToken field missing")
	}
	displayName, ok, err = args.AsString("DisplayName")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("DisplayName field missing")
	}
	pwd, ok, err = args.AsString("Password")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("Password field missing")
	}
	return
}

func requestHandlerV2_schemas_wsRole(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Schemas_WorkspaceRole)
		busRequest.WorkspaceQName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_workspace])
		busRequest.QName = appdef.NewQName(vars[URLPlaceholder_rolePkg], vars[URLPlaceholder_role])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_view(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}

		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Views)
		busRequest.QName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_view])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_extension(reqSender bus.IRequestSender, apiPath processors.APIPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		entity := ""
		switch apiPath {
		case processors.APIPath_Commands:
			entity = vars[URLPlaceholder_command]
		case processors.APIPath_Queries:
			entity = vars[URLPlaceholder_query]
		}
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}

		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(apiPath)
		busRequest.QName = appdef.NewQName(vars[URLPlaceholder_pkg], entity)
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_blobs() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		blobID, notBLOBCreate := vars[URLPlaceholder_blobIDOrSUUID]
		isBLOBCreate := !notBLOBCreate
		_ = blobID
		_ = isBLOBCreate
		// note: request lead to create -> 201 Created
		switch req.Method {
		case http.MethodGet:
		case http.MethodPost:
		case http.MethodPatch:
		case http.MethodDelete:
		case http.MethodPut:
		}
		writeNotImplemented(resp)
	}
}

func requestHandlerV2_table(reqSender bus.IRequestSender, apiPath processors.APIPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		recordIDStr := vars[URLPlaceholder_id]

		// switch req.Method {
		// case http.MethodGet:
		// 	recordIDStr, isSingleDoc := vars[URLPlaceholder_id]
		// 	isReadCollection := !isSingleDoc
		// 	_ = isReadCollection
		// 	_ = recordIDStr
		// case http.MethodPost:
		// case http.MethodPatch:
		// case http.MethodDelete:
		// }
		// note: request lead to create -> 201 Created
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		if len(recordIDStr) > 0 {
			docID, err := strconv.ParseUint(recordIDStr, utils.DecimalBase, utils.BitSize64)
			if err != nil {
				// notest
				panic(err)
			}
			busRequest.DocID = istructs.IDType(docID)
		}
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(apiPath)
		busRequest.QName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_table])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func sendRequestAndReadResponse(req *http.Request, busRequest bus.Request, reqSender bus.IRequestSender, rw http.ResponseWriter) {
	// req's BaseContext is router service's context. See service.Start()
	// router app closing or client disconnected -> req.Context() is done
	// will create new cancellable context and cancel it if http section send is failed.
	// requestCtx.Done() -> SendRequest implementation will notify the handler that the consumer has left us
	requestCtx, cancel := context.WithCancel(req.Context())
	defer cancel() // to avoid context leak
	respCh, respMeta, respErr, err := reqSender.SendRequest(requestCtx, busRequest)
	if err != nil {
		logger.Error("sending request to VVM on", busRequest.QName, "is failed:", err, ". Body:\n", string(busRequest.Body))
		status := http.StatusInternalServerError
		if errors.Is(err, bus.ErrSendTimeoutExpired) {
			status = http.StatusServiceUnavailable
		}
		WriteTextResponse(rw, err.Error(), status)
		return
	}

	initResponse(rw, respMeta.ContentType, respMeta.StatusCode)
	reply_v2(requestCtx, rw, respCh, respErr, cancel, respMeta.Mode())
}
