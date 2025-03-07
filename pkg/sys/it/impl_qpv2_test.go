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
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	it "github.com/voedger/voedger/pkg/vit"
)

func prepareDailyIdx(require *require.Assertions, vit *it.VIT, ws *it.AppWorkspace) (resultOffset istructs.Offset) {
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
	return resultOffsetOfCUD
}

func TestQueryProcessor2_Views(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	expectedOffset := prepareDailyIdx(require, vit, ws)
	t.Run("Read by PK with eq", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK with in", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2024,2025]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2024-01-01","Year":2024,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK and CC with eq", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025,"Month":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK and CC with in", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2022,2023]},"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":3,"Month":2,"StringValue":"2022-02-03","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":2,"StringValue":"2022-02-05","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":4,"StringValue":"2022-04-03","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":4,"StringValue":"2022-04-05","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read without order", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read with order desc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=-Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read with order asc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Use keys constraint", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&keys=Year,Month,Day`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
			{"Day":2,"Month":1,"Year":2021},
			{"Day":2,"Month":1,"Year":2022},
			{"Day":2,"Month":1,"Year":2023}
		]}`, resp.Body)
	})
	t.Run("ACL test", func(t *testing.T) {
		newLoginName := vit.NextName()
		newLogin := vit.SignUp(newLoginName, "1", istructs.AppQName_test1_app1)
		newLoginPrn := vit.SignIn(newLogin)

		as, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
		apiToken, err := iauthnzimpl.IssueAPIToken(as.AppTokens(), time.Hour, []appdef.QName{appdef.NewQName("app1pkg", "LimitedAccessRole")},
			ws.WSID, payloads.PrincipalPayload{
				Login:       newLogin.Name,
				SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: newLoginPrn.ProfileWSID,
			})
		require.NoError(err)

		// LimitedAccessRole has no access to read all the fields
		vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx),
			coreutils.WithAuthorizeBy(apiToken), coreutils.Expect403())

		// LimitedAccessRole has access to read only Year, Month, Day and offs fields
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025}&keys=Year,Month,Day,offs`, ws.WSID, it.QNameApp1_ViewDailyIdx),
			coreutils.WithAuthorizeBy(apiToken))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
				{"Day":1,"Month":1,"Year":2025,"offs":%[1]d},
				{"Day":1,"Month":2,"Year":2025,"offs":%[1]d}
			]}`, expectedOffset), resp.Body)

		// LimitedAccessRole has no access to read CategoryIdx
		vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewCategoryIdx),
			coreutils.WithAuthorizeBy(apiToken), coreutils.Expect403())
	})
}

func TestQueryProcessor2_Queries(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	prepareDailyIdx(require, vit, ws)

	t.Run("Echo function", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/sys.Echo?arg=%s`, ws.WSID, url.QueryEscape(`{"args":{"Text":"Hello world"}}`)))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Res":"Hello world","sys.Container":"Hello world","sys.QName":"sys.EchoResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with arg", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?arg={"args":{"Year":2023,"Month":3}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Day":2,"Month":3,"StringValue":"2023-03-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":3,"StringValue":"2023-03-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":3,"StringValue":"2023-03-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with arg and filter", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&where={"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with order desc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&order=-Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
	t.Run("QryDailyIdx with order asc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&order=Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
	t.Run("QryDailyIdx with keys", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?arg={"args":{"Year":2023}}&keys=Year,Month,Day`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
}
func TestQueryProcessor2_IncludeView(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws_qp2")

	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: it.QNameApp1_ViewClients,
		WS:         ws.WSID,
	})

	dob := func(v string) int64 {
		r, err := time.Parse("2006-01-02T15:04:05", v)
		if err != nil {
			panic(err)
		}
		return r.UnixMilli()
	}

	cuds := coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(1),
					appdef.SystemField_QName: it.QNameApp1_CDocCurrency,
					it.Field_CharCode:        "EUR",
					it.Field_Code:            978,
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(2),
					appdef.SystemField_QName: it.QNameApp1_CDocCurrency,
					it.Field_CharCode:        "GBP",
					it.Field_Code:            826,
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(3),
					appdef.SystemField_QName: it.QNameApp1_CDocCountry,
					it.Field_Name:            "Spain",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(4),
					appdef.SystemField_QName: it.QNameApp1_CDocCountry,
					it.Field_Name:            "United Kingdom",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(100),
					appdef.SystemField_QName: it.QNameApp1_WDocClient,
					it.Field_FirstName:       "Juan",
					it.Field_LastName:        "Carlos",
					it.Field_DOB:             dob("1988-01-03T12:00:00"),
					it.Field_Wallet:          istructs.RecordID(101),
					it.Field_Country:         istructs.RecordID(3),
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(101),
					appdef.SystemField_QName: it.QNameApp1_WDocWallet,
					it.Field_Balance:         1000,
					it.Field_Currency:        istructs.RecordID(1),
					it.Field_Capabilities:    istructs.RecordID(102),
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(102),
					appdef.SystemField_QName: it.QNameApp1_WDocCapabilities,
					it.Field_Deposit:         true,
					it.Field_Withdraw:        true,
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(200),
					appdef.SystemField_QName: it.QNameApp1_WDocClient,
					it.Field_FirstName:       "John",
					it.Field_LastName:        "Deer",
					it.Field_DOB:             dob("2000-07-10T15:00:00"),
					it.Field_Wallet:          istructs.RecordID(201),
					it.Field_Country:         istructs.RecordID(4),
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(201),
					appdef.SystemField_QName: it.QNameApp1_WDocWallet,
					it.Field_Balance:         2000,
					it.Field_Currency:        istructs.RecordID(2),
				},
			},
		},
	}
	offset := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON()).CurrentWLogOffset
	require.EqualValues(offset, <-offsetsChan)
	unsubscribe()

	t.Run("Read by PK and include all", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/views/%s?where={"Year":{"$in":[1988]},"Month":{"$in":[1]}}&include=Client.Wallet.Currency,Client.Country,Client.Wallet.Capabilities`, ws.WSID, it.QNameApp1_ViewClients), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[{
					"Client":{
						"Country":{
							"Name":"Spain",
							"sys.ID":322685000131082,
							"sys.IsActive":true,
							"sys.QName":"app1pkg.Country"
						},
						"DOB":568209600000,
						"FirstName":"Juan",
						"LastName":"Carlos",
						"Wallet":{
							"Balance":1000,
							"Capabilities":{
								"Deposit":true,
								"Withdraw":true,
								"sys.ID":322680000131078,
								"sys.IsActive":true,
								"sys.QName":"app1pkg.Capabilities"
							},
							"Currency":{
								"CharCode":"EUR",
								"Code":978,
								"sys.ID":322685000131080,
								"sys.IsActive":true,
								"sys.QName":"app1pkg.Currency"
							},
							"sys.ID":322680000131077,
							"sys.IsActive":true,
							"sys.QName":"app1pkg.Wallet"
						},
						"sys.ID":322680000131076,
						"sys.IsActive":true,
						"sys.QName":"app1pkg.Client"
					},
					"Day":3,
					"Month":1,
					"Year":1988,
					"offs":13,
					"sys.QName":"app1pkg.Clients"
		}]}`, resp.Body)
	})
}
