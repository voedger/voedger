/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

// wrapped ErrUnexpectedStatusCode is returned -> *HTTPResponse contains a valid response body
// otherwise if err != nil (e.g. socket error)-> *HTTPResponse is nil
func (f *implIFederation) post(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	return f.req(relativeURL, body, optFuncs...)
}

func (f *implIFederation) postReader(relativeURL string, bodyReader io.Reader, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	return f.reqReader(relativeURL, bodyReader, optFuncs...)
}

func (f *implIFederation) get(relativeURL string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodGet))
	return f.req(relativeURL, "", optFuncs...)
}

func (f *implIFederation) reqReader(relativeURL string, bodyReader io.Reader, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	url := f.federationURL().String() + "/" + relativeURL
	return f.httpClient.ReqReader(url, bodyReader, optFuncs...)
}

func (f *implIFederation) req(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	url := f.federationURL().String() + "/" + relativeURL
	return f.httpClient.Req(url, body, optFuncs...)
}

func (f *implIFederation) UploadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobReader iblobstorage.BLOBReader, duration iblobstorage.DurationType,
	optFuncs ...coreutils.ReqOptFunc) (blobSUUID iblobstorage.SUUID, err error) {
	ttl, ok := TemporaryBLOBDurationToURLTTL[duration]
	if ok {
		ttl = "&ttl=" + ttl
	}
	uploadBLOBURL := fmt.Sprintf("blob/%s/%d?name=%s&mimeType=%s%s", appQName.String(), wsid,
		url.QueryEscape(blobReader.Name), url.QueryEscape(blobReader.MimeType), ttl)
	resp, err := f.postReader(uploadBLOBURL, blobReader, optFuncs...)
	if err != nil {
		return "", err
	}
	if !slices.Contains(resp.ExpectedHTTPCodes(), resp.HTTPResp.StatusCode) {
		funcErr, err := getFuncError(resp)
		if err != nil {
			return "", err
		}
		return "", funcErr
	}
	if resp.HTTPResp.StatusCode != http.StatusOK {
		return "", nil
	}
	return iblobstorage.SUUID(resp.Body), nil
}

func (f *implIFederation) UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blob iblobstorage.BLOBReader,
	optFuncs ...coreutils.ReqOptFunc) (blobID istructs.RecordID, err error) {
	newBLOBIDStr, err := f.UploadTempBLOB(appQName, wsid, blob, 0, optFuncs...)
	if err != nil {
		return istructs.NullRecordID, err
	}
	newBLOBID, err := strconv.ParseUint(string(newBLOBIDStr), utils.DecimalBase, utils.BitSize64)
	if err != nil {
		return istructs.NullRecordID, fmt.Errorf("failed to parse the received blobID string: %w", err)
	}
	return istructs.RecordID(newBLOBID), nil
}

func (f *implIFederation) ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobID istructs.RecordID, optFuncs ...coreutils.ReqOptFunc) (res iblobstorage.BLOBReader, err error) {
	return f.ReadTempBLOB(appQName, wsid, iblobstorage.SUUID(utils.UintToString(blobID)), optFuncs...)
}

func (f *implIFederation) ReadTempBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobSUUID iblobstorage.SUUID, optFuncs ...coreutils.ReqOptFunc) (res iblobstorage.BLOBReader, err error) {
	url := fmt.Sprintf(`blob/%s/%d/%s`, appQName, wsid, blobSUUID)
	optFuncs = append(optFuncs, coreutils.WithResponseHandler(func(httpResp *http.Response) {}))
	resp, err := f.post(url, "", optFuncs...)
	if err != nil {
		return res, err
	}
	if resp.HTTPResp.StatusCode != http.StatusOK {
		return iblobstorage.BLOBReader{}, nil
	}
	contentDisposition := resp.HTTPResp.Header.Get(coreutils.ContentDisposition)
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return res, err
	}
	res = iblobstorage.BLOBReader{
		DescrType: iblobstorage.DescrType{
			Name:     params["filename"],
			MimeType: resp.HTTPResp.Header.Get(coreutils.ContentType),
		},
		ReadCloser: resp.HTTPResp.Body,
	}
	return res, nil
}

func (f *implIFederation) N10NUpdate(key in10n.ProjectionKey, val int64, optFuncs ...coreutils.ReqOptFunc) error {
	body := fmt.Sprintf(`{"App": "%s","Projection": "%s","WS": %d}`, key.App, key.Projection, key.WS)
	optFuncs = append(optFuncs, coreutils.WithDiscardResponse())
	_, err := f.post(fmt.Sprintf("n10n/update/%d", val), body, optFuncs...)
	return err
}

func (f *implIFederation) GET(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodGet))
	url := f.federationURL().String() + "/" + relativeURL
	return f.httpClient.Req(url, body, optFuncs...)
}

func (f *implIFederation) Func(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error) {
	httpResp, err := f.post(relativeURL, body, optFuncs...)
	return f.httpRespToFuncResp(httpResp, err)
}

func (f *implIFederation) Query(relativeURL string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error) {
	httpResp, err := f.get(relativeURL, optFuncs...)
	return f.httpRespToFuncResp(httpResp, err)
}

func (f *implIFederation) AdminFunc(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	url := fmt.Sprintf("http://127.0.0.1:%d/%s", f.adminPortGetter(), relativeURL)
	httpResp, err := f.httpClient.Req(url, body, optFuncs...)
	return f.httpRespToFuncResp(httpResp, err)
}

func getFuncError(httpResp *coreutils.HTTPResponse) (funcError coreutils.FuncError, err error) {
	funcError = coreutils.FuncError{
		SysError: coreutils.SysError{
			HTTPStatus: httpResp.HTTPResp.StatusCode,
		},
		ExpectedHTTPCodes: httpResp.ExpectedHTTPCodes(),
	}
	if len(httpResp.Body) == 0 || httpResp.HTTPResp.StatusCode == http.StatusOK {
		return funcError, nil
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(httpResp.Body), &m); err != nil {
		return funcError, fmt.Errorf("IFederation: failed to unmarshal response body to FuncErr: %w. Body:\n%s", err, httpResp.Body)
	}
	sysErrorMap := m["sys.Error"].(map[string]interface{})
	errQNameStr, ok := sysErrorMap["QName"].(string)
	if ok {
		errQName, err := appdef.ParseQName(errQNameStr)
		if err != nil {
			errQName = appdef.NewQName("<err>", sysErrorMap["QName"].(string))
		}
		funcError.SysError.QName = errQName
	}
	funcError.SysError.HTTPStatus = int(sysErrorMap["HTTPStatus"].(float64))
	funcError.SysError.Message = sysErrorMap["Message"].(string)
	funcError.SysError.Data, _ = sysErrorMap["Data"].(string)
	return funcError, nil
}

func (f *implIFederation) httpRespToFuncResp(httpResp *coreutils.HTTPResponse, httpRespErr error) (res *coreutils.FuncResponse, err error) {
	isUnexpectedCode := errors.Is(httpRespErr, coreutils.ErrUnexpectedStatusCode)
	if httpRespErr != nil && !isUnexpectedCode {
		return nil, httpRespErr
	}
	if httpResp == nil {
		return nil, nil
	}
	if isUnexpectedCode {
		funcError, err := getFuncError(httpResp)
		if err != nil {
			return nil, err
		}
		return nil, funcError
	}
	res = &coreutils.FuncResponse{
		CommandResponse: coreutils.CommandResponse{
			NewIDs:    map[string]istructs.RecordID{},
			CmdResult: map[string]interface{}{},
		},
		HTTPResponse: httpResp,
	}
	if len(httpResp.Body) == 0 {
		return res, nil
	}
	if strings.HasPrefix(httpResp.HTTPResp.Request.URL.Path, "/api/v2/") {
		// TODO: eliminate this after https://github.com/voedger/voedger/issues/1313
		err = json.Unmarshal([]byte(httpResp.Body), &res.APIV2Response)
	} else {
		err = json.Unmarshal([]byte(httpResp.Body), &res)
	}
	if err != nil {
		return nil, fmt.Errorf("IFederation: failed to unmarshal response body to FuncResponse: %w. Body:\n%s", err, httpResp.Body)
	}
	if res.SysError.HTTPStatus > 0 && res.ExpectedSysErrorCode() > 0 && res.ExpectedSysErrorCode() != res.SysError.HTTPStatus {
		return nil, fmt.Errorf("sys.Error actual status %d, expected %v: %s", res.SysError.HTTPStatus, res.ExpectedSysErrorCode(), res.SysError.Message)
	}
	return res, nil
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

func (f *implIFederation) N10NSubscribe(projectionKey in10n.ProjectionKey) (offsetsChan OffsetsChan, unsubscribe func(), err error) {
	channelID := ""
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
	resp, err := f.get("n10n/channel?"+params.Encode(), coreutils.WithLongPolling())
	if err != nil {
		return nil, nil, err
	}

	subscribed := make(chan interface{})
	offsetsChan = make(OffsetsChan)
	go func() {
		defer close(offsetsChan)
		scanner := bufio.NewScanner(resp.HTTPResp.Body)
		scanner.Split(coreutils.ScanSSE) // split by sse frames, separator is "\n\n"
		for scanner.Scan() {
			if resp.HTTPResp.Request.Context().Err() != nil {
				return
			}
			messages := strings.Split(scanner.Text(), "\n") // split the frame by ecent and data
			var event, data string
			for _, str := range messages { // read out
				if strings.HasPrefix(str, "event: ") {
					event = strings.TrimPrefix(str, "event: ")
				}
				if strings.HasPrefix(str, "data: ") {
					data = strings.TrimPrefix(str, "data: ")
				}
			}
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("received event from server: %s, data: %s", event, data))
			}
			if event == "channelId" {
				channelID = data
				close(subscribed)
			} else {
				offset, err := strconv.ParseUint(data, utils.DecimalBase, utils.BitSize64)
				if err != nil {
					panic(fmt.Sprint("failed to parse offset", data, err))
				}
				offsetsChan <- istructs.Offset(offset)
			}
		}
	}()

	<-subscribed

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
		_, err := f.get("n10n/unsubscribe?"+params.Encode(), coreutils.WithDiscardResponse())
		if err != nil {
			logger.Error("unsubscribe failed", err.Error())
		}
		resp.HTTPResp.Body.Close()
		for range offsetsChan {
		}
	}
	return
}
