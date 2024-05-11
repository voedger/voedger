/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/blobber"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// wrapped ErrUnexpectedStatusCode is returned -> *HTTPResponse contains a valid response body
// otherwise if err != nil (e.g. socket error)-> *HTTPResponse is nil
func (f *implIFederation) post(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	return f.req(relativeURL, body, optFuncs...)
}

func (f *implIFederation) get(relativeURL string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodGet))
	return f.req(relativeURL, "", optFuncs...)
}

func (f *implIFederation) req(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	url := f.federationURL().String() + "/" + relativeURL
	return f.httpClient.Req(url, body, optFuncs...)
}

func (f *implIFederation) UploadBLOBs(appQName istructs.AppQName, wsid istructs.WSID, blobs []blobber.BLOB, optFuncs ...coreutils.ReqOptFunc) (blobIDs []istructs.RecordID, err error) {
	body := bytes.NewBuffer(nil)
	w := multipart.NewWriter(body)
	boundary := "----------------"
	if err := w.SetBoundary(boundary); err != nil {
		// notest
		return nil, err
	}
	for _, blob := range blobs {
		h := textproto.MIMEHeader{}
		h.Set(coreutils.ContentDisposition, fmt.Sprintf(`form-data; name="%s"`, blob.Name))
		h.Set(coreutils.ContentType, "application/x-binary")
		part, err := w.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err = part.Write(blob.Content); err != nil {
			return nil, err
		}
	}
	if err = w.Close(); err != nil {
		// notest
		return nil, err
	}
	url := fmt.Sprintf("blob/%s/%d", appQName, wsid)
	optFuncs = append(optFuncs, coreutils.WithHeaders(coreutils.ContentType,
		fmt.Sprintf("multipart/form-data; boundary=%s", boundary)))
	resp, err := f.post(url, body.String(), optFuncs...)
	if err != nil {
		return nil, err
	}

	blobIDsStrs := strings.Split(resp.Body, ",")
	for _, blobIDStr := range blobIDsStrs {
		blobID, err := strconv.Atoi(blobIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse one of received blob ids: %s: %w", blobIDsStrs, err)
		}
		blobIDs = append(blobIDs, istructs.RecordID(blobID))
	}
	return blobIDs, nil
}

func (f *implIFederation) UploadBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobName string, blobMimeType string,
	blobContent []byte, optFuncs ...coreutils.ReqOptFunc) (blobID istructs.RecordID, err error) {
	uploadBLOBURL := fmt.Sprintf("blob/%s/%d?name=%s&mimeType=%s", appQName.String(), wsid, blobName, blobMimeType)
	resp, err := f.post(uploadBLOBURL, string(blobContent), optFuncs...)
	if err != nil {
		return 0, err
	}
	if resp.HTTPResp.StatusCode == http.StatusOK {
		newBLOBID, err := strconv.Atoi(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("failed to parse the received blobID string: %w", err)
		}
		return istructs.RecordID(newBLOBID), nil
	}
	return istructs.NullRecordID, nil
}

func (f *implIFederation) ReadBLOB(appQName istructs.AppQName, wsid istructs.WSID, blobID istructs.RecordID, optFuncs ...coreutils.ReqOptFunc) (*coreutils.HTTPResponse, error) {
	url := fmt.Sprintf(`blob/%s/%d/%d`, appQName, wsid, blobID)
	return f.post(url, "", optFuncs...)
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

func (f *implIFederation) AdminFunc(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	url := fmt.Sprintf("http://127.0.0.1:%d/api/%s", f.adminPortGetter(), relativeURL)
	httpResp, err := f.httpClient.Req(url, body, optFuncs...)
	return f.httpRespToFuncResp(httpResp, err)
}

func (f *implIFederation) httpRespToFuncResp(httpResp *coreutils.HTTPResponse, httpRespErr error) (*coreutils.FuncResponse, error) {
	isUnexpectedCode := errors.Is(httpRespErr, coreutils.ErrUnexpectedStatusCode)
	if httpRespErr != nil && !isUnexpectedCode {
		return nil, httpRespErr
	}
	if httpResp == nil {
		return nil, nil
	}
	if isUnexpectedCode {
		m := map[string]interface{}{}
		if err := json.Unmarshal([]byte(httpResp.Body), &m); err != nil {
			return nil, err
		}
		if httpResp.HTTPResp.StatusCode == http.StatusOK {
			return nil, coreutils.FuncError{
				SysError: coreutils.SysError{
					HTTPStatus: http.StatusOK,
				},
				ExpectedHTTPCodes: httpResp.ExpectedHTTPCodes(),
			}
		}
		sysErrorMap := m["sys.Error"].(map[string]interface{})
		return nil, coreutils.FuncError{
			SysError: coreutils.SysError{
				HTTPStatus: int(sysErrorMap["HTTPStatus"].(float64)),
				Message:    sysErrorMap["Message"].(string),
			},
			ExpectedHTTPCodes: httpResp.ExpectedHTTPCodes(),
		}
	}
	res := &coreutils.FuncResponse{
		HTTPResponse: httpResp,
		NewIDs:       map[string]int64{},
		CmdResult:    map[string]interface{}{},
	}
	if len(httpResp.Body) == 0 {
		return res, nil
	}
	if err := json.Unmarshal([]byte(httpResp.Body), &res); err != nil {
		return nil, err
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
	resp, err := f.get(fmt.Sprintf("n10n/channel?%s", params.Encode()), coreutils.WithLongPolling())
	if err != nil {
		return nil, nil, err
	}

	subscribed := make(chan interface{})
	offsetsChan = make(OffsetsChan)
	go func() {
		defer close(offsetsChan)
		scanner := bufio.NewScanner(resp.HTTPResp.Body)
		scanner.Split(coreutils.ScanSSE) // разбиваем на кадры sse, разделитель - два new line: "\n\n"
		for scanner.Scan() {
			if resp.HTTPResp.Request.Context().Err() != nil {
				return
			}
			messages := strings.Split(scanner.Text(), "\n") // делим кадр на событие и данные
			var event, data string
			for _, str := range messages { // вычитываем
				if strings.HasPrefix(str, "event: ") {
					event = strings.TrimPrefix(str, "event: ")
				}
				if strings.HasPrefix(str, "data: ") {
					data = strings.TrimPrefix(str, "data: ")
				}
			}
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("received event: %s, data: %s", event, data))
			}
			if event == "channelId" {
				channelID = data
				close(subscribed)
			} else {
				offset, err := strconv.Atoi(data)
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
		f.get(fmt.Sprintf("n10n/unsubscribe?%s", params.Encode()))
		resp.HTTPResp.Body.Close()
		for range offsetsChan {
		}
	}
	return
}
