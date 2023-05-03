/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"bufio"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/untillpro/airs-bp3/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

// Test_Race_n10n_perSubject: Just Create channel
func Test_Race_n10n_perSubject(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	wg := &sync.WaitGroup{}
	cnt := 100
	resps := make(chan *utils.HTTPResponse, cnt)
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func(ai int) {
			defer wg.Done()
			resps <- createChannel(hit, ai)
		}(i)
	}
	wg.Wait()
	close(resps)

	for resp := range resps {
		resp.HTTPResp.Body.Close()
	}
}

// Test_Race_n10nCHS: Create channel and read event
func Test_Race_n10nCHS(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	wg := sync.WaitGroup{}
	wgSubscribe := &sync.WaitGroup{}
	cnt := 100
	resps := make(chan *utils.HTTPResponse, cnt)
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		wgSubscribe.Add(1)
		go func(ai int) {
			defer wg.Done()
			resp := createChannel(hit, ai)
			subscribe(wgSubscribe, resp)
			resps <- resp
		}(i)
	}
	wg.Wait()
	close(resps)

	for resp := range resps {
		resp.HTTPResp.Body.Close()
	}
	wgSubscribe.Wait()
}

// Test_Race_n10nCHSU: Create channel,  read event, send update
func Test_Race_n10nCHSU(t *testing.T) {
	if it.IsCassandraStorage() {
		return
	}
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	wg := sync.WaitGroup{}
	wgSubscribe := &sync.WaitGroup{}
	cnt := 10
	resps := make(chan *utils.HTTPResponse, cnt)
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		wgSubscribe.Add(1)
		go func(ai int) {
			defer wg.Done()
			resp := createChannel(hit, ai)
			subscribe(wgSubscribe, resp)
			update(hit, ai)
			resps <- resp
		}(i)
	}
	wg.Wait()
	close(resps)

	for resp := range resps {
		resp.HTTPResp.Body.Close()
	}
	wgSubscribe.Wait()
}

func createChannel(hit *it.HIT, ai int) *utils.HTTPResponse {
	query := fmt.Sprintf(`
		{
			"SubjectLogin": "paa%d",
			"ProjectionKey": [
				{
					"App":"untill/Application",
					"Projection":"paa.price",
					"WS":1},
				{
					"App":"untill/Application",
					"Projection":"paa.wine_price",
					"WS":1
				}
			]
		}`, ai)
	params := url.Values{}
	params.Add("payload", string(query))
	resp := hit.Get(fmt.Sprintf("n10n/channel?%s", params.Encode()), coreutils.WithLongPolling())
	return resp
}

func subscribe(wg *sync.WaitGroup, resp *utils.HTTPResponse) {
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(resp.HTTPResp.Body)
		scanner.Split(it.ScanSSE) // разбиваем на кадры sse, разделитель - два new line: "\n\n"
		for scanner.Scan() {
			if resp.HTTPResp.Request.Context().Err() != nil {
				return
			}
		}
	}()
}

func update(hit *it.HIT, aws int) {
	body := fmt.Sprintf(`
		{
			"App": "untill/Application",
			"Projection": "paa.price",
			"WS": %s
		}`, strconv.Itoa(aws))
	hit.Post("n10n/update/13", body)
}
