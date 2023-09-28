/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_n10n(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

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
	resp := vit.Get(fmt.Sprintf("n10n/channel?%s", params.Encode()), coreutils.WithLongPolling())

	done := make(chan interface{})
	subscribed := make(chan interface{})
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
				}
				if strings.HasPrefix(str, "data: ") {
					data = strings.TrimPrefix(str, "data: ")
				}

			}
			log.Printf("Receive event: %s, data: %s\n", event, data)
			switch event {
			case "channelId":
				channelID = data
				close(subscribed)
			case `{"App":"untill/Application","Projection":"paa.price","WS":1}`:
				require.Equal(data, "13")
				done <- nil
			}
		}
		log.Println("done")
	}()

	<-subscribed

	// вызовем тестовый метод update для обновления проекции
	body := `
 		{
 			"App": "untill/Application",
 			"Projection": "paa.price",
 			"WS": 1
 		}`
	vit.Post("n10n/update/13", body)

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
	vit.Get(fmt.Sprintf("n10n/unsubscribe?%s", params.Encode()))

	// закроем запрос, т.к. при unsubscribe завершения связи со стороны сервера не происходит
	resp.HTTPResp.Body.Close()
	<-done // подождем завершения
}
