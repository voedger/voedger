/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
)

func reply_v1(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error,
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
	isCmd := strings.HasPrefix(busRequest.Resource, "c.")
	// if !isCmd {
	// 	if sendSuccess = writeResponse(w, "{"); !sendSuccess {
	// 		return
	// 	}
	// }
	for elem := range responseCh {
		// http client disconnected -> ErrNoConsumer on IMultiResponseSender.SendElement() -> QP will call Close()
		if requestCtx.Err() != nil {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			return
		}

		if !isCmd && elemsCount == 0 {
			if sendSuccess = writeResponse(w, `{"sections":[{"type":"","elements":[`); !sendSuccess {
				return
			}
			sectionsCloser = "]}]"
			responseCloser = "}"
		}

		if elemsCount > 0 {
			sendSuccess = writeResponse(w, ",")
		}

		elemsCount++

		if isCmd || contentType == coreutils.TextPlain {
			sendSuccess = writeResponse(w, elem.(string))
			continue
		}

		// if isCmd {
		// 	res := elem.(string)
		// 	if contentType == coreutils.ApplicationJSON {
		// 		res = strings.TrimPrefix(res, "{")
		// 		res = strings.TrimSuffix(res, "}")
		// 	}
		// 	sendSuccess = writeResponse(w, res)
		// } else if contentType == coreutils.TextPlain {
		// 	switch typed := elem.(type) {
		// 	case error:
		// 		sendSuccess = writeResponse(w, typed.Error())
		// 	default:
		// 		sendSuccess = writeResponse(w, elem.(string))
		// 	}
		// } else if elemsCount == 0 {
		// 	if !isSingle {
		// 		sendSuccess = writeResponse(w, `"sections":[{"type":"","elements":[`)
		// 		sectionsCloser = "]}]"
		// 	}
		// }

		// if sendSuccess && elemsCount > 0 {
		// 	sendSuccess = writeResponse(w, ",")
		// }

		// elemsCount++

		// if !sendSuccess {
		// 	return
		// }

		// if !busRequest.IsAPIV2 && (isCmd || contentType == coreutils.TextPlain) {
		// 	continue
		// }

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
		var sysError coreutils.SysError
		if errors.As(*responseErr, &sysError) {
			jsonErr := sysError.ToJSON_APIV1()
			if elemsCount > 0 {
				jsonErr = strings.TrimPrefix(jsonErr, "{") // need to make "sys.Error" a top-level field within {}
				jsonErr = strings.TrimSuffix(jsonErr, "}") // need to make "sys.Error" a top-level field within {}
			}
			sendSuccess = writeResponse(w, jsonErr)
		} else {
			sendSuccess = writeResponse(w, fmt.Sprintf(`"status":%d,"errorDescription":"%s"`, http.StatusInternalServerError, *responseErr))
		}
	}
	if sendSuccess && len(responseCloser) > 0 {
		sendSuccess = writeResponse(w, responseCloser)
	}
}
