/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestJobjs_BasicUsage_Builtin(t *testing.T) {
	// job will run because time is increased by 1 day per each NewVIT
	// case:
	//   VVM is launched, timer for Job1_builtin is charged to MockTime.Now()+1minute
	//   MockTime.Now+1day is made on NewVIT()
	//   timer for Job1_builtin is fired
	cfg := it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app2, it.ProvideApp2WithJob, it.WithUserLogin("login", "1")),
	)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// note: use vit.TimeAdd(appropriate duration) to force timer fire for the job

	// observe "Job1_builtin works!!!!!!!!!!!!!!" in console output
}

func TestJobs_BasicUsage_Sidecar(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	cfg := it.NewOwnVITConfig(
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// configure VVM to read sidecar apps from /testdata
			cfg.DataPath = filepath.Join(wd, "testdata")
		}),
	)
	// job will run because time is increased by 1 day per each NewVIT
	// case:
	//   VVM is launched, timer for Job1_sidecar is charged to MockTime.Now()+1minute
	//   MockTime.Now+1day is made on NewVIT()
	//   timer for Job1_sidecar is fired
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// note: use vit.TimeAdd(appropriate duration) to force timer fire for the job
	Set log level

	time.Sleep(time.Second)

	// observe "panic: Job1_sidecar works!!!!!!!!!!" in console output
}
