/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_CommandProcessorV2(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{
		"FldRoot": 42,
		"Nested": [
			{
				"FldNested": 43,
				"Third": [
					{"Fld1": 44},
					{"Fld1": 45}
				]
			},
			{
				"FldNested": 46,
				"Third": [
					{"Fld1": 47},
					{"Fld1": 48}
				]
			}
		]
	}`

	resp, err := vit.IFederation.Func(fmt.Sprintf("api/v2/users/test1/apps/app1/workspaces/%d/docs/app1pkg.Root", ws.WSID), body,
		coreutils.WithMethod(http.MethodPost),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	require.NoError(err)
	resp.Println()
}
