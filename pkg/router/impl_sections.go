/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
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
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func createRequest(reqMethod string, req *http.Request, rw http.ResponseWriter, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (res ibus.Request, ok bool) {
	vars := mux.Vars(req)
	wsidStr := vars[WSID]
	wsidInt, err := strconv.ParseInt(wsidStr, parseInt64Base, parseInt64Bits)
	if err != nil {
		//  impossible because of regexp in a handler
		// notest
		panic(err)
	}
	appQNameStr := vars[AppOwner] + appdef.AppQNameQualifierChar + vars[AppName]
	wsid := istructs.WSID(wsidInt)
	if appQName, err := appdef.ParseAppQName(appQNameStr); err == nil {
		if numAppWorkspaces, ok := numsAppsWorkspaces[appQName]; ok {
			baseWSID := wsid.BaseWSID()
			if baseWSID < istructs.MaxPseudoBaseWSID {
				wsid = coreutils.GetAppWSID(wsid, numAppWorkspaces)
			}
		}
	}
	res = ibus.Request{
		Method:   ibus.NameToHTTPMethod[reqMethod],
		WSID:     int64(wsid),
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

func writeSectionedResponse(requestCtx context.Context, w http.ResponseWriter, sections <-chan ibus.ISection, secErr *error, onSendFailed func()) {
	ok := true
	var iSection ibus.ISection
	defer func() {
		if !ok {
			onSendFailed()
			// consume all pending sections or elems to avoid hanging on ibusnats side
			// normally should one pending elem or section because ibusnats implementation
			// will terminate on next elem or section because `onSendFailed()` actually closes the context
			discardSection(iSection, requestCtx)
			for iSection := range sections {
				discardSection(iSection, requestCtx)
			}
		}
	}()

	sectionsOpened := false
	sectionedResponseStarted := false

	closer := ""
	readSections := func() bool {
		for iSection = range sections {
			// possible: ctx is done but on select {sections<-section, <-ctx.Done()} write to sections channel is triggered.
			// ctx.Done() must have the priority
			if requestCtx.Err() != nil {
				ok = false
				break
			}
			if !sectionedResponseStarted {
				if ok = startSectionedResponse(w); !ok {
					return false
				}
				sectionedResponseStarted = true
			}

			if !sectionsOpened {
				if ok = writeResponse(w, `"sections":[`); !ok {
					return false
				}
				closer = "]"
				sectionsOpened = true
			} else {
				if ok = writeResponse(w, ","); !ok {
					return false
				}
			}
			if ok = writeSection(w, iSection, requestCtx); !ok {
				return false
			}
		}
		return true
	}
	mustReturn := readSections()
	if requestCtx.Err() != nil {
		if onRequestCtxClosed != nil {
			onRequestCtxClosed()
		}
		log.Println("client disconnected during sections sending")
		return
	}
	if !mustReturn {
		return
	}

	if *secErr != nil {
		if !sectionedResponseStarted {
			if !startSectionedResponse(w) {
				return
			}
		}
		if sectionsOpened {
			closer = "],"
		}
		var jsonableErr interface{ ToJSON() string }
		if errors.As(*secErr, &jsonableErr) {
			jsonErr := jsonableErr.ToJSON()
			jsonErr = strings.TrimPrefix(jsonErr, "{")
			jsonErr = strings.TrimSuffix(jsonErr, "}")
			writeResponse(w, fmt.Sprintf(`%s%s}`, closer, jsonErr))
		} else {
			writeResponse(w, fmt.Sprintf(`%s"status":%d,"errorDescription":"%s"}`, closer, http.StatusInternalServerError, *secErr))
		}
	} else if sectionedResponseStarted {
		writeResponse(w, fmt.Sprintf(`%s}`, closer))
	}
}

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

func startSectionedResponse(w http.ResponseWriter) bool {
	w.Header().Set(coreutils.ContentType, coreutils.ApplicationJSON)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	return writeResponse(w, "{")
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
				if !writeResponse(w, fmt.Sprintf(`,"elements":[%s`, string(val))) {
					return false
				}
				isFirst = false
				closer = "]}"
			} else if !writeResponse(w, fmt.Sprintf(`,%s`, string(val))) {
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
