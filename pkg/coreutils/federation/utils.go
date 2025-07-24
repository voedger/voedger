/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

// launches listening for sse events from body reader in a separate goroutine
// returns when the first sse packet with channelID is came from the server, i.e. when the subscribing is actually done
// if caller side unsubscribes from events, then it must:
// - read out offsetsChan
// - call waitForDone() to ensure the goroutine finished
func ListenSSEEvents(ctx context.Context, body io.Reader) (offsetsChan OffsetsChan, channelID in10n.ChannelID, waitForDone func()) {
	offsetsChan = make(OffsetsChan)
	subscribed := make(chan interface{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer close(offsetsChan)
		defer wg.Done()
		scanner := bufio.NewScanner(body)
		scanner.Split(coreutils.ScanSSE) // split by sse frames, separator is "\n\n"
		for scanner.Scan() {
			if ctx.Err() != nil {
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
				channelID = in10n.ChannelID(data)
				close(subscribed)
			} else {
				offset, err := strconv.ParseUint(data, utils.DecimalBase, utils.BitSize64)
				if err != nil {
					// notest
					panic(fmt.Sprint("failed to parse offset", data, err))
				}
				offsetsChan <- istructs.Offset(offset)
			}
		}
	}()

	<-subscribed
	return offsetsChan, channelID, func() { wg.Wait() }
}

func HTTPRespToFuncResp(httpResp *coreutils.HTTPResponse, httpRespErr error) (res *coreutils.FuncResponse, err error) {
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
		if httpResp.HTTPResp.Header.Get(coreutils.ContentType) == coreutils.ContentType_ApplicationJSON {
			if err = json.Unmarshal([]byte(httpResp.Body), &res.QPv2Response); err == nil {
				err = json.Unmarshal([]byte(httpResp.Body), &res.CommandResponse)
			}
		}
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
