/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

func (s *httpService) blobHTTPRequestHandler_Write(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withValidateForBLOBs(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		if logger.IsVerbose() {
			logger.Verbose("blob write request:", req.URL.String())
		}
		if !s.blobRequestHandler.HandleWrite(data.appQName, data.wsid, data.header, req.Context(), req.URL.Query(),
			newBLOBOKResponseIniter(rw, http.StatusOK), req.Body, func(sysErr coreutils.SysError) {
				writeCommonError_V1(rw, sysErr, sysErr.HTTPStatus)
			}, s.requestSender) {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	})
}

func (s *httpService) blobHTTPRequestHandler_Read(numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withValidateForBLOBs(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		if logger.IsVerbose() {
			logger.Verbose("blob read request:", req.URL.String())
		}
		vars := mux.Vars(req)
		existingBLOBIDOrSUID := vars[URLPlaceholder_blobIDOrSUUID]
		if !s.blobRequestHandler.HandleRead(data.appQName, data.wsid, data.header, req.Context(),
			newBLOBOKResponseIniter(rw, http.StatusOK), func(sysErr coreutils.SysError) {
				writeCommonError_V1(rw, sysErr, sysErr.HTTPStatus)
			}, existingBLOBIDOrSUID, s.requestSender) {
			rw.WriteHeader(http.StatusServiceUnavailable)
			rw.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	})
}

func newBLOBOKResponseIniter(r http.ResponseWriter, okStatusCode int) func(headersKV ...string) io.Writer {
	return func(headersKV ...string) io.Writer {
		for i := 0; i < len(headersKV); i += 2 {
			r.Header().Set(headersKV[i], headersKV[i+1])
		}
		r.WriteHeader(okStatusCode)
		return r
	}
}
