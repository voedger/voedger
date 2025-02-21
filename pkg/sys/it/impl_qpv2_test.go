/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"net/url"
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

	newDaily := func(year int32, month time.Month, day int32) func(id int) coreutils.CUD {
		return func(id int) coreutils.CUD {
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
	}

	dd := []func(id int) coreutils.CUD{
		newDaily(2025, time.January, 1),
		newDaily(2025, time.February, 1),
		newDaily(2024, time.January, 1),
		newDaily(2022, time.January, 2),
		newDaily(2022, time.January, 3),
		newDaily(2022, time.January, 4),
		newDaily(2022, time.January, 5),
		newDaily(2022, time.February, 2),
		newDaily(2022, time.February, 3),
		newDaily(2022, time.February, 4),
		newDaily(2022, time.February, 5),
		newDaily(2022, time.March, 2),
		newDaily(2022, time.March, 3),
		newDaily(2022, time.March, 4),
		newDaily(2022, time.March, 5),
		newDaily(2022, time.April, 2),
		newDaily(2022, time.April, 3),
		newDaily(2022, time.April, 4),
		newDaily(2022, time.April, 5),
		newDaily(2023, time.January, 2),
		newDaily(2023, time.January, 3),
		newDaily(2023, time.January, 4),
		newDaily(2023, time.January, 5),
		newDaily(2023, time.February, 2),
		newDaily(2023, time.February, 3),
		newDaily(2023, time.February, 4),
		newDaily(2023, time.February, 5),
		newDaily(2023, time.March, 2),
		newDaily(2023, time.March, 3),
		newDaily(2023, time.March, 4),
		newDaily(2023, time.March, 5),
		newDaily(2023, time.April, 2),
		newDaily(2023, time.April, 3),
		newDaily(2023, time.April, 4),
		newDaily(2023, time.April, 5),
		newDaily(2021, time.March, 2),
		newDaily(2021, time.March, 3),
		newDaily(2021, time.March, 4),
		newDaily(2021, time.March, 5),
		newDaily(2021, time.April, 2),
		newDaily(2021, time.April, 3),
		newDaily(2021, time.April, 4),
		newDaily(2021, time.April, 5),
		newDaily(2021, time.January, 2),
		newDaily(2021, time.January, 3),
		newDaily(2021, time.January, 4),
		newDaily(2021, time.January, 5),
		newDaily(2021, time.February, 2),
		newDaily(2021, time.February, 3),
		newDaily(2021, time.February, 4),
		newDaily(2021, time.February, 5),
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
		},
	}
	for i := range dd {
		cuds.Values = append(cuds.Values, dd[i](i+2))
	}
	// force projection update
	resultOffsetOfCUD := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON()).CurrentWLogOffset
	require.EqualValues(resultOffsetOfCUD, <-offsetsChan)
	unsubscribe()

	t.Run("View", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
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
	})
	t.Run("Query", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/sys.Echo?arg=%s`, ws.WSID, url.QueryEscape(`{"args":{"Text":"Hello world"}}`)))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Res":"Hello world","sys.Container":"Hello world","sys.QName":"sys.EchoResult"}
			]}`, resp.Body)
		})
		t.Run("Read by year", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2025}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Read by year and month", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023,"Month":3}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":2,"Month":3,"StringValue":"2023-03-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":3,"StringValue":"2023-03-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":3,"StringValue":"2023-03-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Read by year, month and day", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023,"Month":3,"Day":4}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Read by year and filter by month and day", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&where={"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Order desc", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&order=-Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":2,"Month":4,"StringValue":"2023-04-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":4,"StringValue":"2023-04-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":3,"StringValue":"2023-03-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":3,"StringValue":"2023-03-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":3,"StringValue":"2023-03-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":2,"StringValue":"2023-02-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":2,"StringValue":"2023-02-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":1,"StringValue":"2023-01-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":1,"StringValue":"2023-01-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":1,"StringValue":"2023-01-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Order asc", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&order=Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":1,"StringValue":"2023-01-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":1,"StringValue":"2023-01-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":1,"StringValue":"2023-01-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":2,"StringValue":"2023-02-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":2,"StringValue":"2023-02-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":3,"StringValue":"2023-03-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":3,"StringValue":"2023-03-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":3,"StringValue":"2023-03-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":2,"Month":4,"StringValue":"2023-04-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":4,"StringValue":"2023-04-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
		})
		t.Run("Keys", func(t *testing.T) {
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/query/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&keys=Year,Month,Day`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[
				{"Day":2,"Month":1,"Year":2023},
				{"Day":3,"Month":1,"Year":2023},
				{"Day":4,"Month":1,"Year":2023},
				{"Day":5,"Month":1,"Year":2023},
				{"Day":2,"Month":2,"Year":2023},
				{"Day":3,"Month":2,"Year":2023},
				{"Day":4,"Month":2,"Year":2023},
				{"Day":5,"Month":2,"Year":2023},
				{"Day":2,"Month":3,"Year":2023},
				{"Day":3,"Month":3,"Year":2023},
				{"Day":4,"Month":3,"Year":2023},
				{"Day":5,"Month":3,"Year":2023},
				{"Day":2,"Month":4,"Year":2023},
				{"Day":3,"Month":4,"Year":2023},
				{"Day":4,"Month":4,"Year":2023},
				{"Day":5,"Month":4,"Year":2023}
			]}`, resp.Body)
		})
	})
}
