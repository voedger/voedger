/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
)

func createRequest(reqMethod string, req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (res bus.Request, ok bool) {
	vars := mux.Vars(req)
	wsidStr := vars[URLPlaceholder_wsid]
	wsidUint, err := strconv.ParseUint(wsidStr, utils.DecimalBase, utils.BitSize64)
	if err != nil {
		// impossible because of regexp in a handler
		// notest
		panic(err)
	}
	appQNameStr := vars[URLPlaceholder_appOwner] + appdef.AppQNameQualifierChar + vars[URLPlaceholder_appName]
	wsid := istructs.WSID(wsidUint)
	if appQName, err := appdef.ParseAppQName(appQNameStr); err == nil {
		if numAppWorkspaces, ok := numsAppsWorkspaces[appQName]; ok {
			baseWSID := wsid.BaseWSID()
			if baseWSID <= istructs.MaxPseudoBaseWSID {
				wsid = coreutils.GetAppWSID(wsid, numAppWorkspaces)
			}
		}
	}
	res = bus.Request{
		Method:   reqMethod,
		WSID:     wsid,
		Query:    req.URL.Query(),
		Header:   req.Header,
		AppQName: appQNameStr,
	}
	if req.Body != nil && req.Body != http.NoBody {
		if res.Body, err = io.ReadAll(req.Body); err != nil {
			http.Error(rw, "failed to read body", http.StatusInternalServerError)
		}
	}
	return res, err == nil
}

func reply(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error, contentType string, onSendFailed func(), isCmd bool) {
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
	elemsCount := 0
	closer := ""
	for elem := range responseCh {
		// http client disconnected -> ErrNoConsumer on IMultiResponseSender.SendElement() -> QP will call Close()
		if requestCtx.Err() != nil {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			return
		}
		if elemsCount == 0 {
			if isCmd || contentType == coreutils.TextPlain {
				sendSuccess = writeResponse(w, elem.(string))
			} else {
				sendSuccess = writeResponse(w, `{"sections":[{"type":"","elements":[`)
				closer = "]}]"
			}
		} else {
			sendSuccess = writeResponse(w, ",")
		}

		elemsCount++

		if !sendSuccess {
			return
		}

		if isCmd || contentType == coreutils.TextPlain {
			continue
		}

		elemBytes, err := json.Marshal(&elem)
		if err != nil {
			panic(err)
		}

		if sendSuccess = writeResponse(w, string(elemBytes)); !sendSuccess {
			return
		}
	}
	if len(closer) > 0 {
		if sendSuccess = writeResponse(w, closer); !sendSuccess {
			return
		}
	}
	if *responseErr != nil {
		if elemsCount > 0 {
			sendSuccess = writeResponse(w, ",")
		} else {
			sendSuccess = writeResponse(w, "{")
		}
		if !sendSuccess {
			return
		}
		var jsonableErr interface{ ToJSON() string }
		if errors.As(*responseErr, &jsonableErr) {
			jsonErr := jsonableErr.ToJSON()
			jsonErr = strings.TrimPrefix(jsonErr, "{")
			sendSuccess = writeResponse(w, jsonErr)
		} else {
			sendSuccess = writeResponse(w, fmt.Sprintf(`"status":%d,"errorDescription":"%s"}`, http.StatusInternalServerError, *responseErr))
		}
	} else if sendSuccess && contentType == coreutils.ApplicationJSON && !isCmd {
		if elemsCount == 0 {
			sendSuccess = writeResponse(w, "{}")
		} else {
			sendSuccess = writeResponse(w, "}")
		}
	}
}

func initResponse(w http.ResponseWriter, contentType string, statusCode int) {
	w.Header().Set(coreutils.ContentType, contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
}
