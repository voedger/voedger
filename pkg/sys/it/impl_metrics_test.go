/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	airsbp_it "github.com/untillpro/airs-bp3/packages/air/it"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	commandprocessor "github.com/voedger/voedger/pkg/processors/command"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Metrics(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	require := require.New(t)

	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", hit.MetricsServicePort())
	resp, err := http.Get(url)
	require.Nil(err, err)

	bb, err := io.ReadAll(resp.Body)
	require.NoError(err)
	resp.Body.Close()

	require.Contains(string(bb), "{app=")
}

func TestMetricsService(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	t.Run("service check", func(t *testing.T) {
		log.Println(hit.MetricsRequest(coreutils.WithRelativeURL("/metrics/check")))
	})

	t.Run("404 on wrong url", func(t *testing.T) {
		log.Println(hit.MetricsRequest(coreutils.WithRelativeURL("/unknown"), utils.Expect404()))
	})

	t.Run("404 on wrong method", func(t *testing.T) {
		log.Println(hit.MetricsRequest(coreutils.WithMethod(http.MethodPost), utils.Expect404()))
	})
}

func TestCommandProcessorMetrics(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()
	require := require.New(t)

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.payments","name":"EFT","guid":"0a53b7c6-2c47-491c-ac00-307b8d5ba6f0"}}]}`
	hit.PostWS(ws, "c.sys.CUD", body)

	metrics := hit.MetricsRequest()

	require.Contains(metrics, commandprocessor.CommandsTotal)
	require.Contains(metrics, commandprocessor.CommandsSeconds)
	require.Contains(metrics, commandprocessor.ExecSeconds)
	require.Contains(metrics, commandprocessor.ProjectorsSeconds)
}
