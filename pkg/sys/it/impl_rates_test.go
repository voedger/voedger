/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestRates_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	bodyQry := `{"args":{}, "elements":[{"path":"","fields":["Fld"]}]}`
	bodyCmd := `{"args":{}}`

	// first 2 calls are ok
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}

	// 3rd is failed because per-minute rate is exceeded
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}

	// next are failed again because per-minute rate is exceeded again
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next are failed again because per-hour rate is exceeded
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next hour to restore per-hour rates
	vit.TimeAdd(time.Hour)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}
}
