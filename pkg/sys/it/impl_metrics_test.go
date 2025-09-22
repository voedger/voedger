/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Metrics(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)

	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", vit.MetricsServicePort())
	resp, err := http.Get(url)
	require.NoError(err)

	bb, err := io.ReadAll(resp.Body)
	require.NoError(err)
	resp.Body.Close()

	require.Contains(string(bb), "{app=")
}

func TestMetricsService(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	client, cleanup := httpu.NewIHTTPClient()
	defer cleanup()

	t.Run("service check", func(t *testing.T) {
		log.Println(vit.MetricsRequest(client, httpu.WithRelativeURL("/metrics/check")))
	})

	t.Run("404 on wrong url", func(t *testing.T) {
		log.Println(vit.MetricsRequest(client, httpu.WithRelativeURL("/unknown"), httpu.Expect404()))
	})

	t.Run("404 on wrong method", func(t *testing.T) {
		log.Println(vit.MetricsRequest(client, httpu.WithMethod(http.MethodPost), httpu.Expect404()))
	})
}

func TestCommandProcessorMetrics(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	client, cleanup := httpu.NewIHTTPClient()
	defer cleanup()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	body := `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)

	metrics := vit.MetricsRequest(client)

	require.Contains(metrics, commandprocessor.CommandsTotal)
	require.Contains(metrics, commandprocessor.CommandsSeconds)
	require.Contains(metrics, commandprocessor.ExecSeconds)
	require.Contains(metrics, commandprocessor.ProjectorsSeconds)
}
