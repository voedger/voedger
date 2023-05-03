/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"testing"
	"time"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestRates_BasicUsage(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.DummyWS(istructs.AppQName_test1_app1, 42+istructs.MaxPseudoBaseWSID)
	bodyQry := `{"args":{}, "elements":[{"path":"","fields":["Fld"]}]}`
	bodyCmd := `{"args":{}}`

	// first 2 calls are ok
	for i := 0; i < 2; i++ {
		hit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}

	// 3rd is failed because per-minute rate is exceeded
	hit.PostWS(ws, "q.sys.RatedQry", bodyQry, utils.Expect429())
	hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, utils.Expect429())

	// proceed to the next minute to restore per-minute rates
	hit.TimeAdd(time.Minute)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		hit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}

	// next are failed again because per-minute rate is exceeded again
	hit.PostWS(ws, "q.sys.RatedQry", bodyQry, utils.Expect429())
	hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, utils.Expect429())

	// proceed to the next minute to restore per-minute rates
	hit.TimeAdd(time.Minute)

	// next are failed again because per-hour rate is exceeded
	hit.PostWS(ws, "q.sys.RatedQry", bodyQry, utils.Expect429())
	hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, utils.Expect429())

	// proceed to the next hour to restore per-hour rates
	hit.TimeAdd(time.Hour)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		hit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		hit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}
}
