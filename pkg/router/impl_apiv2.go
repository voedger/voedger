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
		corsHandler(requestHandlerV2_auth_login(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("auth login")

	// auth/login: /api/v2/apps/{owner}/{app}/auth/login
	// [~server.apiv2.auth/cmp.routerRefreshHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/auth/refresh",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_auth_refresh(s.requestSender, s.numsAppsWorkspaces))).
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

func requestHandlerV2_auth_refresh(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Auth_Refresh)
		busRequest.Method = http.MethodGet
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_auth_login(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}

		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Auth_Login)
		busRequest.Method = http.MethodGet
		queryParams := map[string]string{}
		queryParams["arg"] = string(busRequest.Body)
		busRequest.Query = queryParams
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
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
		url := fmt.Sprintf("/api/v2/apps/%s/%s/workspaces/%d/commands/registry.CreateLogin",
			busRequest.AppQName.Owner(), busRequest.AppQName.Name(), pseudoWSID)
		wsKindInitData := fmt.Sprintf(`{"DisplayName":%q}`, displayName)
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":%q,"ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
			email, busRequest.AppQName, istructs.SubjectKind_User, wsKindInitData, istructs.CurrentClusterID(), pwd)
		sysToken, err := payloads.GetSystemPrincipalToken(iTokens, busRequest.AppQName)
		if err != nil {
			ReplyCommonError(rw, "failed to issue sys token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := federation.Func(url, body,
			coreutils.WithAuthorizeBy(sysToken),
			coreutils.WithMethod(http.MethodPost),
		)
		rw.Header().Set(coreutils.ContentType, coreutils.ContentType_ApplicationJSON)
		rw.WriteHeader(resp.HTTPResp.StatusCode)
		if err != nil {
			sysError := resp.SysError.ToJSON_APIV2()
			logger.Error("registry.Createlogin failed:" + sysError)
			if !writeResponse(rw, sysError) {
				logger.Error("failed to send registry.CreateLogin error")
			}
			return
		}
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
