/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestSidecarApps_BasicUsage(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	cfg := it.NewOwnVITConfig(
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// configure VVM to read sidecar apps from /testdata
			cfg.DataPath = filepath.Join(wd, "testdata")
		}),

	)

	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	// sidecar app test2/app1 declared in testdata/apps/test2.app1 is deployed

	// create a login in the sidecar app
	login := vit.SignUp("login", "1", istructs.AppQName_test2_app1)
	prn := vit.SignIn(login)

	// create a workspace with kind defined in sidecar app test2/app1
	ws := vit.CreateWorkspace(it.WSParams{
		Name:      "test_sidecar_ws",
		Kind:      appdef.NewQName("sidecartestapp", "test2app1"),
		ClusterID: istructs.CurrentClusterID(),
	}, prn)

	t.Run("query", func(t *testing.T) {
		// call a TestEcho query defined in wasm in test2/app1 sidecar app
		body := `{"args":{"Str":"world"},"elements":[{"fields":["Res"]}]}`
		resp := vit.PostWS(ws, "q.sidecartestapp.TestEcho", body)
		resp.Println()
		require.Equal(t, "hello, world", resp.SectionRow()[0].(string))
	})

	t.Run("command", func(t *testing.T) {
		// call a TestCmdEcho command defined in wasm in test2/app1 sidecar app
		body := `{"args":{"Str":"world"}}`
		resp := vit.PostWS(ws, "c.sidecartestapp.TestCmdEcho", body)
		require.Equal(t, "hello, world", resp.CmdResult["Res"].(string))
	})

}
