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
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func reply_v1(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error,
	onSendFailed func(), busRequest bus.Request, responseMeta bus.ResponseMeta) {
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
			logger.ErrorCtx(requestCtx, "routing.response.error", "failed to write response")
			onSendFailed()
			for range responseCh {
			}
		}
	}()

	if responseMeta.Mode() == bus.RespondMode_Single {
		select {
		case data := <-responseCh:
			if requestCtx.Err() != nil {
				// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
				// ctx.Done() must have the priority
				return
			}
			switch typed := data.(type) {
			case string:
				w.WriteHeader(responseMeta.StatusCode)
				sendSuccess = writeResponse(w, typed)
			case []byte:
				w.WriteHeader(responseMeta.StatusCode)
				sendSuccess = writeResponse(w, string(typed))
			case coreutils.SysError:
				applySysErrorHeaders(w, typed)
				w.WriteHeader(responseMeta.StatusCode)
				if busRequest.IsAPIV2 {
					sendSuccess = writeResponse(w, typed.ToJSON_APIV2())
				} else {
					sendSuccess = writeResponse(w, typed.ToJSON_APIV1())
				}
			default:
				elemBytes, err := json.Marshal(data)
				if err != nil {
					// notest
					panic(err)
				}
				w.WriteHeader(responseMeta.StatusCode)
				sendSuccess = writeResponse(w, string(elemBytes))
			}
		case <-requestCtx.Done():
		}
		return
	}
	elemsCount := 0
	sectionsCloser := ""
	responseCloser := "{}"
	isCmd := strings.HasPrefix(busRequest.Resource, "c.")
	headerWritten := false
	writeHeaderOnce := func() {
		if !headerWritten {
			w.WriteHeader(responseMeta.StatusCode)
			headerWritten = true
		}
	}
	for elem := range responseCh {
		// http client disconnected -> ErrNoConsumer on IMultiResponseSender.SendElement() -> QP will call Close()
		if requestCtx.Err() != nil {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			return
		}
		if !isCmd && elemsCount == 0 {
			writeHeaderOnce()
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

		if isCmd || responseMeta.ContentType == httpu.ContentType_TextPlain {
			writeHeaderOnce()
			sendSuccess = writeResponse(w, elem.(string))
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
		} else {
			responseCloser = ""
		}
		if !sendSuccess {
			return
		}
		var sysError coreutils.SysError
		if errors.As(*responseErr, &sysError) {
			if !headerWritten {
				applySysErrorHeaders(w, sysError)
			}
			writeHeaderOnce()
			jsonErr := sysError.ToJSON_APIV1()
			if elemsCount > 0 {
				jsonErr = strings.TrimPrefix(jsonErr, "{") // need to make "sys.Error" a top-level field within {}
				jsonErr = strings.TrimSuffix(jsonErr, "}") // need to make "sys.Error" a top-level field within {}
			}
			sendSuccess = writeResponse(w, jsonErr)
		} else {
			writeHeaderOnce()
			sendSuccess = writeResponse(w, fmt.Sprintf(`"status":%d,"errorDescription":"%s"`, http.StatusInternalServerError, *responseErr))
		}
	}
	writeHeaderOnce()
	if sendSuccess && len(responseCloser) > 0 {
		sendSuccess = writeResponse(w, responseCloser)
	}
}
