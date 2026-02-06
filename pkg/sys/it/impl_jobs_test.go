/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

const testJobFireInterval = time.Minute

func TestJobs_BasicUsage_Builtin(t *testing.T) {
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app2, it.ProvideApp2WithJob, it.WithUserLogin("login", "1")),
	)

	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// the job have run here because time is increased by 1 day per each NewVIT
	// case:
	//   VVM is launched, timer for Job1_builtin is charged to MockTime.Now()+1minute (according to cron schedule)
	//   MockTime.Now+1day is made on NewVIT()
	//   timer for Job1_builtin is fired

	// will query the view from an any App Workspace
	anyAppWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID)
	sysToken := vit.GetSystemPrincipal(istructs.AppQName_test1_app2).Token

	// need to wait for the job to fire for the first time beause of day++ on NewVIT()
	require.True(t, isJobFiredForCurrentInstant_builtin(vit, anyAppWSID, sysToken, vit.Now().UnixMilli(), true))

	// proceed to the next minute second by second
	// collect instants on each second to check later that the job has not fired until the next minute
	instantsToCheck := []int64{}
	for second := vit.Now().Second(); second < 59; second++ { // 60 instead of 59 -> time++ -> current time cross the minute if current second if 59 -> fail
		vit.TimeAdd(time.Second)
		instantsToCheck = append(instantsToCheck, vit.Now().UnixMilli())
	}

	// now current second is 59
	// proceed to the next minute -> job should fire
	vit.TimeAdd(time.Second)

	// expect the job have fired and inserted the record into its view
	start := time.Now()
	fired := false
	for time.Since(start) < 3*time.Second {
		body := fmt.Sprintf(`{"args":{"Query":"select * from a1.app2pkg.Jobs where RunUnixMilli = %d"},"elements":[{"fields":["Result"]}]}`, vit.Now().UnixMilli())
		resp := vit.PostApp(istructs.AppQName_test1_app2, anyAppWSID, "q.sys.SqlQuery", body, httpu.WithAuthorizeBy(sysToken))
		if !resp.IsEmpty() {
			fired = true
			break
		}
	}
	require.True(t, fired)

	// check that there are no firings during previous seconds
	for _, currentInstant := range instantsToCheck {
		require.False(t, isJobFiredForCurrentInstant_builtin(vit, anyAppWSID, sysToken, currentInstant, false))
	}
}

func TestJobs_BasicUsage_Sidecar(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	cfg := it.NewOwnVITConfig(
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			cfg.DataPath = filepath.Join(wd, "testdata")
		}),
	)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// the job have run here because time is increased by 1 day per each NewVIT
	// case:
	//   VVM is launched, timer for Job1_builtin is charged to MockTime.Now()+1minute (according to cron schedule)
	//   MockTime.Now+1day is made on NewVIT()
	//   timer for Job1_builtin is fired

	// will query the view from an any App Workspace
	anyAppWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID)
	sysToken := vit.GetSystemPrincipal(istructs.AppQName_test2_app1).Token

	logger.Info("vit.Now() before 1st job run:", vit.Now())

	// need to wait for the job to fire for the first time beause day++ on NewVIT()
	waitForSidecarJobCounter(vit, anyAppWSID, sysToken, 1)

	// expect that the job will not fire again during the current minute
	for second := vit.Now().Second(); second < 59; second++ { // 60 instead of 59 -> time++ -> current time cross the minute if current second is 59 -> fail
		vit.TimeAdd(time.Second)
		waitForSidecarJobCounter(vit, anyAppWSID, sysToken, 1)
	}

	// now current second is 59
	// proceed to the next minute -> job should fire
	vit.TimeAdd(time.Second)

	// expect the job have fired and inserted the record into its view
	waitForSidecarJobCounter(vit, anyAppWSID, sysToken, 2)

	logger.Info("vit.Now() after 2nd job run:", vit.Now())
}

func isJobFiredForCurrentInstant_builtin(vit *it.VIT, wsid istructs.WSID, token string, instant int64, waitForFire bool) bool {
	start := time.Now()
	currentInstant := instant
	for time.Since(start) < 5*time.Second {
		body := fmt.Sprintf(`{"args":{"Query":"select * from a1.app2pkg.Jobs where RunUnixMilli = %d"},"elements":[{"fields":["Result"]}]}`, currentInstant)
		resp := vit.PostApp(istructs.AppQName_test1_app2, wsid, "q.sys.SqlQuery", body, httpu.WithAuthorizeBy(token))
		if waitForFire {
			if !resp.IsEmpty() {
				return true
			}
			time.Sleep(100 * time.Millisecond)
			vit.TimeAdd(testJobFireInterval) // force job to fire
			currentInstant += testJobFireInterval.Milliseconds()
		} else {
			return !resp.IsEmpty()
		}
	}
	return false
}

func TestJobs_SendEmail(t *testing.T) {
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app2, it.ProvideApp2WithJobSendMail),
	)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	email := vit.CaptureEmail()
	require.Equal(t, "Test Subject", email.Subject)
	require.Equal(t, "from@test.com", email.From)
	require.Equal(t, []string{"to@test.com"}, email.To)
	require.Equal(t, "Test body", email.Body)
}

func waitForSidecarJobCounter(vit *it.VIT, wsid istructs.WSID, token string, expectedMinimalCounterValue int) {
	vit.T.Helper()
	start := time.Now()
	lastValue := 0
	counter := 0
	for time.Since(start) < 5*time.Second {
		body := `{"args":{"Query":"select * from a0.sidecartestapp.JobStateView where Pk = 1 and Cc = 1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostApp(istructs.AppQName_test2_app1, wsid, "q.sys.SqlQuery", body, httpu.WithAuthorizeBy(token))
		if resp.IsEmpty() {
			if counter == 0 {
				// force job to fire only once.
				vit.TimeAdd(testJobFireInterval)
			}
			time.Sleep(100 * time.Millisecond)
			counter++
			continue
		}
		m := map[string]interface{}{}
		require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		lastValue = int(m["Counter"].(float64))
		if lastValue == expectedMinimalCounterValue {
			return
		}
	}
	vit.T.Fatal("failed to wait for sidecar job counter. Last value:", lastValue, ", expected:", expectedMinimalCounterValue)
}
