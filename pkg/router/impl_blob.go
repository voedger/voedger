/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
)

func newErrorResponder(w http.ResponseWriter) blobprocessor.ErrorResponder {
	return func(statusCode int, args ...interface{}) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(fmt.Sprint(args...)))
	}
}

func (s *httpService) blobHTTPRequestHandler_Write() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if logger.IsVerbose() {
			logger.Verbose("blob write request:", req.URL.String())
		}
		appQName, wsid, headers, ok := parseURLParams(req, resp)
		if !ok {
			return
		}
		if !s.blobRequestHandler.HandleWrite(appQName, wsid, headers, req.Context(), req.URL.Query(),
			newBLOBOKResponseIniter(resp), req.Body, func(statusCode int, args ...interface{}) {
				WriteTextResponse(resp, fmt.Sprint(args...), statusCode)
			}, s.requestSender) {
			resp.WriteHeader(http.StatusServiceUnavailable)
			resp.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
}

func (s *httpService) blobHTTPRequestHandler_Read() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if logger.IsVerbose() {
			logger.Verbose("blob read request:", req.URL.String())
		}
		appQName, wsid, headers, ok := parseURLParams(req, resp)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		existingBLOBIDOrSUID := vars[URLPlaceholder_blobIDOrSUUID]
		if !s.blobRequestHandler.HandleRead(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(resp), func(statusCode int, args ...interface{}) {
				WriteTextResponse(resp, fmt.Sprint(args...), statusCode)
			}, existingBLOBIDOrSUID, s.requestSender) {
			resp.WriteHeader(http.StatusServiceUnavailable)
			resp.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
}

func parseURLParams(req *http.Request, resp http.ResponseWriter) (appQName appdef.AppQName, wsid istructs.WSID, headers http.Header, ok bool) {
	vars := mux.Vars(req)
	wsidUint, err := strconv.ParseUint(vars[URLPlaceholder_wsid], utils.DecimalBase, utils.BitSize64)
	if err != nil {
		// notest: checked by router url rule
		panic(err)
	}
	headers = maps.Clone(req.Header)
	if _, ok := headers[coreutils.Authorization]; !ok {
		// no token among headers -> look among cookies
		// no token among cookies as well -> just do nothing, 403 will happen on call helper commands further in BLOBs processor
		cookie, err := req.Cookie(coreutils.Authorization)
		if !errors.Is(err, http.ErrNoCookie) {
			if err != nil {
				// notest
				panic(err)
			}
			val, err := url.QueryUnescape(cookie.Value)
			if err != nil {
				WriteTextResponse(resp, "failed to unescape cookie '"+cookie.Value+"'", http.StatusBadRequest)
				return appQName, wsid, headers, false
			}
			// authorization token in cookies -> q.sys.DownloadBLOBAuthnz requires it in headers
			headers[coreutils.Authorization] = []string{val}
		}
	}
	appQName = appdef.NewAppQName(vars[URLPlaceholder_appOwner], vars[URLPlaceholder_appName])
	return appQName, istructs.WSID(wsidUint), headers, true
}

func newBLOBOKResponseIniter(r http.ResponseWriter) func(headersKV ...string) io.Writer {
	return func(headersKV ...string) io.Writer {
		for i := 0; i < len(headersKV); i += 2 {
			r.Header().Set(headersKV[i], headersKV[i+1])
		}
		r.WriteHeader(http.StatusOK)
		return r
	}
}
