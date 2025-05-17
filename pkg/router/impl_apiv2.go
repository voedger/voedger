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
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
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
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/blobs",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
		corsHandler(requestHandlerV2_blobs_create(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodPost).Name("blobs create")

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
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/auth/login",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_auth_login(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("auth login")

	// auth/login: /api/v2/apps/{owner}/{app}/auth/login
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/auth/refresh",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_auth_refresh(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("auth refresh")

	// create user /api/v2/apps/{owner}/{app}/users
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/users",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_create_user(s.numsAppsWorkspaces, s.iTokens, s.federation))).
		Methods(http.MethodPost).Name("create user")

	// create device /api/v2/apps/{owner}/{app}/devices
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/devices",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_create_device(s.numsAppsWorkspaces, s.federation))).
		Methods(http.MethodPost).Name("create device")

	// /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{doc}/blobs/{field}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/docs/{%s}.{%s}/blobs/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table, URLPlaceholder_field),
		corsHandler(requestHandlerV2_blobs_create(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodPost).Name("blobs create")

	// GET /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}/blobs/{fieldName}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/docs/{%s}.{%s}/{%s}/blobs/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg,
		URLPlaceholder_table, URLPlaceholder_id, URLPlaceholder_field),
		corsHandler(requestHandlerV2_blobs_read(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodGet).Name("blobs read")
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

// [~server.users/cmp.router.UsersCreatePathHandler~impl]
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
		if err != nil {
			replyErr(rw, err)
			return
		}
		ReplyJSON(rw, resp.Body, http.StatusCreated)
	}
}

// [~server.devices/cmp.routerDevicesCreatePathHandler~impl]
func requestHandlerV2_create_device(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, federation federation.IFederation) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		if len(busRequest.Body) > 0 {
			ReplyCommonError(rw, "unexpected body", http.StatusBadRequest)
			return
		}
		login, pwd := coreutils.DeviceRandomLoginPwd()
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
		url := fmt.Sprintf("api/v2/apps/sys/registry/workspaces/%d/commands/registry.CreateLogin", pseudoWSID)
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
			login, busRequest.AppQName, istructs.SubjectKind_Device, istructs.CurrentClusterID(), pwd)
		_, err := federation.Func(url, body, coreutils.WithMethod(http.MethodPost))
		if err != nil {
			replyErr(rw, err)
			return
		}
		result := fmt.Sprintf(`{"Login":"%s","Password":"%s"}`, login, pwd)
		ReplyJSON(rw, result, http.StatusCreated)
	}
}

// [~server.apiv2.auth/cmp.routerRefreshHandler~impl]
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

// [~server.apiv2.auth/cmp.routerLoginPathHandler~impl]
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
		queryParams["args"] = string(busRequest.Body)
		busRequest.Query = queryParams
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_blobs_read(blobRequestHandler blobprocessor.IRequestHandler,
	requestSender bus.IRequestSender) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, rw)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		ownerRecord := appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_table])
		ownerRecordField := vars[URLPlaceholder_field]
		ownerID, err := strconv.ParseUint(vars[URLPlaceholder_id], utils.DecimalBase, utils.BitSize64)
		if err != nil {
			// notest: checked by router url rule
			panic(err)
		}
		if !blobRequestHandler.HandleWrite_V2(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(rw), req.Body, func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, requestSender, ownerRecord, ownerRecordField) {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
}

func requestHandlerV2_blobs_create(blobRequestHandler blobprocessor.IRequestHandler,
	requestSender bus.IRequestSender) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, rw)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		ownerRecord := appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_table])
		ownerRecordField := vars[URLPlaceholder_field]
		if !blobRequestHandler.HandleWrite_V2(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(rw), req.Body, func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, requestSender, ownerRecord, ownerRecordField) {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
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

func parseCreateLoginArgs(body string) (verifiedEmailToken, displayName, pwd string, err error) {
	args := coreutils.MapObject{}
	if err = json.Unmarshal([]byte(body), &args); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal body: %w:\n%s", err, body)
	}
	ok := false
	verifiedEmailToken, ok, err = args.AsString("VerifiedEmailToken")
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
		return "", "", "", errors.New("displayName field missing")
	}
	pwd, ok, err = args.AsString("Password")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("password field missing")
	}
	return
}

func replyErr(rw http.ResponseWriter, err error) {
	var sysError coreutils.SysError
	if errors.As(err, &sysError) {
		ReplyJSON(rw, sysError.ToJSON_APIV2(), sysError.HTTPStatus)
	} else {
		ReplyCommonError(rw, err.Error(), http.StatusInternalServerError)
	}
}
