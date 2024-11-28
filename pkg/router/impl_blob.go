/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/istructs"
)

type blobWriteDetailsSingle struct {
	name     string
	mimeType string
	duration iblobstorage.DurationType
}

type blobWriteDetailsMultipart struct {
	boundary string
	duration iblobstorage.DurationType
}

type blobReadDetails_Persistent struct {
	blobID istructs.RecordID
}

type blobReadDetails_Temporary struct {
	suuid iblobstorage.SUUID
}

type blobBaseMessage struct {
	req         *http.Request
	resp        http.ResponseWriter
	doneChan    chan struct{}
	wsid        istructs.WSID
	appQName    appdef.AppQName
	header      map[string][]string
	blobMaxSize iblobstorage.BLOBMaxSizeType
}

type blobMessage struct {
	blobBaseMessage
	// could be blobReadDetails_Temporary or blobReadDetails_Persistent
	blobDetails interface{}
}

func (bm *blobBaseMessage) Release() {
	bm.req.Body.Close()
}

func blobReadMessageHandler(bbm blobBaseMessage, blobReadDetails interface{}, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration) {
	defer close(bbm.doneChan)

	// request to VVM to check the principalToken
	req := ibus.Request{
		Method:   ibus.HTTPMethodPOST,
		WSID:     bbm.wsid,
		AppQName: bbm.appQName.String(),
		Resource: "q.sys.DownloadBLOBAuthnz",
		Header:   bbm.header,
		Body:     []byte(`{}`),
		Host:     localhost,
	}
	blobHelperResp, _, _, err := bus.SendRequest2(bbm.req.Context(), req, busTimeout)
	if err != nil {
		WriteTextResponse(bbm.resp, "failed to exec q.sys.DownloadBLOBAuthnz: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if blobHelperResp.StatusCode != http.StatusOK {
		WriteTextResponse(bbm.resp, "q.sys.DownloadBLOBAuthnz returned error: "+string(blobHelperResp.Data), blobHelperResp.StatusCode)
		return
	}

	// read the BLOB
	var blobKey iblobstorage.IBLOBKey
	switch typedKey := blobReadDetails.(type) {
	case blobReadDetails_Persistent:
		blobKey = &iblobstorage.PersistentBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bbm.wsid,
			BlobID:       typedKey.blobID,
		}
	case blobReadDetails_Temporary:
		blobKey = &iblobstorage.TempBLOBKeyType{
			ClusterAppID: istructs.ClusterAppID_sys_blobber,
			WSID:         bbm.wsid,
			SUUID:        typedKey.suuid,
		}
	default:
		// notest
		panic(fmt.Sprintf("unexpected blobReadDetails: %T", blobReadDetails))
	}
	stateWriterDiscard := func(state iblobstorage.BLOBState) error {
		if state.Status != iblobstorage.BLOBStatus_Completed {
			return errors.New("blob is not completed")
		}
		if len(state.Error) > 0 {
			return errors.New(state.Error)
		}
		bbm.resp.Header().Set(coreutils.ContentType, state.Descr.MimeType)
		bbm.resp.Header().Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, state.Descr.Name))
		bbm.resp.WriteHeader(http.StatusOK)
		return nil
	}
	if err := blobStorage.ReadBLOB(bbm.req.Context(), blobKey, stateWriterDiscard, bbm.resp, iblobstoragestg.RLimiter_Null); err != nil {
		logger.Error(fmt.Sprintf("failed to read or send BLOB: id %d, appQName %s, wsid %d: %s", blobKey.ID(), bbm.appQName, bbm.wsid, err.Error()))
		if errors.Is(err, iblobstorage.ErrBLOBNotFound) {
			WriteTextResponse(bbm.resp, err.Error(), http.StatusNotFound)
			return
		}
		WriteTextResponse(bbm.resp, err.Error(), http.StatusInternalServerError)
	}
}

func writeBLOB(ctx context.Context, wsid istructs.WSID, appQName string, header map[string][]string, resp http.ResponseWriter,
	blobName, blobMimeType string, blobDuration iblobstorage.DurationType, blobStorage iblobstorage.IBLOBStorage, body io.ReadCloser,
	blobMaxSize iblobstorage.BLOBMaxSizeType, bus ibus.IBus, busTimeout time.Duration) (blobID istructs.RecordID) {
	// request VVM for check the principalToken and get a blobID
	req := ibus.Request{
		Method:   ibus.HTTPMethodPOST,
		WSID:     wsid,
		AppQName: appQName,
		Resource: "c.sys.UploadBLOBHelper",
		Body:     []byte(`{}`),
		Header:   header,
		Host:     localhost,
	}
	blobHelperResp, _, _, err := bus.SendRequest2(ctx, req, busTimeout)
	if err != nil {
		WriteTextResponse(resp, "failed to exec c.sys.UploadBLOBHelper: "+err.Error(), http.StatusInternalServerError)
		return 0
	}
	if blobHelperResp.StatusCode != http.StatusOK {
		WriteTextResponse(resp, "c.sys.UploadBLOBHelper returned error: "+string(blobHelperResp.Data), blobHelperResp.StatusCode)
		return 0
	}
	cmdResp := map[string]interface{}{}
	if err := json.Unmarshal(blobHelperResp.Data, &cmdResp); err != nil {
		WriteTextResponse(resp, "failed to json-unmarshal c.sys.UploadBLOBHelper result: "+err.Error(), http.StatusInternalServerError)
		return 0
	}
	newIDs := cmdResp["NewIDs"].(map[string]interface{})

	blobID = istructs.RecordID(newIDs["1"].(float64))

	// write the BLOB
	if blobDuration > 0 {
		// temporary blob
		key := iblobstorage.TempBLOBKeyType {
			ClusterAppID:  istructs.ClusterAppID_sys_blobber,
			WSID: wsid,
			SUUID: iblobstorage.NewSUUID(),
		}
		blobStorage.WriteTempBLOB(ctx, key, descr, body, )
	}
	key := iblobstorage.PersistentBLOBKeyType{
		AppID: istructs.ClusterAppID_sys_blobber,
		WSID:  wsid,
		ID:    blobID,
	}
	descr := iblobstorage.DescrType{
		Name:     blobName,
		MimeType: blobMimeType,
	}

	if err := blobStorage.WriteBLOB(ctx, key, descr, body, blobMaxSize); err != nil {
		if errors.Is(err, iblobstorage.ErrBLOBSizeQuotaExceeded) {
			WriteTextResponse(resp, fmt.Sprintf("blob size quouta exceeded (max %d allowed)", blobMaxSize), http.StatusForbidden)
			return 0
		}
		WriteTextResponse(resp, err.Error(), http.StatusInternalServerError)
		return 0
	}

	// set WDoc<sys.BLOB>.status = BLOBStatus_Completed
	req.Resource = "c.sys.CUD"
	req.Body = []byte(fmt.Sprintf(`{"cuds":[{"sys.ID": %d,"fields":{"status":%d}}]}`, blobID, iblobstorage.BLOBStatus_Completed))
	cudWDocBLOBUpdateResp, _, _, err := bus.SendRequest2(ctx, req, busTimeout)
	if err != nil {
		WriteTextResponse(resp, "failed to exec c.sys.CUD: "+err.Error(), http.StatusInternalServerError)
		return 0
	}
	if cudWDocBLOBUpdateResp.StatusCode != http.StatusOK {
		WriteTextResponse(resp, "c.sys.CUD returned error: "+string(cudWDocBLOBUpdateResp.Data), cudWDocBLOBUpdateResp.StatusCode)
		return 0
	}

	return blobID
}

func blobWriteMessageHandlerMultipart(bbm blobBaseMessage, blobStorage iblobstorage.IBLOBStorage, blobDetails blobWriteDetailsMultipart,
	bus ibus.IBus, busTimeout time.Duration) {
	defer close(bbm.doneChan)

	r := multipart.NewReader(bbm.req.Body, blobDetails.boundary)
	var part *multipart.Part
	var err error
	blobIDs := []string{}
	partNum := 0
	for err == nil {
		part, err = r.NextPart()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				WriteTextResponse(bbm.resp, "failed to parse multipart: "+err.Error(), http.StatusBadRequest)
				return
			} else if partNum == 0 {
				WriteTextResponse(bbm.resp, "empty multipart request", http.StatusBadRequest)
				return
			}
			break
		}
		contentDisposition := part.Header.Get(coreutils.ContentDisposition)
		mediaType, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			WriteTextResponse(bbm.resp, fmt.Sprintf("failed to parse Content-Disposition of part number %d: %s", partNum, contentDisposition), http.StatusBadRequest)
		}
		if mediaType != "form-data" {
			WriteTextResponse(bbm.resp, fmt.Sprintf("unsupported ContentDisposition mediaType of part number %d: %s", partNum, mediaType), http.StatusBadRequest)
		}
		contentType := part.Header.Get(coreutils.ContentType)
		if len(contentType) == 0 {
			contentType = coreutils.ApplicationXBinary
		}
		part.Header[coreutils.Authorization] = bbm.header[coreutils.Authorization] // add auth header for c.sys.*BLOBHelper
		blobID := writeBLOB(bbm.req.Context(), bbm.wsid, bbm.appQName.String(), part.Header, bbm.resp,
			params["name"], contentType, blobDetails.duration, blobStorage, part, bbm.blobMaxSize, bus, busTimeout)
		if blobID == 0 {
			return // request handled
		}
		blobIDs = append(blobIDs, utils.UintToString(blobID))
		partNum++
	}
	WriteTextResponse(bbm.resp, strings.Join(blobIDs, ","), http.StatusOK)
}

func blobWriteMessageHandlerSingle(bbm blobBaseMessage, blobWriteDetails blobWriteDetailsSingle, blobStorage iblobstorage.IBLOBStorage, header map[string][]string,
	bus ibus.IBus, busTimeout time.Duration) {
	defer close(bbm.doneChan)

	blobID := writeBLOB(bbm.req.Context(), bbm.wsid, bbm.appQName.String(), header, bbm.resp, blobWriteDetails.name,
		blobWriteDetails.mimeType, blobWriteDetails.duration, blobStorage, bbm.req.Body, bbm.blobMaxSize, bus, busTimeout)
	if blobID > 0 {
		WriteTextResponse(bbm.resp, utils.UintToString(blobID), http.StatusOK)
	}
}

// ctx here is VVM context. It used to track VVM shutdown. Blobber will use the request's context
func blobMessageHandler(vvmCtx context.Context, sc iprocbus.ServiceChannel, blobStorage iblobstorage.IBLOBStorage, bus ibus.IBus, busTimeout time.Duration) {
	for vvmCtx.Err() == nil {
		select {
		case mesIntf := <-sc:
			blobMessage := mesIntf.(blobMessage)
			switch blobDetails := blobMessage.blobDetails.(type) {
			case blobReadDetails_Persistent, blobReadDetails_Temporary:
				blobReadMessageHandler(blobMessage.blobBaseMessage, blobDetails, blobStorage, bus, busTimeout)
			case blobWriteDetailsSingle:
				blobWriteMessageHandlerSingle(blobMessage.blobBaseMessage, blobDetails, blobStorage, blobMessage.header, bus, busTimeout)
			case blobWriteDetailsMultipart:
				blobWriteMessageHandlerMultipart(blobMessage.blobBaseMessage, blobStorage, blobDetails, bus, busTimeout)
			}
		case <-vvmCtx.Done():
			return
		}
	}
}

func (s *httpService) blobRequestHandler(resp http.ResponseWriter, req *http.Request, details interface{}) {
	vars := mux.Vars(req)
	wsid, err := strconv.ParseUint(vars[WSID], utils.DecimalBase, utils.BitSize64)
	if err != nil {
		// notest: checked by router url rule
		panic(err)
	}
	mes := blobMessage{
		blobBaseMessage: blobBaseMessage{
			req:         req,
			resp:        resp,
			wsid:        istructs.WSID(wsid),
			doneChan:    make(chan struct{}),
			appQName:    appdef.NewAppQName(vars[AppOwner], vars[AppName]),
			header:      req.Header,
			blobMaxSize: s.BLOBMaxSize,
		},
		blobDetails: details,
	}
	if _, ok := mes.blobBaseMessage.header[coreutils.Authorization]; !ok {
		cookie, err := req.Cookie(coreutils.Authorization)
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			// notest
			panic(err)
		}
		val, err := url.QueryUnescape(cookie.Value)
		if err != nil {
			// notest
			panic(err)
		}
		// authorization token in cookies -> q.sys.DownloadBLOBAuthnz requires it in headers
		mes.blobBaseMessage.header[coreutils.Authorization] = []string{val}
	}
	if !s.BlobberParams.procBus.Submit(0, 0, mes) {
		resp.WriteHeader(http.StatusServiceUnavailable)
		resp.Header().Add("Retry-After", strconv.Itoa(s.BlobberParams.RetryAfterSecondsOn503))
		return
	}
	<-mes.doneChan
}

func (s *httpService) blobReadRequestHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		var blobReadDetails interface{}
		if len(blobID) > 40 {
			// consider the blobID contains SUUID of a temporary BLOB
			blobReadDetails = blobReadDetails_Temporary{
				suuid: blobID,
			}
		} else {
			// conider the BLOB is persistent
			blobID, err := strconv.ParseUint(vars[blobID], utils.DecimalBase, utils.BitSize64)
			if err != nil {
				// notest: checked by router url rule
				panic(err)
			}
			blobReadDetails = blobReadDetails_Persistent{
				blobID: istructs.RecordID(blobID),
			}
		}
		principalToken := headerOrCookieAuth(resp, req)
		if len(principalToken) == 0 {
			return
		}
		s.blobRequestHandler(resp, req, blobReadDetails)
	}
}

func (s *httpService) blobWriteRequestHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		principalToken, isHandled := headerAuth(resp, req)
		if len(principalToken) == 0 {
			if !isHandled {
				writeUnauthorized(resp)
			}
			return
		}

		queryParamName, queryParamMimeType, boundary, duration, ok := getBlobParams(resp, req)
		if !ok {
			return
		}

		if len(queryParamName) > 0 {
			s.blobRequestHandler(resp, req, blobWriteDetailsSingle{
				name:     queryParamName,
				mimeType: queryParamMimeType,
				duration: duration,
			})
		} else {
			s.blobRequestHandler(resp, req, blobWriteDetailsMultipart{
				boundary: boundary,
				duration: duration,
			})
		}
	}
}

func headerAuth(rw http.ResponseWriter, req *http.Request) (principalToken string, isHandled bool) {
	authHeader := req.Header.Get(coreutils.Authorization)
	if len(authHeader) > 0 {
		if len(authHeader) < bearerPrefixLen || authHeader[:bearerPrefixLen] != coreutils.BearerPrefix {
			writeUnauthorized(rw)
			return "", true
		}
		return authHeader[bearerPrefixLen:], false
	}
	return "", false
}

func headerOrCookieAuth(rw http.ResponseWriter, req *http.Request) (principalToken string) {
	principalToken, isHandled := headerAuth(rw, req)
	if isHandled {
		return ""
	}
	if len(principalToken) > 0 {
		return principalToken
	}
	for _, c := range req.Cookies() {
		if c.Name == coreutils.Authorization {
			val, err := url.QueryUnescape(c.Value)
			if err != nil {
				WriteTextResponse(rw, "failed to unescape cookie '"+c.Value+"'", http.StatusBadRequest)
				return ""
			}
			if len(val) < bearerPrefixLen || val[:bearerPrefixLen] != coreutils.BearerPrefix {
				writeUnauthorized(rw)
				return ""
			}
			return val[bearerPrefixLen:]
		}
	}
	writeUnauthorized(rw)
	return ""
}

// determines BLOBs write kind: name+mimeType in query params -> single BLOB, body is BLOB content, otherwise -> body is multipart/form-data
// (is multipart/form-data) == len(boundary) > 0
func getBlobParams(rw http.ResponseWriter, req *http.Request) (name, mimeType, boundary string, duration iblobstorage.DurationType, ok bool) {
	badRequest := func(msg string) {
		WriteTextResponse(rw, msg, http.StatusBadRequest)
	}
	values := req.URL.Query()
	nameQuery, isSingleBLOB := values["name"]
	mimeTypeQuery, hasMimeType := values["mimeType"]
	ttlQuery := values["ttl"]
	if (isSingleBLOB && !hasMimeType) || (!isSingleBLOB && hasMimeType) {
		badRequest("both name and mimeType query params must be specified")
		return
	}

	if len(ttlQuery) > 0 {
		// temporary BLOB
		ttl := ttlQuery[0]
		ttlSupported := false
		if duration, ttlSupported = temporaryBLOBTTLs[ttl]; !ttlSupported {
			badRequest(`"1d" is only supported for now for temporary blob ttl`)
			return
		}
	}

	contentType := req.Header.Get(coreutils.ContentType)
	if isSingleBLOB {
		if contentType == "multipart/form-data" {
			badRequest(`name+mimeType query params and "multipart/form-data" Content-Type header are mutual exclusive`)
			return
		}
		name = nameQuery[0]
		mimeType = mimeTypeQuery[0]
		ok = true
		return
	}
	if len(contentType) == 0 {
		badRequest(`neither "name"+"mimeType" query params nor Content-Type header is not provided`)
		return
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		badRequest("failed ot parse Content-Type header: " + contentType)
		return
	}
	if mediaType != "multipart/form-data" {
		badRequest("name+mimeType query params are not provided -> Content-Type must be mutipart/form-data but actual is " + contentType)
		return
	}
	boundary = params["boundary"]
	if len(boundary) == 0 {
		badRequest("boundary of multipart/form-data is not specified")
		return
	}
	return name, mimeType, boundary, duration, true
}
