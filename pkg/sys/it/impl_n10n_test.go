/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	airsbp_it "github.com/untillpro/airs-bp3/packages/air/it"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_n10n(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	// создадим канал и подпишемся на изменения проекции
	query := `
		{
			"SubjectLogin": "paa",
			"ProjectionKey": [
				{
					"App":"untill/Application",
					"Projection":"paa.price",
					"WS":1
				},
				{
					"App":"untill/Application",
					"Projection":"paa.wine_price",
					"WS":1
				}
			]
		}`
	params := url.Values{}
	params.Add("payload", query)
	resp := hit.Get(fmt.Sprintf("n10n/channel?%s", params.Encode()), coreutils.WithLongPolling())

	done := make(chan interface{})
	channelID := ""
	go func() {
		defer close(done) // отсигналим, что прочитали
		scanner := bufio.NewScanner(resp.HTTPResp.Body)
		scanner.Split(it.ScanSSE) // разбиваем на кадры sse, разделитель - два new line: "\n\n"
		for scanner.Scan() {
			if resp.HTTPResp.Request.Context().Err() != nil {
				return
			}
			messages := strings.Split(scanner.Text(), "\n") // делим кадр на событие и данные
			var event, data string

			for _, str := range messages { // вычитываем
				if strings.HasPrefix(str, "event: ") {
					event = strings.TrimPrefix(str, "event: ")
					log.Println("Receive event: " + event)
				}
				if strings.HasPrefix(str, "data: ") {
					data = strings.TrimPrefix(str, "data: ")
					log.Println("Receive data: " + data)
				}
				if event == "channelId" {
					channelID = data
				}
			}
			if strings.Compare(event, "{\"App\":\"untill/Application\",\"Projection\":\"paa.price\",\"WS\":1}") == 0 {
				require.Equal(data, "13")
				done <- nil
			}
		}
		log.Println("done")
	}()

	// вызовем тестовый метод update для обновления проекции
	body := `
 		{
 			"App": "untill/Application",
 			"Projection": "paa.price",
 			"WS": 1
 		}`
	hit.Post("n10n/update/13", body)

	<-done // подождем чтения

	// отпишемся
	query = fmt.Sprintf(`
		{
			"Channel": "%s",
			"ProjectionKey":[
				{
					"App": "untill/Application",
					"Projection":"paa.price",
					"WS":1
				}
			]
		}
	`, channelID)
	params = url.Values{}
	params.Add("payload", string(query))
	hit.Get(fmt.Sprintf("n10n/unsubscribe?%s", params.Encode()))

	// закроем запрос, т.к. при unsubscribe завершения связи со стороны сервера не происходит
	resp.HTTPResp.Body.Close()
	<-done // подождем завершения
}
