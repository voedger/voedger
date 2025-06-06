/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"bufio"
	"context"
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
