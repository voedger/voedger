/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_HTTPConventions(t *testing.T) {
	require := require.New(t)
	vit.MockQryExec = func(input string, _ istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
		rr := &rr{res: input}
		require.NoError(callback(rr))
		return errors.New("test error")
	}
	vit.MockCmdExec = func(input string, args istructs.ExecCommandArgs) error {
		return errors.New("test error")
	}
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("query", func(t *testing.T) {
		body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
		resp := vit.PostWS(ws, "q.app1pkg.MockQry", body, coreutils.ExpectSysError500())
		require.Equal("world", resp.SectionRow()[0])
		require.Equal(coreutils.ContentType_ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Equal(http.StatusInternalServerError, resp.SysError.HTTPStatus)
		require.Empty(resp.SysError.Data)
		require.Empty(resp.SysError.QName)
		resp.Println()
	})

	t.Run("command", func(t *testing.T) {
		body := `{"args": {"Input": "1"}}`
		resp := vit.PostWS(ws, "c.app1pkg.MockCmd", body, coreutils.Expect500())
		require.Equal(coreutils.ContentType_ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusInternalServerError, resp.HTTPResp.StatusCode)
		require.Equal(http.StatusInternalServerError, resp.SysError.HTTPStatus)
		require.Empty(resp.SysError.Data)
		require.Empty(resp.SysError.QName)
		resp.Println()
	})
}
