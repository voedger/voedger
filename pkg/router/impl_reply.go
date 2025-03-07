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
	"github.com/voedger/voedger/pkg/processors/query2"
)

func createBusRequest(reqMethod string, req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (res bus.Request, ok bool) {
	vars := mux.Vars(req)
	wsidStr := vars[URLPlaceholder_wsid]
	wsidUint, err := strconv.ParseUint(wsidStr, utils.DecimalBase, utils.BitSize64)
	if err != nil {
		// notest: impossible because of regexp in a handler
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
		Query:    map[string]string{},
		Header:   map[string]string{},
		AppQName: appdef.NewAppQName(vars[URLPlaceholder_appOwner], vars[URLPlaceholder_appName]),
		Resource: vars[URLPlaceholder_resourceName],
	}

	if docIDStr, hasDocID := vars[URLPlaceholder_id]; hasDocID {
		docIDUint64, err := strconv.ParseUint(docIDStr, utils.DecimalBase, utils.BitSize64)
		if err != nil {
			// notest: prased already by route regexp
			panic(err)
		}
		res.DocID = istructs.IDType(docIDUint64)
	}
	for k, v := range req.URL.Query() {
		res.Query[k] = v[0]
	}
	for k, v := range req.Header {
		res.Header[k] = v[0]
	}
	if req.Body != nil && req.Body != http.NoBody {
		if res.Body, err = io.ReadAll(req.Body); err != nil {
			http.Error(rw, "failed to read body", http.StatusInternalServerError)
		}
	}
	return res, err == nil
}

func reply(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error,
	contentType string, onSendFailed func(), busRequest bus.Request) {
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
	sectionsCloser := ""
	responseCloser := ""
	isCmd := false
	if busRequest.IsAPIV2 {
		isCmd = busRequest.ApiPath == int(query2.ApiPath_Commands)
	} else {
		isCmd = strings.HasPrefix(busRequest.Resource, "c.")
	}
	if contentType == coreutils.ApplicationJSON {
		if sendSuccess = writeResponse(w, "{"); !sendSuccess {
			return
		}
	}
	for elem := range responseCh {
		// http client disconnected -> ErrNoConsumer on IMultiResponseSender.SendElement() -> QP will call Close()
		if requestCtx.Err() != nil {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			return
		}
		if busRequest.IsAPIV2 {
			if elemsCount == 0 {
				sendSuccess = writeResponse(w, `"results":[`)
				responseCloser = "]"
			}
		} else {
			if isCmd {
				res := elem.(string)
				if contentType == coreutils.ApplicationJSON {
					res = strings.TrimPrefix(res, "{")
					res = strings.TrimSuffix(res, "}")
				}
				sendSuccess = writeResponse(w, res)
			} else if contentType == coreutils.TextPlain {
				sendSuccess = writeResponse(w, elem.(string))
			} else if elemsCount == 0 {
				sendSuccess = writeResponse(w, `"sections":[{"type":"","elements":[`)
				sectionsCloser = "]}]"
			}
		}

		if sendSuccess && elemsCount > 0 {
			sendSuccess = writeResponse(w, ",")
		}

		elemsCount++

		if !sendSuccess {
			return
		}

		if !busRequest.IsAPIV2 && (isCmd || contentType == coreutils.TextPlain) {
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
	if len(sectionsCloser) > 0 {
		if sendSuccess = writeResponse(w, sectionsCloser); !sendSuccess {
			return
		}
	}
	if *responseErr != nil {
		if elemsCount > 0 {
			sendSuccess = writeResponse(w, ",")
		}
		if !sendSuccess {
			return
		}
		var jsonableErr interface{ ToJSON() string }
		if errors.As(*responseErr, &jsonableErr) {
			jsonErr := jsonableErr.ToJSON()
			jsonErr = strings.TrimPrefix(jsonErr, "{") // need to make "sys.Error" a top-level field within {}
			jsonErr = strings.TrimSuffix(jsonErr, "}") // need to make "sys.Error" a top-level field within {}
			sendSuccess = writeResponse(w, jsonErr)
		} else {
			sendSuccess = writeResponse(w, fmt.Sprintf(`"status":%d,"errorDescription":"%s"`, http.StatusInternalServerError, *responseErr))
		}
	}

	if len(responseCloser) > 0 {
		sendSuccess = writeResponse(w, responseCloser)
	}
	if sendSuccess && contentType == coreutils.ApplicationJSON {
		sendSuccess = writeResponse(w, "}")
	}
}

func initResponse(w http.ResponseWriter, contentType string, statusCode int) {
	w.Header().Set(coreutils.ContentType, contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
}
