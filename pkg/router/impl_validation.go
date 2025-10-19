/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

func withValidateForFuncs(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(req *http.Request, rw http.ResponseWriter, data validatedData)) http.HandlerFunc {
	return withValidate(numsAppsWorkspaces, handler, readBody, cookiesTokenToHeaders)
}

func withValidateForN10N(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(req *http.Request, rw http.ResponseWriter, data validatedData)) http.HandlerFunc {
	return withValidate(numsAppsWorkspaces, handler, readBody, cookiesTokenToHeaders)
}

func withValidateForBLOBs(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(req *http.Request, rw http.ResponseWriter, data validatedData)) http.HandlerFunc {
	return withValidate(numsAppsWorkspaces, handler, cookiesTokenToHeaders)
}

func withValidate(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(req *http.Request, rw http.ResponseWriter, data validatedData), validators ...validatorFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		data, ok := validate(req, rw, numsAppsWorkspaces, validators...)
		if !ok {
			return
		}
		handler(req, rw, data)
	}
}

func validateRequest(req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (validatedData, bool) {
	vars := mux.Vars(req)
	wsidStr := vars[URLPlaceholder_wsid]
	var wsid istructs.WSID
	var err error
	if len(wsidStr) > 0 {
		wsid, err = coreutils.ClarifyJSONWSID(json.Number(wsidStr))
		if err != nil {
			ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
			return validatedData{}, false
		}
	}

	appQNameStr := vars[URLPlaceholder_appOwner] + appdef.AppQNameQualifierChar + vars[URLPlaceholder_appName]
	appQName, err := appdef.ParseAppQName(appQNameStr)
	if err != nil {
		ReplyCommonError(rw, err.Error(), http.StatusBadRequest)
		return validatedData{}, false
	}

	if numAppWorkspaces, ok := numsAppsWorkspaces[appQName]; ok {
		baseWSID := wsid.BaseWSID()
		if baseWSID <= istructs.MaxPseudoBaseWSID {
			wsid = coreutils.PseudoWSIDToAppWSID(wsid, numAppWorkspaces)
		}
	}

	res := validatedData{
		vars:     vars,
		wsid:     wsid,
		appQName: appQName,
		header:   map[string]string{},
	}

	for k, v := range req.Header {
		res.header[k] = v[0]
	}
	return res, true
}

func validate(req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, validators ...validatorFunc) (validatedData, bool) {
	validatedData, ok := validateRequest(req, rw, numsAppsWorkspaces)
	if !ok {
		return validatedData, false
	}
	for _, validator := range validators {
		validatedData, ok = validator(validatedData, req, rw)
		if !ok {
			return validatedData, false
		}
	}
	return validatedData, true
}

func readBody(validatedData validatedData, req *http.Request, rw http.ResponseWriter) (validatedData, bool) {
	if req.Body == nil || req.Body == http.NoBody {
		return validatedData, true
	}
	var err error
	validatedData.body, err = io.ReadAll(req.Body)
	if err != nil {
		// notest
		logger.Error("failed to read body", err.Error())
		return validatedData, false
	}
	return validatedData, true
}

// does not read body
func cookiesTokenToHeaders(validatedData validatedData, req *http.Request, rw http.ResponseWriter) (validatedData, bool) {
	if _, ok := validatedData.header[httpu.Authorization]; !ok {
		// no token among headers -> look among cookies
		// no token among cookies as well -> just do nothing, 403 will happen on call helper commands further in BLOBs processor
		cookieBearerToken, ok, err := GetCookieBearerAuth(req)
		if err != nil {
			WriteTextResponse(rw, err.Error(), http.StatusBadRequest)
			return validatedData, false
		}
		if ok {
			// authorization token in cookies -> q.sys.DownloadBLOBAuthnz requires it in headers
			validatedData.header[httpu.Authorization] = cookieBearerToken
		}
	}
	return validatedData, true
}
