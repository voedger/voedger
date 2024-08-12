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
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
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

func (f *implIFederation) UploadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blob coreutils.BLOBReader,
	optFuncs ...coreutils.ReqOptFunc) (blobID istructs.RecordID, err error) {
	uploadBLOBURL := fmt.Sprintf("blob/%s/%d?name=%s&mimeType=%s", appQName.String(), wsid, blob.Name, blob.MimeType)
	resp, err := f.postReader(uploadBLOBURL, blob, optFuncs...)
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

func (f *implIFederation) ReadBLOB(appQName appdef.AppQName, wsid istructs.WSID, blobID istructs.RecordID, optFuncs ...coreutils.ReqOptFunc) (res coreutils.BLOBReader, err error) {
	url := fmt.Sprintf(`blob/%s/%d/%d`, appQName, wsid, blobID)
	optFuncs = append(optFuncs, coreutils.WithResponseHandler(func(httpResp *http.Response) {}))
	resp, err := f.post(url, "", optFuncs...)
	if err != nil {
		return res, err
	}
	if resp.HTTPResp.StatusCode != http.StatusOK {
		return coreutils.BLOBReader{}, nil
	}
	contentDisposition := resp.HTTPResp.Header.Get(coreutils.ContentDisposition)
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return res, err
	}
	res = coreutils.BLOBReader{
		BLOBDesc: coreutils.BLOBDesc{
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

func (f *implIFederation) AdminFunc(relativeURL string, body string, optFuncs ...coreutils.ReqOptFunc) (*coreutils.FuncResponse, error) {
	optFuncs = append(optFuncs, coreutils.WithMethod(http.MethodPost))
	url := fmt.Sprintf("http://127.0.0.1:%d/%s", f.adminPortGetter(), relativeURL)
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
		funcError := coreutils.FuncError{
			SysError: coreutils.SysError{
				HTTPStatus: httpResp.HTTPResp.StatusCode,
			},
			ExpectedHTTPCodes: httpResp.ExpectedHTTPCodes(),
		}
		if len(httpResp.Body) == 0 || httpResp.HTTPResp.StatusCode == http.StatusOK {
			return nil, funcError
		}
		m := map[string]interface{}{}
		if err := json.Unmarshal([]byte(httpResp.Body), &m); err != nil {
			return nil, err
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
		return nil, funcError
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
	resp, err := f.get("n10n/channel?"+params.Encode(), coreutils.WithLongPolling())
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
