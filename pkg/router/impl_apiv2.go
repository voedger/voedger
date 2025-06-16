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
	"time"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
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
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspaceName),
		corsHandler(requestHandlerV2_schemas_wsRoles(s.requestSender, s.numsAppsWorkspaces))).
		Methods(http.MethodGet).Name("schemas, workspace roles")

	// schemas, workspace role: get workspace schema: /api/v2/apps/{owner}/{app}/schemas/{pkg}.{workspace}/roles/{pkg}.{role}
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/schemas/{%s}.{%s}/roles/{%s}.{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_pkg, URLPlaceholder_workspaceName, URLPlaceholder_rolePkg, URLPlaceholder_role),
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

	// change password user /api/v2/apps/{owner}/{app}/users/change-password
	// [~server.users/cmp.routerUsersChangePasswordPathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/users/change-password",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_changePassword(s.numsAppsWorkspaces, s.federation))).
		Methods(http.MethodPost).Name("change password")

	// create device /api/v2/apps/{owner}/{app}/devices
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/devices",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_create_device(s.numsAppsWorkspaces, s.federation))).
		Methods(http.MethodPost).Name("create device")

	// blob create /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{doc}/blobs/{field}
	// [~server.apiv2.blobs/cmp.routerBlobsCreatePathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/docs/{%s}.{%s}/blobs/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg, URLPlaceholder_table, URLPlaceholder_field),
		corsHandler(requestHandlerV2_blobs_create(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodPost).Name("blobs create")

	// blob read GET /api/v2/apps/{owner}/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}/blobs/{fieldName}
	// [~server.apiv2.blobs/cmp.routerBlobsReadPathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/docs/{%s}.{%s}/{%s}/blobs/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_pkg,
		URLPlaceholder_table, URLPlaceholder_id, URLPlaceholder_field),
		corsHandler(requestHandlerV2_blobs_read(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodGet).Name("blobs read")

	// temp blob create /api/v2/apps/{owner}/{app}/workspaces/{wsid}/tblobs
	// [~server.apiv2.tblobs/cmp.routerTBlobsCreatePathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/tblobs",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
		corsHandler(requestHandlerV2_tempblobs_create(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodPost).Name("temp blobs create")

	// temp blob read GET /api/v2/apps/{owner}/{app}/workspaces/{wsid}/tblobs/{suuid}
	// [~server.apiv2.blobs/cmp.routerTBlobsReadPathHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/workspaces/{%s}/tblobs/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_blobIDOrSUUID),
		corsHandler(requestHandlerV2_tempblobs_read(s.blobRequestHandler, s.requestSender))).
		Methods(http.MethodGet).Name("temp blobs read")

	// notifications subscribe+watch /api/v2/apps/{owner}/{app}/notifications
	// [~server.n10n/cmp.routerCreateChannelHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/notifications",
		URLPlaceholder_appOwner, URLPlaceholder_appName),
		corsHandler(requestHandlerV2_notifications_subscribeAndWatch(s.numsAppsWorkspaces, s.n10n, s.appTokensFactory))).
		Methods(http.MethodPost).Name("notifications subscribe + watch")

	// notifications unsubscribe /api/v2/apps/{owner}/{app}/notifications/{channelId}/workspaces/{wsid}/subscriptions/{entity}
	// [~server.n10n/cmp.routerUnsubscribeHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/notifications/{%s}/workspaces/{%s}/subscriptions/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_channelID, URLPlaceholder_wsid, URLPlaceholder_view),
		corsHandler(requestHandlerV2_notifications(s.numsAppsWorkspaces, s.n10n, s.appTokensFactory))).
		Methods(http.MethodDelete).Name("notifications unsubscribe")

	// notifications subscribe to an extra view /api/v2/apps/{owner}/{app}/notifications/{channelId}/workspaces/{wsid}/subscriptions/{entity}
	// [~server.n10n/cmp.routerAddSubscriptionHandler~impl]
	s.router.HandleFunc(fmt.Sprintf("/api/v2/apps/{%s}/{%s}/notifications/{%s}/workspaces/{%s}/subscriptions/{%s}",
		URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_channelID, URLPlaceholder_wsid, URLPlaceholder_view),
		corsHandler(requestHandlerV2_notifications(s.numsAppsWorkspaces, s.n10n, s.appTokensFactory))).
		Methods(http.MethodPut).Name("notifications subscribe to an extra view")
}

func requestHandlerV2_schemas(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPaths_Schema)
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

func requestHandlerV2_schemas_wsRoles(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Schemas_WorkspaceRoles)
		busRequest.WorkspaceQName = appdef.NewQName(data.vars[URLPlaceholder_pkg], data.vars[URLPlaceholder_workspaceName])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

// [~server.users/cmp.routerUsersChangePasswordPathHandler~impl]
func requestHandlerV2_changePassword(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, federation federation.IFederation) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		login, oldPassword, newPassword, err := parseChangePasswordArgs(string(busRequest.Body))
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"%s","NewPassword":"%s"}}`,
			login, busRequest.AppQName, oldPassword, newPassword)
		url := fmt.Sprintf("api/v2/apps/sys/registry/workspaces/%d/commands/registry.ChangePassword", pseudoWSID)
		if _, err = federation.Func(url, body, coreutils.WithMethod(http.MethodPost), coreutils.WithDiscardResponse()); err != nil { // null auth
			replyErr(rw, err)
			return
		}
		ReplyJSON(rw, "", http.StatusOK)
	})
}

// [~server.users/cmp.router.UsersCreatePathHandler~impl]
func requestHandlerV2_create_user(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces,
	iTokens itokens.ITokens, federation federation.IFederation) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		verifiedEmailToken, displayName, pwd, err := parseCreateLoginArgs(string(busRequest.Body))
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}
		payload := payloads.VerifiedValuePayload{}
		_, err = iTokens.ValidateToken(verifiedEmailToken, &payload)
		if err != nil {
			ReplyCommonError(rw, fmt.Sprintf("verifiedEmailToken validation failed: %s", err.Error()), http.StatusBadRequest)
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
			// notest
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
	})
}

func authorize(appTokensFactory payloads.IAppTokensFactory, busRequest bus.Request) (principalPayload payloads.PrincipalPayload, err error) {
	principalToken, err := bus.GetPrincipalToken(busRequest)
	if err != nil {
		return principalPayload, err
	}
	appTokens := appTokensFactory.New(busRequest.AppQName)
	_, err = appTokens.ValidateToken(principalToken, &principalPayload)
	return principalPayload, err
}

func requestHandlerV2_notifications_subscribeAndWatch(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces,
	n10n in10n.IN10nBroker, appTokensFactory payloads.IAppTokensFactory) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		flusher, ok := rw.(http.Flusher)
		if !ok {
			// notest
			WriteTextResponse(rw, "streaming unsupported!", http.StatusInternalServerError)
			return
		}

		busRequest := createBusRequest(req.Method, data, req)
		principalPayload, err := authorize(appTokensFactory, busRequest)
		if err != nil {
			// [~server.n10n/err.routerCreateChannelInvalidToken~impl]
			ReplyCommonError(rw, err.Error(), http.StatusUnauthorized)
			return
		}

		subscriptions, expiresIn, err := parseN10nArgs(string(busRequest.Body))
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}

		subjectLogin := istructs.SubjectLogin(principalPayload.Login)
		channel, err := n10n.NewChannel(subjectLogin, expiresIn)
		if err != nil {
			ReplyCommonError(rw, "create new channel failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("Connection", "keep-alive")

		if _, err = fmt.Fprintf(rw, "event: channelId\ndata: %s\n\n", channel); err != nil {
			// notest
			logger.Error("failed to write created channel id to client:", err)
			return
		}
		flusher.Flush()

		subscribedProjectionKeys := []in10n.ProjectionKey{}

		for i, sub := range subscriptions {
			projectionKey := in10n.ProjectionKey{
				App:        busRequest.AppQName,
				Projection: sub.entity,
				WS:         sub.wsid,
			}
			err := n10n.Subscribe(channel, projectionKey)
			if err != nil {
				for _, subscribedKey := range subscribedProjectionKeys {
					if err = n10n.Unsubscribe(channel, subscribedKey); err != nil {
						logger.Error(fmt.Sprintf("failed to unsubscribe key %#v: %s", subscribedKey, err))
					}
				}
				ReplyCommonError(rw, fmt.Sprintf("subscriptions[%d]: subscribe failed: %s", i, err), http.StatusInternalServerError)
				return
			}
			subscribedProjectionKeys = append(subscribedProjectionKeys, projectionKey)
		}

		serveN10NChannel(req.Context(), rw, flusher, channel, n10n, subjectLogin)
	})
}

// handles both unsubscribe and subscribe to an extra view
func requestHandlerV2_notifications(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces,
	n10n in10n.IN10nBroker, appTokensFactory payloads.IAppTokensFactory) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)

		if _, err := authorize(appTokensFactory, busRequest); err != nil {
			// [~server.n10n/err.routerAddSubscriptionInvalidToken~impl]
			// [~server.n10n/err.routerUnsubscribeInvalidToken~impl]
			ReplyCommonError(rw, err.Error(), http.StatusUnauthorized)
			return
		}

		if len(busRequest.Body) > 0 {
			ReplyCommonError(rw, "unexpected body on n10n unsubscribe", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(req)
		channelID := vars[URLPlaceholder_channelID]

		entity, err := appdef.ParseQName(vars[URLPlaceholder_view])
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return
		}

		projectionKey := in10n.ProjectionKey{
			App:        busRequest.AppQName,
			Projection: entity,
			WS:         data.wsid,
		}

		code := http.StatusOK
		switch req.Method {
		case http.MethodPut:
			err = n10n.Subscribe(in10n.ChannelID(channelID), projectionKey)
		case http.MethodDelete:
			err = n10n.Unsubscribe(in10n.ChannelID(channelID), projectionKey)
			code = http.StatusNoContent
		default:
			// notest: guarded by the rule for the url path
			panic("unexpected method " + req.Method)
		}

		if err != nil {
			code = http.StatusInternalServerError
			if errors.Is(err, in10n.ErrChannelDoesNotExist) {
				code = http.StatusNotFound
			}
			ReplyCommonError(rw, "failed to unsubscribe: "+err.Error(), code)
			return
		}
		rw.WriteHeader(code)
	})
}

func parseN10nArgs(body string) (subscriptions []subscription, expiresIn time.Duration, err error) {
	n10nArgs := N10nArgs{}
	if err := coreutils.JSONUnmarshalDisallowUnknownFields([]byte(body), &n10nArgs); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal request body: %w", err)
	}
	if n10nArgs.ExpiresInSeconds == 0 {
		n10nArgs.ExpiresInSeconds = defaultN10NExpiresInSeconds
	} else if n10nArgs.ExpiresInSeconds < 0 {
		return nil, 0, fmt.Errorf("invalid expiresIn value %d", n10nArgs.ExpiresInSeconds)
	}
	expiresIn = time.Duration(n10nArgs.ExpiresInSeconds) * time.Second
	if len(n10nArgs.Subscriptions) == 0 {
		return nil, 0, errors.New("no subscriptions provided")
	}
	for i, subscr := range n10nArgs.Subscriptions {
		if len(subscr.Entity) == 0 || len(subscr.WSIDNumber.String()) == 0 {
			return nil, 0, fmt.Errorf("subscriptions[%d]: entity and\\or wsid is not provided", i)
		}
		wsid, err := coreutils.ClarifyJSONWSID(subscr.WSIDNumber)
		if err != nil {
			return nil, 0, err
		}
		entity, err := appdef.ParseQName(subscr.Entity)
		if err != nil {
			return nil, 0, fmt.Errorf("subscriptions[%d]: failed to parse entity %s as a QName: %w", i, subscr.Entity, err)
		}
		subscriptions = append(subscriptions, subscription{
			entity: entity,
			wsid:   wsid,
		})
	}
	return subscriptions, expiresIn, err
}

// [~server.devices/cmp.routerDevicesCreatePathHandler~impl]
func requestHandlerV2_create_device(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, federation federation.IFederation) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
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
		result := fmt.Sprintf(`{"%s":"%s","%s":"%s"}`, fieldLogin, login, fieldPassword, pwd)
		ReplyJSON(rw, result, http.StatusCreated)
	})
}

// [~server.authnz/cmp.routerRefreshHandler~impl]
func requestHandlerV2_auth_refresh(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Auth_Refresh)
		busRequest.Method = http.MethodGet
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

// [~server.authnz/cmp.routerLoginPathHandler~impl]
func requestHandlerV2_auth_login(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Auth_Login)
		busRequest.Method = http.MethodGet
		queryParams := map[string]string{}
		queryParams["args"] = string(busRequest.Body)
		busRequest.Query = queryParams
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
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
		if !blobRequestHandler.HandleRead_V2(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(rw, http.StatusOK), func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, ownerRecord, ownerRecordField, istructs.RecordID(ownerID), requestSender) {
			replyServiceUnavailable(rw)
		}
	}
}

func requestHandlerV2_tempblobs_read(blobRequestHandler blobprocessor.IRequestHandler,
	requestSender bus.IRequestSender) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, rw)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		suuid := iblobstorage.SUUID(vars[URLPlaceholder_blobIDOrSUUID])
		if !blobRequestHandler.HandleReadTemp_V2(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(rw, http.StatusOK), func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, requestSender, suuid) {
			replyServiceUnavailable(rw)
		}
	}
}

func requestHandlerV2_tempblobs_create(blobRequestHandler blobprocessor.IRequestHandler,
	requestSender bus.IRequestSender) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, rw)
		if !ok {
			return
		}
		if !blobRequestHandler.HandleWriteTemp_V2(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(rw, http.StatusCreated), req.Body, func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, requestSender) {
			replyServiceUnavailable(rw)
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
			newBLOBOKResponseIniter(rw, http.StatusCreated), req.Body, func(statusCode int, args ...interface{}) {
				replyErr(rw, args[0].(error))
			}, requestSender, ownerRecord, ownerRecordField) {
			replyServiceUnavailable(rw)
		}
	}
}

func requestHandlerV2_schemas_wsRole(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Schemas_WorkspaceRole)
		busRequest.WorkspaceQName = appdef.NewQName(data.vars[URLPlaceholder_pkg], data.vars[URLPlaceholder_workspaceName])
		busRequest.QName = appdef.NewQName(data.vars[URLPlaceholder_rolePkg], data.vars[URLPlaceholder_role])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

func requestHandlerV2_view(reqSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(processors.APIPath_Views)
		busRequest.QName = appdef.NewQName(data.vars[URLPlaceholder_pkg], data.vars[URLPlaceholder_view])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

func requestHandlerV2_extension(reqSender bus.IRequestSender, apiPath processors.APIPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		entity := ""
		switch apiPath {
		case processors.APIPath_Commands:
			entity = data.vars[URLPlaceholder_command]
		case processors.APIPath_Queries:
			entity = data.vars[URLPlaceholder_query]
		}
		busRequest := createBusRequest(req.Method, data, req)
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(apiPath)
		busRequest.QName = appdef.NewQName(data.vars[URLPlaceholder_pkg], entity)
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
}

func requestHandlerV2_table(reqSender bus.IRequestSender, apiPath processors.APIPath, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withRequestValidation(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		busRequest := createBusRequest(req.Method, data, req)
		if recordIDStr, hasDocID := data.vars[URLPlaceholder_id]; hasDocID {
			docID, err := strconv.ParseUint(recordIDStr, utils.DecimalBase, utils.BitSize64)
			if err != nil {
				// notest
				panic(err)
			}
			busRequest.DocID = istructs.IDType(docID)
		}
		busRequest.IsAPIV2 = true
		busRequest.APIPath = int(apiPath)
		busRequest.QName = appdef.NewQName(data.vars[URLPlaceholder_pkg], data.vars[URLPlaceholder_table])
		sendRequestAndReadResponse(req, busRequest, reqSender, rw)
	})
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

func parseChangePasswordArgs(body string) (login, oldPassword, newPassword string, err error) {
	args := coreutils.MapObject{}
	if err = json.Unmarshal([]byte(body), &args); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal body: %w", err)
	}
	ok := false
	login, ok, err = args.AsString("login")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("login field missing")
	}
	oldPassword, ok, err = args.AsString("oldPassword")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("oldPassword field missing")
	}
	newPassword, ok, err = args.AsString("newPassword")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("newPassword field missing")
	}
	return login, oldPassword, newPassword, nil
}

func parseCreateLoginArgs(body string) (verifiedEmailToken, displayName, pwd string, err error) {
	args := coreutils.MapObject{}
	if err = json.Unmarshal([]byte(body), &args); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal body: %w", err)
	}
	ok := false
	verifiedEmailToken, ok, err = args.AsString("verifiedEmailToken")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("verifiedEmailToken field missing")
	}
	displayName, ok, err = args.AsString("displayName")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("displayName field missing")
	}
	pwd, ok, err = args.AsString("password")
	if err != nil {
		return "", "", "", err
	}
	if !ok {
		return "", "", "", errors.New("password field missing")
	}
	return
}
