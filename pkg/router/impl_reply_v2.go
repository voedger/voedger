/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/strconvu"
	"github.com/voedger/voedger/pkg/istructs"
)

// validatedData contains validated data from HTTP request
type validatedData struct {
	vars     map[string]string
	wsid     istructs.WSID
	appQName appdef.AppQName
	header   map[string]string
	body     []byte
}

// does not read body
func validateRequestForBLOBs(req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (validatedData, bool) {
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

	if _, ok := res.header[httpu.Authorization]; !ok {
		// no token among headers -> look among cookies
		// no token among cookies as well -> just do nothing, 403 will happen on call helper commands further in BLOBs processor
		cookieBearerToken, ok, err := GetCookieBearerAuth(req)
		if err != nil {
			WriteTextResponse(rw, err.Error(), http.StatusBadRequest)
			return validatedData{}, false
		}
		if ok {
			// authorization token in cookies -> q.sys.DownloadBLOBAuthnz requires it in headers
			res.header[httpu.Authorization] = cookieBearerToken
		}
	}

	return res, true
}

// validateRequest validates the HTTP request and returns validated data or error
func validateRequest(req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (validatedData, bool) {
	res, ok := validateRequestForBLOBs(req, rw, numsAppsWorkspaces)
	if !ok {
		return validatedData{}, false
	}
	if req.Body == nil || req.Body == http.NoBody {
		return res, true
	}
	var err error
	res.body, err = io.ReadAll(req.Body)
	if err != nil {
		// notest
		logger.Error("failed to read body", err.Error())
		return validatedData{}, false
	}
	return res, true
}

// createBusRequest creates a bus.Request from validated data
func createBusRequest(reqMethod string, data validatedData, req *http.Request) bus.Request {
	res := bus.Request{
		Method:   reqMethod,
		WSID:     data.wsid,
		Query:    map[string]string{},
		Header:   data.header,
		AppQName: data.appQName,
		Resource: data.vars[URLPlaceholder_resourceName],
		Body:     data.body,
	}

	if docIDStr, hasDocID := data.vars[URLPlaceholder_id]; hasDocID {
		docIDUint64, err := strconvu.ParseUint64(docIDStr)
		if err != nil {
			// notest: parsed already by route regexp
			panic(err)
		}
		res.DocID = istructs.IDType(docIDUint64)
	}

	for k, v := range req.URL.Query() {
		res.Query[k] = v[0]
	}
	return res
}

// withRequestValidation is a middleware that validates the request before passing it to the handler
func withRequestValidation(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(*http.Request, http.ResponseWriter, validatedData)) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		data, ok := validateRequest(req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		handler(req, rw, data)
	}
}

func withBLOBsRequestValidation(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces, handler func(*http.Request, http.ResponseWriter, validatedData)) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		data, ok := validateRequestForBLOBs(req, rw, numsAppsWorkspaces)
		if !ok {
			return
		}
		handler(req, rw, data)
	}
}

func reply_v2(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error, onSendFailed func(), respMode bus.RespondMode) {
	sendSuccess := true
	defer func() {
		if requestCtx.Err() != nil {
			if onRequestCtxClosed != nil {
				onRequestCtxClosed()
			}
			log.Println("client disconnected during sections sending")
			return
		}
		if !sendSuccess {
			onSendFailed()
			for range responseCh {
			}
		}
	}()

	// ApiArray and no elems -> {"results":[]}

	if respMode == bus.RespondMode_ApiArray {
		if sendSuccess = writeResponse(w, `{"results":[`); !sendSuccess {
			return
		}
	}
	elemsCount := 0
	for elem := range responseCh {
		if requestCtx.Err() != nil {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			return
		}

		toSend := ""

		if respMode == bus.RespondMode_ApiArray {
			if elemsCount > 0 {
				if sendSuccess = writeResponse(w, ","); !sendSuccess {
					return
				}
			}
			toSendBytes, err := json.Marshal(&elem)
			if err != nil {
				panic(err)
			}

			toSend = string(toSendBytes)
		} else {
			switch typed := elem.(type) {
			case nil:
				toSend = "{}"
			case string:
				toSend = typed
			case []byte:
				toSend = string(typed)
			case coreutils.SysError:
				toSend = typed.ToJSON_APIV2()
			default:
				elemBytes, err := json.Marshal(elem)
				if err != nil {
					// notest
					panic(err)
				}
				toSend = string(elemBytes)
			}
		}

		if sendSuccess = writeResponse(w, toSend); !sendSuccess {
			return
		}

		elemsCount++
	}

	if respMode == bus.RespondMode_ApiArray {
		if sendSuccess = writeResponse(w, "]"); !sendSuccess {
			return
		}
	}

	if *responseErr != nil {
		// actual for ApiArray mode only
		if sendSuccess = writeResponse(w, ","); !sendSuccess {
			return
		}
		var sysError coreutils.SysError
		if errors.As(*responseErr, &sysError) {
			jsonErr := sysError.ToJSON_APIV2()
			sendSuccess = writeResponse(w, `"error":`+jsonErr)
		} else {
			if sendSuccess = writeResponse(w, `"error":`); sendSuccess {
				sendSuccess = writeCommonError(w, (*responseErr).Error(), http.StatusInternalServerError)
			}
		}
	}

	if sendSuccess && respMode == bus.RespondMode_ApiArray {
		sendSuccess = writeResponse(w, "}")
	}
}
