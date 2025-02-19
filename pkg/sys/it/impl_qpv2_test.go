/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestQueryProcessor_V2(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: it.QNameApp1_ViewCategoryIdx,
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	newDaily := func(id int, year int32, month time.Month, day int32) coreutils.CUD {
		return coreutils.CUD{
			Fields: map[string]interface{}{
				appdef.SystemField_ID:    istructs.RecordID(id),
				appdef.SystemField_QName: it.QNameApp1_CDocDaily,
				it.Field_Year:            year,
				it.Field_Month:           month,
				it.Field_Day:             day,
				it.Field_StringValue:     fmt.Sprintf("%04d-%02d-%02d", year, month, day),
			},
		}
	}

	cuds := coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(1),
					appdef.SystemField_QName: it.QNameApp1_CDocCategory,
					"name":                   "Awesome food",
				},
			},
			newDaily(2, 2025, time.January, 1),
			newDaily(3, 2025, time.February, 1),
			newDaily(4, 2024, time.January, 1),
			newDaily(5, 2022, time.January, 2),
			newDaily(6, 2022, time.January, 3),
			newDaily(7, 2022, time.January, 4),
			newDaily(8, 2022, time.January, 5),
			newDaily(9, 2022, time.February, 2),
			newDaily(10, 2022, time.February, 3),
			newDaily(11, 2022, time.February, 4),
			newDaily(12, 2022, time.February, 5),
			newDaily(13, 2022, time.March, 2),
			newDaily(14, 2022, time.March, 3),
			newDaily(15, 2022, time.March, 4),
			newDaily(16, 2022, time.March, 5),
			newDaily(17, 2022, time.April, 2),
			newDaily(18, 2022, time.April, 3),
			newDaily(19, 2022, time.April, 4),
			newDaily(20, 2022, time.April, 5),
			newDaily(21, 2023, time.January, 2),
			newDaily(22, 2023, time.January, 3),
			newDaily(23, 2023, time.January, 4),
			newDaily(24, 2023, time.January, 5),
			newDaily(25, 2023, time.February, 2),
			newDaily(26, 2023, time.February, 3),
			newDaily(27, 2023, time.February, 4),
			newDaily(28, 2023, time.February, 5),
			newDaily(29, 2023, time.March, 2),
			newDaily(30, 2023, time.March, 3),
			newDaily(31, 2023, time.March, 4),
			newDaily(32, 2023, time.March, 5),
			newDaily(33, 2023, time.April, 2),
			newDaily(34, 2023, time.April, 3),
			newDaily(35, 2023, time.April, 4),
			newDaily(36, 2023, time.April, 5),
			newDaily(37, 2021, time.March, 2),
			newDaily(38, 2021, time.March, 3),
			newDaily(39, 2021, time.March, 4),
			newDaily(40, 2021, time.March, 5),
			newDaily(41, 2021, time.April, 2),
			newDaily(42, 2021, time.April, 3),
			newDaily(43, 2021, time.April, 4),
			newDaily(44, 2021, time.April, 5),
			newDaily(45, 2021, time.January, 2),
			newDaily(46, 2021, time.January, 3),
			newDaily(47, 2021, time.January, 4),
			newDaily(48, 2021, time.January, 5),
			newDaily(49, 2021, time.February, 2),
			newDaily(50, 2021, time.February, 3),
			newDaily(51, 2021, time.February, 4),
			newDaily(52, 2021, time.February, 5),
		},
	}
	// force projection update
	resultOffsetOfCUD := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON()).CurrentWLogOffset
	require.EqualValues(resultOffsetOfCUD, <-offsetsChan)
	unsubscribe()

	t.Run("View", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/app1pkg.CategoryIdx?where={"IntFld":43}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{
			"results":[
				{"Dummy":1,"IntFld":43,"Name":"Awesome food","Val":42,"offs":15,"sys.QName":"app1pkg.CategoryIdx"}
			]}`, resp.Body)
	})
	t.Run("Read by year", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Read by year", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2025]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Read by years", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2024,2025]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2024-01-01","Year":2024,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Read by year and month", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025,"Month":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Read by years and months and days", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2022,2023]},"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":3,"Month":2,"StringValue":"2022-02-03","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":2,"StringValue":"2022-02-05","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":4,"StringValue":"2022-04-03","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":4,"StringValue":"2022-04-05","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Without order", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Order desc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=-Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Order asc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":15,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":15,"sys.QName":"app1pkg.DailyIdx"}
		]}`, resp.Body)
	})
	t.Run("Keys", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&keys=Year,Month,Day`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":2,"Month":1,"Year":2021},
			{"Day":2,"Month":1,"Year":2022},
			{"Day":2,"Month":1,"Year":2023}
		]}`, resp.Body)
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
