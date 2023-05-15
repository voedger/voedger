/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestRates_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.DummyWS(istructs.AppQName_test1_app1, 42+istructs.MaxPseudoBaseWSID)
	bodyQry := `{"args":{}, "elements":[{"path":"","fields":["Fld"]}]}`
	bodyCmd := `{"args":{}}`

	// first 2 calls are ok
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}

	// 3rd is failed because per-minute rate is exceeded
	vit.PostWS(ws, "q.sys.RatedQry", bodyQry, coreutils.Expect429())
	vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, coreutils.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}

	// next are failed again because per-minute rate is exceeded again
	vit.PostWS(ws, "q.sys.RatedQry", bodyQry, coreutils.Expect429())
	vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, coreutils.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next are failed again because per-hour rate is exceeded
	vit.PostWS(ws, "q.sys.RatedQry", bodyQry, coreutils.Expect429())
	vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd, coreutils.Expect429())

	// proceed to the next hour to restore per-hour rates
	vit.TimeAdd(time.Hour)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.sys.RatedQry", bodyQry)
		vit.PostWS(ws, "c.sys.RatedCmd", bodyCmd)
	}
}
