/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestJobjs_BasicUsage_Builtin(t *testing.T) {
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app2, it.ProvideApp2WithJob, it.WithUserLogin("login", "1")),
	)

	loggedJob := make(chan string)
	logger.PrintLine = func(level logger.TLogLevel, line string) {
		if strings.Contains(line, "job done") {
			loggedJob <- line
		}
	}

	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	select {
	case <-loggedJob:
	case <-time.After(5 * time.Second):
		t.Fatalf("job was not fired")
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
	loggedJobs := make(chan string, 10)
	logger.PrintLine = func(level logger.TLogLevel, line string) {
		if strings.Contains(line, "job:") {
			loggedJobs <- line
		}
	}

	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	waitForJob := func(job string) {
		select {
		case job := <-loggedJobs:
			require.True(t, strings.HasSuffix(job, job))
		case <-time.After(5 * time.Second):
			t.Fatalf("job %s was not fired", job)
		}
	}

	addMinutes := func(minutes int) {
		time.Sleep(10 * time.Millisecond) // to let scheduler to schedule the text job
		vit.TimeAdd(time.Duration(minutes) * time.Minute)
	}

	waitForJob("job:1")
	addMinutes(1)
	waitForJob("job:2")
	addMinutes(5)
	waitForJob("job:3")

}
