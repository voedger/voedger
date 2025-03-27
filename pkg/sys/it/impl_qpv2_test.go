/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/processors/query2"
	it "github.com/voedger/voedger/pkg/vit"
)

func prepareDailyIdx(require *require.Assertions, vit *it.VIT, ws *it.AppWorkspace) (resultOffset istructs.Offset, newIDs map[string]istructs.RecordID) {
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
	resp := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())
	resultOffsetOfCUD := resp.CurrentWLogOffset
	require.EqualValues(resultOffsetOfCUD, <-offsetsChan)
	unsubscribe()
	return resultOffsetOfCUD, resp.NewIDs
}

func TestQueryProcessor2_Views(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	expectedOffset, _ := prepareDailyIdx(require, vit, ws)
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
func TestQueryProcessor2_Include(t *testing.T) {
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

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(1),
					appdef.SystemField_QName: it.QNameApp1_CDocCfg,
					it.Field_Name:            "CfgA",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(2),
					appdef.SystemField_QName: it.QNameApp1_CDocCfg,
					it.Field_Name:            "CfgB",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(3),
					appdef.SystemField_QName: it.QNameApp1_CDocCfg,
					it.Field_Name:            "CfgBatch",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(4),
					appdef.SystemField_QName: it.QNameApp1_CDocBatch,
					it.Field_Number:          101,
					it.Field_Cfg:             istructs.RecordID(3),
				},
			},
		},
	}
	resp := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())

	cfgAID := resp.NewIDs["1"]
	cfgBID := resp.NewIDs["2"]
	batchID := resp.NewIDs["4"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batchID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batchID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batchID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "TaskB1",
				},
			},
		},
	}
	resp = vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())

	taskA1ID := resp.NewIDs["1"]
	taskA2ID := resp.NewIDs["2"]
	taskB1ID := resp.NewIDs["3"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(4),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(5),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskA2ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(6),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  taskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskB1",
				},
			},
		},
	}
	subTaskB1TaskB1ID := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON()).NewIDs["6"]
	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA2_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
		},
	}
	subSubTaskB1SubTaskB1TaskB1ID := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON()).NewIDs["3"]
	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA1_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA2_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  subSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
		},
	}
	vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())

	t.Run("View", func(t *testing.T) {
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
	})
	t.Run("Document", func(t *testing.T) {
		t.Run("Read by ID and include all", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA.Cfg",
				"GroupB.Cfg",
				"GroupA.GroupA.Cfg",
				"GroupA.GroupB.Cfg",
				"GroupB.GroupB.Cfg",
				"GroupB.GroupB.GroupA.Cfg",
				"GroupB.GroupB.GroupB.Cfg",
				"GroupB.GroupB.GroupB.GroupB.Cfg",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 322685000131086,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 322685000131084,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131091,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 322685000131084,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131092,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													}
												],
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 322685000131085,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskB1_TaskA1",
														"sys.Container": "GroupB",
														"sys.ID": 322685000131093,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 322685000131085,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskB2_TaskA1",
														"sys.Container": "GroupB",
														"sys.ID": 322685000131094,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 322685000131084,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131095,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131089,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"GroupB": [
											{
												"Cfg": {
													"Name": "CfgB",
													"sys.ID": 322685000131085,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 322685000131085,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 322685000131084,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubSubTaskA1_SubTaskB1_TaskB1",
																"sys.Container": "GroupA",
																"sys.ID": 322685000131097,
																"sys.IsActive": true,
																"sys.ParentID": 322685000131096,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 322685000131084,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubSubTaskA2_SubTaskB1_TaskB1",
																"sys.Container": "GroupA",
																"sys.ID": 322685000131098,
																"sys.IsActive": true,
																"sys.ParentID": 322685000131096,
																"sys.QName": "app1pkg.Task"
															}
														],
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 322685000131085,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"GroupB": [
																	{
																		"Cfg": {
																			"Name": "CfgB",
																			"sys.ID": 322685000131085,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 322685000131102,
																		"sys.IsActive": true,
																		"sys.ParentID": 322685000131099,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 322685000131099,
																"sys.IsActive": true,
																"sys.ParentID": 322685000131096,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "SubTaskB1_TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 322685000131096,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131090,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskB1",
												"sys.Container": "GroupB",
												"sys.ID": 322685000131090,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 reference field", func(t *testing.T) {
			include := []string{
				"Cfg",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 322685000131086,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 container field", func(t *testing.T) {
			include := []string{
				"GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 322685000131086,
										"GroupA": [
											{
												"Cfg": 322685000131084,
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 322685000131084,
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 container field and level 0 reference field", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 322685000131086,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"GroupA": [
											{
												"Cfg": 322685000131084,
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 322685000131084,
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 reference field", func(t *testing.T) {
			include := []string{
				"GroupA.Cfg",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 322685000131086,
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 container field", func(t *testing.T) {
			include := []string{
				"GroupA.GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 322685000131086,
										"GroupA": [
											{
												"Cfg": 322685000131084,
												"GroupA": [
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131091,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131092,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 322685000131084,
												"GroupA": [
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131095,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131089,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 container field and level 1 reference field", func(t *testing.T) {
			include := []string{
				"GroupA.Cfg",
				"GroupA.GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 322685000131086,
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131091,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131092,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131088,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131088,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 322685000131084,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": 322685000131084,
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 322685000131095,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131089,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 322685000131089,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID include level 6 container field, level 5 and level 6 are empty", func(t *testing.T) {
			include := []string{
				"GroupB.GroupB.GroupB.GroupB.GroupB.GroupB.GroupB",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 322685000131086,
										"GroupB": [
											{
												"Cfg": 322685000131085,
												"GroupB": [
													{
														"Cfg": 322685000131085,
														"GroupB": [
															{
																"Cfg": 322685000131085,
																"GroupB": [
																	{
																		"Cfg": 322685000131085,
																		"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 322685000131102,
																		"sys.IsActive": true,
																		"sys.ParentID": 322685000131099,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 322685000131099,
																"sys.IsActive": true,
																"sys.ParentID": 322685000131096,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "SubTaskB1_TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 322685000131096,
														"sys.IsActive": true,
														"sys.ParentID": 322685000131090,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskB1",
												"sys.Container": "GroupB",
												"sys.ID": 322685000131090,
												"sys.IsActive": true,
												"sys.ParentID": 322685000131087,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 322685000131087,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 0", func(t *testing.T) {
			include := []string{
				"Level0",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'Level0', 'Level0' - unexpected field"}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 1", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
				"GroupA.Level1",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'GroupA.Level1', 'Level1' - unexpected field"}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 2", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
				"GroupA.Cfg",
				"GroupB",
				"GroupB.Cfg",
				"GroupA.GroupA",
				"GroupA.GroupB",
				"GroupB.GroupB",
				"GroupA.GroupA.Level2",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'GroupA.GroupA.Level2', 'Level2' - unexpected field"}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 3", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
				"GroupA.Cfg",
				"GroupB",
				"GroupB.Cfg",
				"GroupA.GroupA",
				"GroupA.GroupB",
				"GroupB.GroupB",
				"GroupA.GroupA.Cfg",
				"GroupA.GroupB.Cfg",
				"GroupB.GroupB.Cfg",
				"GroupB.GroupB.GroupA",
				"GroupB.GroupB.GroupB",
				"GroupB.GroupB.GroupA.Cfg",
				"GroupB.GroupB.GroupB.Level3",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'GroupB.GroupB.GroupB.Level3', 'Level3' - unexpected field"}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 4", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
				"GroupA.Cfg",
				"GroupB",
				"GroupB.Cfg",
				"GroupA.GroupA",
				"GroupA.GroupB",
				"GroupB.GroupB",
				"GroupA.GroupA.Cfg",
				"GroupA.GroupB.Cfg",
				"GroupB.GroupB.Cfg",
				"GroupB.GroupB.GroupA",
				"GroupB.GroupB.GroupB",
				"GroupB.GroupB.GroupA.Cfg",
				"GroupB.GroupB.GroupB.Cfg",
				"GroupB.GroupB.GroupB.GroupB",
				"GroupB.GroupB.GroupB.GroupB.Level4",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batchID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'GroupB.GroupB.GroupB.GroupB.Level4', 'Level4' - unexpected field"}`, resp.Body)
		})
	})
}

func TestQueryProcessor2_Schemas(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	//	fmt.Printf("Port: %d\n", vit.Port())
	t.Run("read app schema", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas`)
		require.NoError(err)
		require.Equal(`<html><head><title>App test1/app1 schema</title></head><body><h1>App test1/app1 schema</h1><ul><li><a href="/api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles">app1pkg.test_wsWS</a></li></ul></body></html>`, resp.Body)
	})
}

func TestQueryProcessor2_SchemasRoles(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	//fmt.Printf("Port: %d\n", vit.Port())

	t.Run("read app workspace roles", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles`)
		require.NoError(err)
		require.Equal(`<html><head><title>App test1/app1: workspace app1pkg.test_wsWS published roles</title></head><body><h1>App test1/app1</h1><h2>Workspace app1pkg.test_wsWS published roles</h2><ul><li><a href="/api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole">app1pkg.ApiRole</a></li></ul></body></html>`, resp.Body)
	})

	t.Run("unknown ws", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/pkg.unknown/roles`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"workspace pkg.unknown not found"}`, resp.Body)
	})
}

func TestQueryProcessor2_SchemasWorkspaceRole(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	t.Run("read app workspace role in JSON", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole`,
			coreutils.WithHeaders("Accept", "application/json"))
		require.NoError(err)
		require.True(strings.HasPrefix(resp.Body, `{`))
	})
	t.Run("read app workspace role in HTML", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole`,
			coreutils.WithHeaders("Accept", "text/html"))
		require.NoError(err)
		require.True(strings.HasPrefix(resp.Body, `<`))
	})
	t.Run("unknown role", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.UnknownRole`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"role app1pkg.UnknownRole not found in workspace app1pkg.test_wsWS"}`, resp.Body)
	})

	t.Run("unknown ws", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/users/test1/apps/app1/schemas/pkg.unknown/roles/app1pkg.ApiRole`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"workspace pkg.unknown not found"}`, resp.Body)
	})
}

func TestQueryProcessor2_Docs(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	_, ids := prepareDailyIdx(require, vit, ws)

	t.Run("read document", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, ids["1"])
		resp, err := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"name":"Awesome food", "sys.ID":%d, "sys.IsActive":true, "sys.QName":"app1pkg.category"}`, ids["1"]), resp.Body)
	})

	t.Run("400 document type not defined", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameODoc1, 123)
		resp, _ := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())
		require.JSONEq(`{"status":400,"message":"document or record app1pkg.odoc1 is not defined in Workspace «app1pkg.test_wsWS»"}`, resp.Body)
	})

	t.Run("403 not authorized", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, ids["1"])
		vit.IFederation.Query(path, coreutils.Expect403())
	})

	t.Run("404 not found", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/users/test1/apps/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, 123)
		resp, _ := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect404())
		require.JSONEq(`{"status":404,"message":"document app1pkg.category with ID 123 not found"}`, resp.Body)
	})
}

func TestOpenAPI(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	appDef, err := vit.AppDef(istructs.AppQName_test1_app1)
	ws := appDef.Workspace(appdef.NewQName("app1pkg", "test_wsWS"))
	require.NotNil(ws)
	require.NoError(err)

	writer := new(bytes.Buffer)

	schemaMeta := query2.SchemaMeta{
		SchemaTitle:   "Test Schema",
		SchemaVersion: "1.0.0",
		AppName:       appdef.NewAppQName("voedger", "testapp"),
	}

	err = query2.CreateOpenApiSchema(writer, ws, appdef.NewQName("app1pkg", "ApiRole"), acl.PublishedTypes, schemaMeta)

	require.NoError(err)

	json := writer.String()
	require.Contains(json, "\"components\": {")
	require.Contains(json, "\"app1pkg.Currency\": {")
	require.Contains(json, "\"paths\": {")
	require.Contains(json, "/users/voedger/apps/testapp/workspaces/{wsid}/docs/app1pkg.Currency")
}
