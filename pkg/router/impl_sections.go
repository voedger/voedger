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
	"github.com/valyala/bytebufferpool"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func createRequest(reqMethod string, req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (res ibus.Request, ok bool) {
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
	res = ibus.Request{
		Method:   ibus.NameToHTTPMethod[reqMethod],
		WSID:     wsid,
		Query:    req.URL.Query(),
		Header:   req.Header,
		AppQName: appQNameStr,
		Host:     req.Host,
	}
	if req.Body != nil && req.Body != http.NoBody {
		if res.Body, err = io.ReadAll(req.Body); err != nil {
			http.Error(rw, "failed to read body", http.StatusInternalServerError)
		}
	}
	return res, err == nil
}

func reply(requestCtx context.Context, w http.ResponseWriter, responseCh <-chan any, responseErr *error, contentType string, onSendFailed func()) {
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
			if contentType == coreutils.TextPlain {
				sendSuccess = writeResponse(w, elem.(string))
			} else {
				sendSuccess = writeResponse(w, `{"sections":[{"type":"","elements":[`)
				closer = "]}]"
			}
		} else {
			sendSuccess = writeResponse(w, ",")
		}

		if !sendSuccess {
			return
		}

		if contentType == coreutils.TextPlain {
			continue
		}

		elemBytes, err := json.Marshal(&elem)
		if err != nil {
			panic(err)
		}

		if sendSuccess = writeResponse(w, string(elemBytes)); !sendSuccess {
			return
		}
		elemsCount++
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
	} else if sendSuccess && contentType == coreutils.ApplicationJSON {
		if elemsCount == 0 {
			sendSuccess = writeResponse(w, "{}")
		} else {
			sendSuccess = writeResponse(w, "}")
		}
	}
}

// func writeSectionedResponse_(requestCtx context.Context, w http.ResponseWriter, marshaledElems <-chan string, secErr *error, onSendFailed func()) {
// 	sendSuccess := false
// 	defer func() {
// 		if requestCtx.Err() != nil {
// 			if onRequestCtxClosed != nil {
// 				onRequestCtxClosed()
// 			}
// 			log.Println("client disconnected during sections sending")
// 			return
// 		}
// 		if !sendSuccess {
// 			onSendFailed()
// 			for range marshaledElems {
// 			}
// 		}
// 	}()
// 	elemsCount := 0
// 	for marshaledElem := range marshaledElems {
// 		// http client disconnected -> ErrNoConsumer on IMultiResponseSender.SendElement() -> QP will call Close()
// 		if requestCtx.Err() != nil {
// 			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
// 			// ctx.Done() must have the priority
// 			return
// 		}
// 		if elemsCount == 0 {
// 			sendSuccess = initResponse(w, coreutils.ApplicationJSON, http.StatusOK) && writeResponse(w, `"sections":[{"type":"","elements":[`)
// 		} else {
// 			sendSuccess = writeResponse(w, ",")
// 		}

// 		if !sendSuccess {
// 			return
// 		}

// 		if sendSuccess = writeResponse(w, marshaledElem); !sendSuccess {
// 			return
// 		}
// 		elemsCount++
// 	}
// 	if elemsCount > 0 {
// 		if sendSuccess = writeResponse(w, "]}]"); !sendSuccess {
// 			return
// 		}
// 	}
// 	if *secErr != nil {
// 		if elemsCount == 0 {
// 			// no elements -> let's see which status code to send
// 			headerStatusCode := http.StatusInternalServerError
// 			var sysErr coreutils.SysError
// 			if errors.As(*secErr, &sysErr) {
// 				headerStatusCode = sysErr.HTTPStatus
// 			}
// 			sendSuccess = initResponse(w, coreutils.ApplicationJSON, headerStatusCode)
// 		} else {
// 			sendSuccess = writeResponse(w, ",")
// 		}
// 		if !sendSuccess {
// 			return
// 		}
// 		var jsonableErr interface{ ToJSON() string }
// 		if errors.As(*secErr, &jsonableErr) {
// 			jsonErr := jsonableErr.ToJSON()
// 			// will not eliminate { and } because currently router will send not initial "{" and final "}" if the first element is string
// 			// that is done for backward compatibility for handling text/plain responses
// 			sendSuccess = writeResponse(w, jsonErr)
// 		} else {
// 			sendSuccess = writeResponse(w, fmt.Sprintf(`"status":%d,"errorDescription":"%s"}`, http.StatusInternalServerError, *secErr))
// 		}
// 	}
// 	if sendSuccess {
// 		sendSuccess = writeResponse(w, "}")
// 	}
// }

// func writeSectionedResponse(requestCtx context.Context, w http.ResponseWriter, marshaledElems <-chan string, secErr *error, onSendFailed func()) {
// 	ok := true
// 	var iSection ibus.ISection
// 	defer func() {
// 		if !ok {
// 			onSendFailed()
// 			// consume all pending sections or elems to avoid hanging on ibusnats side
// 			// normally should one pending elem or section because ibusnats implementation
// 			// will terminate on next elem or section because `onSendFailed()` actually closes the context
// 			discardSection(iSection, requestCtx)
// 			for iSection := range sections {
// 				discardSection(iSection, requestCtx)
// 			}
// 		}
// 	}()

// 	sectionsOpened := false
// 	sectionedResponseStarted := false

// 	closer := ""
// 	readSections := func() bool {
// 		for iSection = range sections {
// 			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
// 			// ctx.Done() must have the priority
// 			if requestCtx.Err() != nil {
// 				ok = false
// 				break
// 			}
// 			if !sectionedResponseStarted {
// 				if ok = startSectionedResponse(w); !ok {
// 					return false
// 				}
// 				sectionedResponseStarted = true
// 			}

// 			if !sectionsOpened {
// 				if ok = writeResponse(w, `"sections":[`); !ok {
// 					return false
// 				}
// 				closer = "]"
// 				sectionsOpened = true
// 			} else {
// 				if ok = writeResponse(w, ","); !ok {
// 					return false
// 				}
// 			}
// 			if ok = writeSection(w, iSection, requestCtx); !ok {
// 				return false
// 			}
// 		}
// 		return true
// 	}
// 	mustReturn := readSections()
// 	if requestCtx.Err() != nil {
// 		if onRequestCtxClosed != nil {
// 			onRequestCtxClosed()
// 		}
// 		log.Println("client disconnected during sections sending")
// 		return
// 	}
// 	if !mustReturn {
// 		return
// 	}

// 	if *secErr != nil {
// 		if !sectionedResponseStarted {
// 			if !startSectionedResponse(w) {
// 				return
// 			}
// 		}
// 		if sectionsOpened {
// 			closer = "],"
// 		}
// 		var jsonableErr interface{ ToJSON() string }
// 		if errors.As(*secErr, &jsonableErr) {
// 			jsonErr := jsonableErr.ToJSON()
// 			jsonErr = strings.TrimPrefix(jsonErr, "{")
// 			jsonErr = strings.TrimSuffix(jsonErr, "}")
// 			writeResponse(w, fmt.Sprintf(`%s%s}`, closer, jsonErr))
// 		} else {
// 			writeResponse(w, fmt.Sprintf(`%s"status":%d,"errorDescription":"%s"}`, closer, http.StatusInternalServerError, *secErr))
// 		}
// 	} else if sectionedResponseStarted {
// 		writeResponse(w, closer+"}")
// 	}
// }

func discardSection(iSection ibus.ISection, requestCtx context.Context) {
	switch t := iSection.(type) {
	case nil:
	case ibus.IObjectSection:
		t.Value(requestCtx)
	case ibus.IMapSection:
		for _, _, ok := t.Next(requestCtx); ok; _, _, ok = t.Next(requestCtx) {
		}
	case ibus.IArraySection:
		for _, ok := t.Next(requestCtx); ok; _, ok = t.Next(requestCtx) {
		}
	}
}

func initResponse(w http.ResponseWriter, contentType string, statusCode int) {
	w.Header().Set(coreutils.ContentType, contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
}

func writeSection(w http.ResponseWriter, isec ibus.ISection, requestCtx context.Context) bool {
	switch sec := isec.(type) {
	case ibus.IArraySection:
		if !writeSectionHeader(w, sec) {
			return false
		}
		isFirst := true
		closer := "}"
		// ctx.Done() is tracked by ibusnats implementation: writing to section elem channel -> read here, ctxdone -> close elem channel
		for val, ok := sec.Next(requestCtx); ok; val, ok = sec.Next(requestCtx) {
			if isFirst {
				if !writeResponse(w, `,"elements":[`+string(val)) {
					return false
				}
				isFirst = false
				closer = "]}"
			} else if !writeResponse(w, ","+string(val)) {
				return false
			}
		}
		if !writeResponse(w, closer) {
			return false
		}
	case ibus.IObjectSection:
		if !writeSectionHeader(w, sec) {
			return false
		}
		val := sec.Value(requestCtx)
		if !writeResponse(w, fmt.Sprintf(`,"elements":%s}`, string(val))) {
			return false
		}
	case ibus.IMapSection:
		if !writeSectionHeader(w, sec) {
			return false
		}
		isFirst := true
		closer := "}"
		// ctx.Done() is tracked by ibusnats implementation: writing to section elem channel -> read here, ctxdone -> close elem channel
		for name, val, ok := sec.Next(requestCtx); ok; name, val, ok = sec.Next(requestCtx) {
			if isFirst {
				if !writeResponse(w, fmt.Sprintf(`,"elements":{%q:%s`, name, string(val))) {
					return false
				}
				isFirst = false
				closer = "}}"
			} else if !writeResponse(w, fmt.Sprintf(`,%q:%s`, name, string(val))) {
				return false
			}
		}
		if !writeResponse(w, closer) {
			return false
		}
	}
	return true
}

func writeSectionHeader(w http.ResponseWriter, sec ibus.IDataSection) bool {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, _ = buf.WriteString(fmt.Sprintf(`{"type":%q`, sec.Type())) // error impossible
	if len(sec.Path()) > 0 {
		_, _ = buf.WriteString(`,"path":[`) // error impossible
		for i, p := range sec.Path() {
			if i > 0 {
				_, _ = buf.WriteString(",") // error impossible
			}
			_, _ = buf.WriteString(fmt.Sprintf(`%q`, p)) // error impossible
		}
		_, _ = buf.WriteString("]") // error impossible
	}
	if !writeResponse(w, string(buf.Bytes())) {
		return false
	}
	return true
}
