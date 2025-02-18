/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vit"
)

func TestQueryProcessor_V2(t *testing.T) {
	require := require.New(t)
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("app1pkg", "CategoryIdx"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	// force projection update
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	resultOffsetOfCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset
	require.EqualValues(resultOffsetOfCUD, <-offsetsChan)
	unsubscribe()

	t.Run("view", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/app1pkg.CategoryIdx?where={"IntFld":43}`, ws.WSID))
		require.NoError(err)
		require.JSONEq(`{"results":[{"Dummy":1,"IntFld":43,"Name":"Awesome food","Val":42,"offs":15,"sys.QName":"app1pkg.CategoryIdx"}]}`, resp.Body)
	})

	/* TODO: next
	t.Run("query", func(t *testing.T) {
		args := `{"args": {"Text": "world"},"elements":[{"fields":["Res"]}]}`
		args = url.QueryEscape(args)
		url := fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/sys.Echo?arg=%s`, ws.WSID, args)
		_, err := vit.IFederation.Query(url)
		require.NoError(err)
	})
	*/
}
