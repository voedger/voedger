/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

// wrapped ErrUnexpectedStatusCode is returned -> *HTTPResponse contains a valid response body
// otherwise if err != nil (e.g. socket error)-> *HTTPResponse is nil
func (f *implIFederation) post(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	optFuncs = append(optFuncs, httpu.WithDefaultMethod(http.MethodPost))
	return f.reqFederation(relativeURL, body, optFuncs...)
}

func (f *implIFederation) postReader(relativeURL string, bodyReader io.Reader, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	optFuncs = append(optFuncs, httpu.WithDefaultMethod(http.MethodPost))
	return f.reqReader(relativeURL, bodyReader, optFuncs...)
}

func (f *implIFederation) get(relativeURL string, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	optFuncs = append(optFuncs, httpu.WithDefaultMethod(http.MethodGet))
	return f.reqFederation(relativeURL, "", optFuncs...)
}

func (f *implIFederation) reqFederation(apiPath string, body string, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	url := f.federationURL().String() + "/" + apiPath
	return f.reqURL(url, body, optFuncs...)
}

func (f *implIFederation) reqReader(relativeURL string, bodyReader io.Reader, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	url := f.federationURL().String() + "/" + relativeURL
	optFuncs = append(slices.Clone(f.defaultReqOptFuncs), optFuncs...)
	return f.httpClient.ReqReader(f.vvmCtx, url, bodyReader, optFuncs...)
}

func (f *implIFederation) reqURL(url string, body string, optFuncs ...httpu.ReqOptFunc) (*httpu.HTTPResponse, error) {
	optFuncs = append(slices.Clone(f.defaultReqOptFuncs), optFuncs...)
	return f.httpClient.Req(f.vvmCtx, url, body, optFuncs...)
}

func (f *implIFederation) UploadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobReader iblobstorage.BLOBReader, duration iblobstorage.DurationType,
	optFuncs ...httpu.ReqOptFunc) (blobSUUID iblobstorage.SUUID, err error) {
	ttl, ok := TemporaryBLOBDurationToURLTTL[duration]
	if !ok {
		return "", fmt.Errorf("unsupported duration: %d", duration)
	}
	uploadBLOBURL := fmt.Sprintf("api/v2/apps/%s/%s/workspaces/%d/tblobs", appQName.Owner(), appQName.Name(), wsid)
	optFuncs = append(optFuncs, httpu.WithHeaders(
		coreutils.BlobName, blobReader.Name,
		httpu.ContentType, blobReader.ContentType,
		"TTL", ttl,
	))
	resp, err := f.postReader(uploadBLOBURL, blobReader, optFuncs...)
	if err != nil {
		return "", err
	}
	if !slices.Contains(resp.Opts.ExpectedHTTPCodes(), resp.HTTPResp.StatusCode) {
		sysErr, err := getSysError(resp)
		if err != nil {
			return "", err
		}
		return "", sysErr
	}
	if resp.HTTPResp.StatusCode != http.StatusOK && resp.HTTPResp.StatusCode != http.StatusCreated {
		return "", nil
	}
	matches := blobCreateTempRespRE.FindStringSubmatch(resp.Body)
	if len(matches) < 2 {
		// notest
		return "", errors.New("wrong blob create response: " + resp.Body)
	}
	return iblobstorage.SUUID(matches[1]), nil
}

func (f *implIFederation) UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobReader iblobstorage.BLOBReader,
	optFuncs ...httpu.ReqOptFunc) (blobID istructs.RecordID, err error) {
	uploadBLOBURL := fmt.Sprintf("api/v2/apps/%s/%s/workspaces/%d/docs/%s/blobs/%s",
		appQName.Owner(), appQName.Name(), wsid, blobReader.OwnerRecord, blobReader.OwnerRecordField)
	optFuncs = append(optFuncs, httpu.WithHeaders(
		coreutils.BlobName, blobReader.Name,
		httpu.ContentType, blobReader.ContentType,
	))
	resp, err := f.postReader(uploadBLOBURL, blobReader, optFuncs...)
	if err != nil {
		return istructs.NullRecordID, err
	}
	if !slices.Contains(resp.Opts.ExpectedHTTPCodes(), resp.HTTPResp.StatusCode) {
		sysErr, err := getSysError(resp)
		if err != nil {
			return istructs.NullRecordID, err
		}
		return istructs.NullRecordID, sysErr
	}
	if resp.HTTPResp.StatusCode != http.StatusCreated {
		return istructs.NullRecordID, nil
	}
	matches := blobCreatePersistentRespRE.FindStringSubmatch(resp.Body)
	if len(matches) != 2 {
		// notest
		return istructs.NullRecordID, errors.New("wrong blob create response: " + resp.Body)
	}
	newBLOBIDIntf, err := coreutils.ClarifyJSONNumber(json.Number(matches[1]), appdef.DataKind_RecordID)
	if err != nil {
		// notest
		return istructs.NullRecordID, fmt.Errorf("failed to parse the received blobID string: %w", err)
	}
	return newBLOBIDIntf.(istructs.RecordID), nil
}

func (f *implIFederation) ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, ownerRecord appdef.QName, ownerRecordField appdef.FieldName, ownerID istructs.RecordID,
	optFuncs ...httpu.ReqOptFunc) (res iblobstorage.BLOBReader, err error) {
	url := fmt.Sprintf(`api/v2/apps/%s/%s/workspaces/%d/docs/%s/%d/blobs/%s`, appQName.Owner(), appQName.Name(), wsid, ownerRecord, ownerID, ownerRecordField)
	optFuncs = append(optFuncs, httpu.WithResponseHandler(func(httpResp *http.Response) {}))
	resp, err := f.get(url, optFuncs...)
	if err != nil {
		return res, err
	}
	if resp.HTTPResp.StatusCode != http.StatusOK {
		return iblobstorage.BLOBReader{}, nil
	}
	res = iblobstorage.BLOBReader{
		DescrType: iblobstorage.DescrType{
			Name:        resp.HTTPResp.Header.Get(coreutils.BlobName),
			ContentType: resp.HTTPResp.Header.Get(httpu.ContentType),
		},
		ReadCloser: resp.HTTPResp.Body,
	}
	return res, nil
}

func (f *implIFederation) ReadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobSUUID iblobstorage.SUUID, optFuncs ...httpu.ReqOptFunc) (res iblobstorage.BLOBReader, err error) {
	url := fmt.Sprintf(`api/v2/apps/%s/%s/workspaces/%d/tblobs/%s`, appQName.Owner(), appQName.Name(), wsid, blobSUUID)
	optFuncs = append(optFuncs, httpu.WithResponseHandler(func(httpResp *http.Response) {}))
	resp, err := f.get(url, optFuncs...)
	if err != nil {
		return res, err
	}
	if resp.HTTPResp.StatusCode != http.StatusOK {
		return iblobstorage.BLOBReader{}, nil
	}
	res = iblobstorage.BLOBReader{
		DescrType: iblobstorage.DescrType{
			Name:        resp.HTTPResp.Header.Get(coreutils.BlobName),
			ContentType: resp.HTTPResp.Header.Get(httpu.ContentType),
		},
		ReadCloser: resp.HTTPResp.Body,
	}
	return res, nil
}

func (f *implIFederation) N10NUpdate(key in10n.ProjectionKey, val int64, optFuncs ...httpu.ReqOptFunc) error {
	body := fmt.Sprintf(`{"App": "%s","Projection": "%s","WS": %d}`, key.App, key.Projection, key.WS)
	optFuncs = append(optFuncs, httpu.WithDiscardResponse())
	_, err := f.post(fmt.Sprintf("n10n/update/%d", val), body, optFuncs...)
	return err
}

func (f *implIFederation) Func(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error) {
	httpResp, err := f.post(relativeURL, body, optFuncs...)
	return HTTPRespToFuncResp(httpResp, err)
}

func (f *implIFederation) Query(relativeURL string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error) {
	httpResp, err := f.get(relativeURL, optFuncs...)
	return HTTPRespToFuncResp(httpResp, err)
}

func (f *implIFederation) AdminFunc(relativeURL string, body string, optFuncs ...httpu.ReqOptFunc) (*FuncResponse, error) {
	optFuncs = append(optFuncs, httpu.WithMethod(http.MethodPost))
	url := fmt.Sprintf("http://127.0.0.1:%d/%s", f.adminPortGetter(), relativeURL)
	optFuncs = append(slices.Clone(f.defaultReqOptFuncs), optFuncs...)
	httpResp, err := f.httpClient.Req(f.vvmCtx, url, body, optFuncs...)
	return HTTPRespToFuncResp(httpResp, err)
}

func getSysError(httpResp *httpu.HTTPResponse) (sysError coreutils.SysError, err error) {
	sysError = coreutils.SysError{
		HTTPStatus: httpResp.HTTPResp.StatusCode,
	}
	if len(httpResp.Body) == 0 || httpResp.HTTPResp.StatusCode == http.StatusOK {
		return sysError, nil
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(httpResp.Body), &m); err != nil {
		return sysError, fmt.Errorf("IFederation: failed to unmarshal response body to FuncErr: %w. Body:\n%s", err, httpResp.Body)
	}
	sysErrorIntf, hasSysError := m["sys.Error"]
	if hasSysError {
		sysErrorMap := sysErrorIntf.(map[string]interface{})
		errQNameStr, ok := sysErrorMap["QName"].(string)
		if ok {
			errQName, err := appdef.ParseQName(errQNameStr)
			if err != nil {
				errQName = appdef.NewQName("<err>", sysErrorMap["QName"].(string))
			}
			sysError.QName = errQName
		}
		sysError.HTTPStatus = int(sysErrorMap["HTTPStatus"].(float64))
		sysError.Message = sysErrorMap["Message"].(string)
		sysError.Data, _ = sysErrorMap["Data"].(string)
	} else {
		if apiV2QueryError, ok := m["error"]; ok {
			m = apiV2QueryError.(map[string]interface{})
		}
		if commonErrorStatusIntf, ok := m["status"]; ok {
			sysError.HTTPStatus = int(commonErrorStatusIntf.(float64))
		}
		if commonErrorMessageIntf, ok := m["message"]; ok {
			sysError.Message = commonErrorMessageIntf.(string)
		}
	}
	return sysError, nil
}

func (f *implIFederation) URLStr() string {
	return f.federationURL().String()
}

func (f *implIFederation) Port() int {
	res, err := strconv.Atoi(f.federationURL().Port())
	if err != nil {
		// notest
		panic(err)
	}
	return res
}

func (f *implIFederation) N10NSubscribe(projectionKey in10n.ProjectionKey, optFuncs ...httpu.ReqOptFunc) (offsetsChan OffsetsChan, unsubscribe func(), err error) {
	query := fmt.Sprintf(`
		{
			"SubjectLogin": "test_%d",
			"ProjectionKey": [
				{
					"App":"%s",
					"Projection":"%s",
					"WS":%d
				}
			]
		}`, projectionKey.WS, projectionKey.App, projectionKey.Projection, projectionKey.WS)
	params := url.Values{}
	params.Add("payload", query)
	opts := slices.Clone(optFuncs)
	opts = append(opts, httpu.WithLongPolling())
	resp, err := f.get("n10n/channel?"+params.Encode(), opts...)
	if err != nil {
		return nil, nil, err
	}

	offsetsChan, channelID, waitForDone := ListenSSEEvents(resp.HTTPResp.Request.Context(), resp.HTTPResp.Body)

	unsubscribe = func() {
		body := fmt.Sprintf(`
			{
				"Channel": "%s",
				"ProjectionKey":[
					{
						"App": "%s",
						"Projection":"%s",
						"WS":%d
					}
				]
			}
		`, channelID, projectionKey.App, projectionKey.Projection, projectionKey.WS)
		params := url.Values{}
		params.Add("payload", body)
		_, err := f.get("n10n/unsubscribe?" + params.Encode())
		if err != nil {
			logger.Error("unsubscribe failed", err.Error())
		}
		resp.HTTPResp.Body.Close()
		for range offsetsChan {
		}
		waitForDone()
	}
	return
}

func (f *implIFederation) dummy() {}

func (f *implIFederation) WithRetry() IFederationWithRetry {
	return &implIFederation{
		httpClient:      f.httpClient,
		federationURL:   f.federationURL,
		adminPortGetter: f.adminPortGetter,
		defaultReqOptFuncs: []httpu.ReqOptFunc{
			httpu.WithRetryPolicy(f.policyOptsForWithRetry...),
		},
		vvmCtx:                 f.vvmCtx,
		policyOptsForWithRetry: f.policyOptsForWithRetry,
	}
}
