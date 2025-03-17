/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors/query2"
)

func (s *httpService) registerHandlersV2() {
	// create: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/docs/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table),
		corsHandler(requestHandlerV2_table(s.requestSender, query2.ApiPath_Docs, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("create")

	// update, deactivate, read single doc: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/docs/{%s}.{%s}/{%s:[0-9]+}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table,
		URLPlaceholder_id),
		corsHandler(requestHandlerV2_table(s.requestSender, query2.ApiPath_Docs, s.numsAppsWorkspaces))).
		Methods(http.MethodPatch, http.MethodDelete, http.MethodGet).Name("update")

	// read collection: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/cdocs/{pkg}.{table}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/cdocs/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table),
		corsHandler(requestHandlerV2_table(s.requestSender, query2.ApiPath_CDocs, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("read collection")

	// execute cmd: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/commands/{pkg}.{command}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/commands/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_command),
		corsHandler(requestHandlerV2_extension(s.requestSender, query2.ApiPath_Commands, s.numsAppsWorkspaces))).
		Methods(http.MethodPost).Name("exec cmd")

	// execute query: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/queries/{pkg}.{query}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/queries/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_query),
		corsHandler(requestHandlerV2_extension(s.requestSender, query2.ApiPath_Queries, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("exec query")

	// view: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/views/{pkg}.{view}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/views/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_view),
		corsHandler(requestHandlerV2_view(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("view")

	// blobs: create /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/blobs",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
		corsHandler(requestHandlerV2_blobs())).
		Methods(http.MethodPost).Name("blobs read")

	// blobs: replace, upload, download, update meta: /api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/workspaces/{%s:[0-9]+}/blobs/{%s:[a-zA-Z0-9-_]+}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_blobIDOrSUUID),
		corsHandler(requestHandlerV2_blobs())).
		Methods(http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete).Name("blobs anything but read")

	// schemas: get workspace schema: /api/v2/users/{owner}/apps/{app}/schemas
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/schemas",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_schemas(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas")

	// schemas, workspace roles: get workspace schema: /api/v2/users/{owner}/apps/{app}/schemas/{pkg}.{workspace}/roles
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/schemas/{%s}.{%s}/roles",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspace),
		corsHandler(requestHandlerV2_schemas_wsRoles(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas, workspace roles")

	// schemas, workspace role: get workspace schema: /api/v2/users/{owner}/apps/{app}/schemas/{pkg}.{workspace}/roles/{pkg}.{role}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/users/{%s}/apps/{%s}/schemas/{%s}.{%s}/roles/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspace, URLPlaceholder_rolePkg, URLPlaceholder_role),
		corsHandler(requestHandlerV2_schemas_wsRole(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas, workspace role")
}

func requestHandlerV2_schemas(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		busRequest.IsAPIV2 = true
		busRequest.ApiPath = int(query2.ApiPaths_Schemas)
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
		busRequest.ApiPath = int(query2.ApiPaths_Schemas_WorkspaceRoles)
		busRequest.WorkspaceQName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_workspace])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
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
		busRequest.ApiPath = int(query2.ApiPaths_Schemas_WorkspaceRoles)
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
		busRequest.ApiPath = int(query2.ApiPath_Views)
		busRequest.QName = appdef.NewQName(vars[URLPlaceholder_pkg], vars[URLPlaceholder_view])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	}
}

func requestHandlerV2_extension(reqSender bus.IRequestSender, apiPath query2.ApiPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		entity := ""
		switch apiPath {
		case query2.ApiPath_Commands:
			entity = vars[URLPlaceholder_command]
		case query2.ApiPath_Queries:
			entity = vars[URLPlaceholder_query]
		}
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}

		busRequest.IsAPIV2 = true
		busRequest.ApiPath = int(apiPath)
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

func requestHandlerV2_table(reqSender bus.IRequestSender, apiPath query2.ApiPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		switch req.Method {
		case http.MethodGet:
			recordIDStr, isSingleRecord := vars[URLPlaceholder_id]
			isReadCollection := !isSingleRecord
			_ = isReadCollection
			_ = recordIDStr
		case http.MethodPost:
		case http.MethodPatch:
		case http.MethodDelete:
		}
		// note: request lead to create -> 201 Created
		busRequest, ok := createBusRequest(req.Method, req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		busRequest.IsAPIV2 = true
		busRequest.ApiPath = int(apiPath)
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
