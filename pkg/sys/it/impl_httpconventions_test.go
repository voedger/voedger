/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_HTTPConventions(t *testing.T) {
	require := require.New(t)
	vit.MockQryExec = func(input string, callback istructs.ExecQueryCallback) error {
		rr := &rr{res: input}
		require.Nil(callback(rr))
		return errors.New("test error")
	}
	vit.MockCmdExec = func(input string) error {
		return errors.New("test error")
	}
	vit := vit.NewVIT(t, &vit.SharedConfig_Simple)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	t.Run("query", func(t *testing.T) {
		body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
		resp := vit.PostProfile(prn, "q.sys.MockQry", body, coreutils.ExpectSysError500())
		require.Equal("world", resp.SectionRow()[0])
		require.Equal(coreutils.ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Equal(http.StatusInternalServerError, resp.SysError.HTTPStatus)
		require.Empty(resp.SysError.Data)
		require.Empty(resp.SysError.QName)
		resp.Println()
	})

	t.Run("command", func(t *testing.T) {
		body := `{"args": {"Input": "1"}}`
		resp := vit.PostProfile(prn, "c.sys.MockCmd", body, coreutils.Expect500())
		require.Equal(coreutils.ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusInternalServerError, resp.HTTPResp.StatusCode)
		require.Equal(http.StatusInternalServerError, resp.SysError.HTTPStatus)
		require.Empty(resp.SysError.Data)
		require.Empty(resp.SysError.QName)
		resp.Println()
	})
}
