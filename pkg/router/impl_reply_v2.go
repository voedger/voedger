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
			jsonErr = strings.TrimPrefix(jsonErr, "{")
			jsonErr = strings.TrimSuffix(jsonErr, "}")
			sendSuccess = writeResponse(w, jsonErr)
		} else {
			sendSuccess = writeResponse(w, fmt.Sprintf(`"error":{"status":%d,"message":"%s"}`, http.StatusInternalServerError, *responseErr))
		}
	}

	if sendSuccess && respMode == bus.RespondMode_ApiArray {
		sendSuccess = writeResponse(w, "}")
	}
}
