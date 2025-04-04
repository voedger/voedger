/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */

package sys_it

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_FederationCommand(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	it.MockQryExec = func(input string, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
		kb, err := args.State.KeyBuilder(sys.Storage_FederationCommand, appdef.NullQName)
		if err != nil {
			return err
		}
		kb.PutQName(sys.Storage_FederationCommand_Field_Command, appdef.NewQName("app1pkg", "TestCmd"))
		kb.PutString(sys.Storage_HTTP_Field_Body, `{
			"args": {
				"Arg1": 2
			}
		}`)
		v, err := args.State.MustExist(kb)
		if err != nil {
			return err
		}

		result := v.AsValue("Result")

		if result.AsInt32("Int") != 42 {
			return fmt.Errorf("unexpected result: %d", result)
		}
		return nil
	}

	t.Run("call MockQry to FederationCommand", func(t *testing.T) {
		body := `{"args": {"Input": "Anything"},"elements": [{"fields": ["Res"]}]}`
		resp := vit.PostWS(ws, "q.app1pkg.MockQry", body)
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Equal(0, resp.SysError.HTTPStatus)
		resp.Println()
	})

}
