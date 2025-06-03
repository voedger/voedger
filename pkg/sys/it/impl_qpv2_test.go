/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK with in", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2024,2025]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":1,"StringValue":"2024-01-01","Year":2024,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK and CC with eq", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025,"Month":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK and CC with eq", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025,"Day":1}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results": [
			{"Day":1,"Month":1,"StringValue":"2025-01-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":1,"Month":2,"StringValue":"2025-02-01","Year":2025,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read by PK and CC with in", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2022,2023]},"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read with order desc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=-Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Read with order asc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&order=Year`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
			{"Day":2,"Month":1,"StringValue":"2021-01-02","Year":2021,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2022-01-02","Year":2022,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"},
			{"Day":2,"Month":1,"StringValue":"2023-01-02","Year":2023,"offs":%[1]d,"sys.QName":"app1pkg.DailyIdx"}
		]}`, expectedOffset), resp.Body)
	})
	t.Run("Use keys constraint", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[2021,2022,2023,2024,2025]},"Month":1,"Day":2}&keys=Year,Month,Day`, ws.WSID, it.QNameApp1_ViewDailyIdx), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
		vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx),
			coreutils.WithAuthorizeBy(apiToken), coreutils.Expect403())

		// LimitedAccessRole has access to read only Year, Month, Day and offs fields
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025}&keys=Year,Month,Day,offs`, ws.WSID, it.QNameApp1_ViewDailyIdx),
			coreutils.WithAuthorizeBy(apiToken))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
				{"Day":1,"Month":1,"Year":2025,"offs":%[1]d},
				{"Day":1,"Month":2,"Year":2025,"offs":%[1]d}
			]}`, expectedOffset), resp.Body)

		// LimitedAccessRole has no access to read CategoryIdx
		vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewCategoryIdx),
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
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/sys.Echo?args=%s`, ws.WSID, url.QueryEscape(`{"Text":"Hello world"}`)))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Res":"Hello world","sys.Container":"Hello world","sys.QName":"sys.EchoResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with arg", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?args={"Year":2023,"Month":3}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Day":2,"Month":3,"StringValue":"2023-03-02","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":3,"StringValue":"2023-03-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":4,"Month":3,"StringValue":"2023-03-04","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":3,"StringValue":"2023-03-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with arg and filter", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?args={"Year":2023}&where={"Month":{"$in":[2,4]},"Day":{"$in":[3,5]}}`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(`{"results":[
				{"Day":3,"Month":2,"StringValue":"2023-02-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":2,"StringValue":"2023-02-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":3,"Month":4,"StringValue":"2023-04-03","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"},
				{"Day":5,"Month":4,"StringValue":"2023-04-05","Year":2023,"sys.Container":"","sys.QName":"app1pkg.QryDailyIdxResult"}
			]}`, resp.Body)
	})
	t.Run("QryDailyIdx with order desc", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?args={"Year":2023}&order=-Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?args={"Year":2023}&order=Month`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryDailyIdx?args={"Year":2023}&keys=Year,Month,Day`, ws.WSID), coreutils.WithAuthorizeBy(ws.Owner.Token))
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

	t.Run("void", func(t *testing.T) {
		// just expecting no errors
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/app1pkg.QryVoid`, ws.WSID),
			coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		resp.Println()
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
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(5),
					appdef.SystemField_QName: it.QNameApp1_CDocBatch,
					it.Field_Number:          102,
					it.Field_Cfg:             istructs.RecordID(3),
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:    istructs.RecordID(6),
					appdef.SystemField_QName: it.QNameApp1_CDocBatch,
					it.Field_Number:          103,
					it.Field_Cfg:             istructs.RecordID(3),
				},
			},
		},
	}
	resp := vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())
	cfgAID := resp.NewIDs["1"]
	cfgBID := resp.NewIDs["2"]
	batch101ID := resp.NewIDs["4"]
	batch102ID := resp.NewIDs["5"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(4),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(5),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(6),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "TaskB1",
				},
			},
		},
	}
	resp = vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())
	batch101TaskA1ID := resp.NewIDs["1"]
	batch101TaskA2ID := resp.NewIDs["2"]
	batch101TaskB1ID := resp.NewIDs["3"]
	batch102TaskA1ID := resp.NewIDs["4"]
	batch102TaskA2ID := resp.NewIDs["5"]
	batch102TaskB1ID := resp.NewIDs["6"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(4),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(5),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskA2ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(6),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(7),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(8),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(9),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(10),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskA1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB2_TaskA1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(11),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskA2ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubTaskA1_TaskA2",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(12),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubTaskB1_TaskB1",
				},
			},
		},
	}
	resp = vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())
	batch101SubTaskB1TaskB1ID := resp.NewIDs["6"]
	batch102SubTaskB1TaskB1ID := resp.NewIDs["12"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA2_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(4),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(5),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubTaskA2_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(6),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
		},
	}
	resp = vit.PostWS(ws, "c.sys.CUD", cuds.MustToJSON())
	batch101SubSubTaskB1SubTaskB1TaskB1ID := resp.NewIDs["3"]
	batch102SubSubTaskB1SubTaskB1TaskB1ID := resp.NewIDs["6"]

	cuds = coreutils.CUDs{
		Values: []coreutils.CUD{
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(1),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA1_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(2),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA2_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(3),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch101SubSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupB,
					it.Field_Cfg:                 cfgBID,
					it.Field_Name:                "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(4),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA1_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(5),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubSubTaskB1SubTaskB1TaskB1ID,
					appdef.SystemField_Container: it.Field_GroupA,
					it.Field_Cfg:                 cfgAID,
					it.Field_Name:                "SubSubSubTaskA2_SubSubTaskB1_SubTaskB1_TaskB1",
				},
			},
			{
				Fields: map[string]interface{}{
					appdef.SystemField_ID:        istructs.RecordID(6),
					appdef.SystemField_QName:     it.QNameApp1_CRecordTask,
					appdef.SystemField_ParentID:  batch102SubSubTaskB1SubTaskB1TaskB1ID,
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
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[1988]},"Month":{"$in":[1]}}&include=Client.Wallet.Currency,Client.Country,Client.Wallet.Capabilities`, ws.WSID, it.QNameApp1_ViewClients), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{"results":[{
					"Client":{
						"Country":{
							"Name":"Spain",
							"sys.ID":200015,
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
								"sys.ID":200019,
								"sys.IsActive":true,
								"sys.QName":"app1pkg.Capabilities"
							},
							"Currency":{
								"CharCode":"EUR",
								"Code":978,
								"sys.ID":200013,
								"sys.IsActive":true,
								"sys.QName":"app1pkg.Currency"
							},
							"sys.ID":200018,
							"sys.IsActive":true,
							"sys.QName":"app1pkg.Wallet"
						},
						"sys.ID":200017,
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
		t.Run("Expected error https://github.com/voedger/voedger/issues/3714", func(t *testing.T) {
			_, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[1988]},"Month":{"$in":[1]}}&include=EpicFail`, ws.WSID, it.QNameApp1_ViewClients),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
			)
			var e coreutils.SysError
			_ = errors.As(err, &e)
			require.Equal(http.StatusBadRequest, e.HTTPStatus)
			require.Equal("field expression - 'EpicFail', 'EpicFail' - unexpected field", e.Message)
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
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 200024,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200034,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200035,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													}
												],
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 200023,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskB1_TaskA1",
														"sys.Container": "GroupB",
														"sys.ID": 200036,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 200023,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskB2_TaskA1",
														"sys.Container": "GroupB",
														"sys.ID": 200037,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 200038,
														"sys.IsActive": true,
														"sys.ParentID": 200029,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"GroupB": [
											{
												"Cfg": {
													"Name": "CfgB",
													"sys.ID": 200023,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 200023,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubSubTaskA1_SubTaskB1_TaskB1",
																"sys.Container": "GroupA",
																"sys.ID": 200046,
																"sys.IsActive": true,
																"sys.ParentID": 200039,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubSubTaskA2_SubTaskB1_TaskB1",
																"sys.Container": "GroupA",
																"sys.ID": 200047,
																"sys.IsActive": true,
																"sys.ParentID": 200039,
																"sys.QName": "app1pkg.Task"
															}
														],
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"GroupB": [
																	{
																		"Cfg": {
																			"Name": "CfgB",
																			"sys.ID": 200023,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 200054,
																		"sys.IsActive": true,
																		"sys.ParentID": 200048,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 200048,
																"sys.IsActive": true,
																"sys.ParentID": 200039,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "SubTaskB1_TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 200039,
														"sys.IsActive": true,
														"sys.ParentID": 200030,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskB1",
												"sys.Container": "GroupB",
												"sys.ID": 200030,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 reference field", func(t *testing.T) {
			include := []string{
				"Cfg",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 200024,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 container field", func(t *testing.T) {
			include := []string{
				"GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 200024,
										"GroupA": [
											{
												"Cfg": 200022,
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 200022,
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 0 container field and level 0 reference field", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": {
											"Name": "CfgBatch",
											"sys.ID": 200024,
											"sys.IsActive": true,
											"sys.QName": "app1pkg.Cfg"
										},
										"GroupA": [
											{
												"Cfg": 200022,
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 200022,
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 reference field", func(t *testing.T) {
			include := []string{
				"GroupA.Cfg",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 200024,
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 container field", func(t *testing.T) {
			include := []string{
				"GroupA.GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 200024,
										"GroupA": [
											{
												"Cfg": 200022,
												"GroupA": [
													{
														"Cfg": 200022,
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200034,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": 200022,
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200035,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": 200022,
												"GroupA": [
													{
														"Cfg": 200022,
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 200038,
														"sys.IsActive": true,
														"sys.ParentID": 200029,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and include level 1 container field and level 1 reference field", func(t *testing.T) {
			include := []string{
				"GroupA.Cfg",
				"GroupA.GroupA",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 200024,
										"GroupA": [
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": 200022,
														"Name": "SubTaskA1_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200034,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": 200022,
														"Name": "SubTaskA2_TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200035,
														"sys.IsActive": true,
														"sys.ParentID": 200028,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA1",
												"sys.Container": "GroupA",
												"sys.ID": 200028,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											},
											{
												"Cfg": {
													"Name": "CfgA",
													"sys.ID": 200022,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": 200022,
														"Name": "SubTaskA1_TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 200038,
														"sys.IsActive": true,
														"sys.ParentID": 200029,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskA2",
												"sys.Container": "GroupA",
												"sys.ID": 200029,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID include level 6 container field, level 5 and level 6 are empty", func(t *testing.T) {
			include := []string{
				"GroupB.GroupB.GroupB.GroupB.GroupB.GroupB.GroupB",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"Cfg": 200024,
										"GroupB": [
											{
												"Cfg": 200023,
												"GroupB": [
													{
														"Cfg": 200023,
														"GroupB": [
															{
																"Cfg": 200023,
																"GroupB": [
																	{
																		"Cfg": 200023,
																		"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 200054,
																		"sys.IsActive": true,
																		"sys.ParentID": 200048,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 200048,
																"sys.IsActive": true,
																"sys.ParentID": 200039,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "SubTaskB1_TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 200039,
														"sys.IsActive": true,
														"sys.ParentID": 200030,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Name": "TaskB1",
												"sys.Container": "GroupB",
												"sys.ID": 200030,
												"sys.IsActive": true,
												"sys.ParentID": 200025,
												"sys.QName": "app1pkg.Task"
											}
										],
										"Number": 101,
										"sys.ID": 200025,
										"sys.IsActive": true,
										"sys.QName": "app1pkg.Batch"
									}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 0", func(t *testing.T) {
			include := []string{
				"Level0",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'Level0', 'Level0' - unexpected field"}`, resp.Body)
		})
		t.Run("Read by ID and get unexpected field error on level 1", func(t *testing.T) {
			include := []string{
				"Cfg",
				"GroupA",
				"GroupA.Level1",
			}
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

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
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

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
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

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
			resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%[1]d/docs/%[2]s/%[3]d?include=%[4]s`, ws.WSID, it.QNameApp1_CDocBatch, batch101ID, strings.Join(include, ",")), coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())

			require.NoError(err)
			require.JSONEq(`{"status":400,"message":"field expression - 'GroupB.GroupB.GroupB.GroupB.Level4', 'Level4' - unexpected field"}`, resp.Body)
		})
	})
	t.Run("Documents", func(t *testing.T) {
		t.Run("Read all and include all", func(t *testing.T) {
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
			path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/cdocs/%s?include=%s`, ws.WSID, it.QNameApp1_CDocBatch, strings.Join(include, ","))
			resp, err := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token))
			require.NoError(err)
			require.JSONEq(`{
										"results": [
											{
												"Cfg": {
													"Name": "CfgBatch",
													"sys.ID": 200024,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA1_TaskA1",
																"sys.Container": "GroupA",
																"sys.ID": 200034,
																"sys.IsActive": true,
																"sys.ParentID": 200028,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA2_TaskA1",
																"sys.Container": "GroupA",
																"sys.ID": 200035,
																"sys.IsActive": true,
																"sys.ParentID": 200028,
																"sys.QName": "app1pkg.Task"
															}
														],
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskB1_TaskA1",
																"sys.Container": "GroupB",
																"sys.ID": 200036,
																"sys.IsActive": true,
																"sys.ParentID": 200028,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskB2_TaskA1",
																"sys.Container": "GroupB",
																"sys.ID": 200037,
																"sys.IsActive": true,
																"sys.ParentID": 200028,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200028,
														"sys.IsActive": true,
														"sys.ParentID": 200025,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA1_TaskA2",
																"sys.Container": "GroupA",
																"sys.ID": 200038,
																"sys.IsActive": true,
																"sys.ParentID": 200029,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 200029,
														"sys.IsActive": true,
														"sys.ParentID": 200025,
														"sys.QName": "app1pkg.Task"
													}
												],
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 200023,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"GroupA": [
																	{
																		"Cfg": {
																			"Name": "CfgA",
																			"sys.ID": 200022,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubTaskA1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupA",
																		"sys.ID": 200046,
																		"sys.IsActive": true,
																		"sys.ParentID": 200039,
																		"sys.QName": "app1pkg.Task"
																	},
																	{
																		"Cfg": {
																			"Name": "CfgA",
																			"sys.ID": 200022,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubTaskA2_SubTaskB1_TaskB1",
																		"sys.Container": "GroupA",
																		"sys.ID": 200047,
																		"sys.IsActive": true,
																		"sys.ParentID": 200039,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"GroupB": [
																	{
																		"Cfg": {
																			"Name": "CfgB",
																			"sys.ID": 200023,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"GroupB": [
																			{
																				"Cfg": {
																					"Name": "CfgB",
																					"sys.ID": 200023,
																					"sys.IsActive": true,
																					"sys.QName": "app1pkg.Cfg"
																				},
																				"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																				"sys.Container": "GroupB",
																				"sys.ID": 200054,
																				"sys.IsActive": true,
																				"sys.ParentID": 200048,
																				"sys.QName": "app1pkg.Task"
																			}
																		],
																		"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 200048,
																		"sys.IsActive": true,
																		"sys.ParentID": 200039,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 200039,
																"sys.IsActive": true,
																"sys.ParentID": 200030,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 200030,
														"sys.IsActive": true,
														"sys.ParentID": 200025,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Number": 101,
												"sys.ID": 200025,
												"sys.IsActive": true,
												"sys.QName": "app1pkg.Batch"
											},
											{
												"Cfg": {
													"Name": "CfgBatch",
													"sys.ID": 200024,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"GroupA": [
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA1_TaskA1",
																"sys.Container": "GroupA",
																"sys.ID": 200040,
																"sys.IsActive": true,
																"sys.ParentID": 200031,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA2_TaskA1",
																"sys.Container": "GroupA",
																"sys.ID": 200041,
																"sys.IsActive": true,
																"sys.ParentID": 200031,
																"sys.QName": "app1pkg.Task"
															}
														],
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskB1_TaskA1",
																"sys.Container": "GroupB",
																"sys.ID": 200042,
																"sys.IsActive": true,
																"sys.ParentID": 200031,
																"sys.QName": "app1pkg.Task"
															},
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskB2_TaskA1",
																"sys.Container": "GroupB",
																"sys.ID": 200043,
																"sys.IsActive": true,
																"sys.ParentID": 200031,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskA1",
														"sys.Container": "GroupA",
														"sys.ID": 200031,
														"sys.IsActive": true,
														"sys.ParentID": 200026,
														"sys.QName": "app1pkg.Task"
													},
													{
														"Cfg": {
															"Name": "CfgA",
															"sys.ID": 200022,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupA": [
															{
																"Cfg": {
																	"Name": "CfgA",
																	"sys.ID": 200022,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"Name": "SubTaskA1_TaskA2",
																"sys.Container": "GroupA",
																"sys.ID": 200044,
																"sys.IsActive": true,
																"sys.ParentID": 200032,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskA2",
														"sys.Container": "GroupA",
														"sys.ID": 200032,
														"sys.IsActive": true,
														"sys.ParentID": 200026,
														"sys.QName": "app1pkg.Task"
													}
												],
												"GroupB": [
													{
														"Cfg": {
															"Name": "CfgB",
															"sys.ID": 200023,
															"sys.IsActive": true,
															"sys.QName": "app1pkg.Cfg"
														},
														"GroupB": [
															{
																"Cfg": {
																	"Name": "CfgB",
																	"sys.ID": 200023,
																	"sys.IsActive": true,
																	"sys.QName": "app1pkg.Cfg"
																},
																"GroupA": [
																	{
																		"Cfg": {
																			"Name": "CfgA",
																			"sys.ID": 200022,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubTaskA1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupA",
																		"sys.ID": 200049,
																		"sys.IsActive": true,
																		"sys.ParentID": 200045,
																		"sys.QName": "app1pkg.Task"
																	},
																	{
																		"Cfg": {
																			"Name": "CfgA",
																			"sys.ID": 200022,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"Name": "SubSubTaskA2_SubTaskB1_TaskB1",
																		"sys.Container": "GroupA",
																		"sys.ID": 200050,
																		"sys.IsActive": true,
																		"sys.ParentID": 200045,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"GroupB": [
																	{
																		"Cfg": {
																			"Name": "CfgB",
																			"sys.ID": 200023,
																			"sys.IsActive": true,
																			"sys.QName": "app1pkg.Cfg"
																		},
																		"GroupB": [
																			{
																				"Cfg": {
																					"Name": "CfgB",
																					"sys.ID": 200023,
																					"sys.IsActive": true,
																					"sys.QName": "app1pkg.Cfg"
																				},
																				"Name": "SubSubSubTaskB1_SubSubTaskB1_SubTaskB1_TaskB1",
																				"sys.Container": "GroupB",
																				"sys.ID": 200057,
																				"sys.IsActive": true,
																				"sys.ParentID": 200051,
																				"sys.QName": "app1pkg.Task"
																			}
																		],
																		"Name": "SubSubTaskB1_SubTaskB1_TaskB1",
																		"sys.Container": "GroupB",
																		"sys.ID": 200051,
																		"sys.IsActive": true,
																		"sys.ParentID": 200045,
																		"sys.QName": "app1pkg.Task"
																	}
																],
																"Name": "SubTaskB1_TaskB1",
																"sys.Container": "GroupB",
																"sys.ID": 200045,
																"sys.IsActive": true,
																"sys.ParentID": 200033,
																"sys.QName": "app1pkg.Task"
															}
														],
														"Name": "TaskB1",
														"sys.Container": "GroupB",
														"sys.ID": 200033,
														"sys.IsActive": true,
														"sys.ParentID": 200026,
														"sys.QName": "app1pkg.Task"
													}
												],
												"Number": 102,
												"sys.ID": 200026,
												"sys.IsActive": true,
												"sys.QName": "app1pkg.Batch"
											},
											{
												"Cfg": {
													"Name": "CfgBatch",
													"sys.ID": 200024,
													"sys.IsActive": true,
													"sys.QName": "app1pkg.Cfg"
												},
												"Number": 103,
												"sys.ID": 200027,
												"sys.IsActive": true,
												"sys.QName": "app1pkg.Batch"
											}
										]
									}`, resp.Body)
		})
	})
	t.Run("Expected error https://github.com/voedger/voedger/issues/3696", func(t *testing.T) {
		_, _ = vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":{"$in":[1988]},"Month":{"$in":[1]}}&include=Client.Country.Name`, ws.WSID, it.QNameApp1_ViewClients),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400(),
		)
	})
}

func TestQueryProcessor2_Schemas(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	//	fmt.Printf("Port: %d\n", vit.Port())

	t.Run("read app schema", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas`)
		require.NoError(err)
		require.Equal(`<html><head><title>App test1/app1 schema</title></head><body><h1>App test1/app1 schema</h1><h2>Package app1pkg</h2><ul><li><a href="/api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles">app1pkg.test_wsWS</a></li></ul></body></html>`, resp.Body)
	})

	t.Run("read app schema as a sys.Developer", func(t *testing.T) {
		// Generate sys.Developer token
		pp := payloads.PrincipalPayload{
			Login:       "Login",
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: 1,
			Roles:       []payloads.RoleType{{WSID: 1, QName: appdef.QNameRoleDeveloper}},
		}
		tokenDeveloper, err := vit.IssueToken(istructs.AppQName_test1_app1, 100*time.Minute, &pp)
		require.NoError(err)
		// fmt.Printf("Developer token: %s\n", tokenDeveloper)

		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas`, coreutils.WithAuthorizeBy(tokenDeveloper))
		require.NoError(err)
		require.Contains(resp.Body, `<h2>Package sys</h2>`)
	})
}

func TestQueryProcessor2_SchemasRoles(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	//fmt.Printf("Port: %d\n", vit.Port())

	t.Run("read app workspace roles", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles`)
		require.NoError(err)
		require.Equal(`<html><head><title>App test1/app1: workspace app1pkg.test_wsWS published roles</title></head><body><h1>App test1/app1</h1><h2>Workspace app1pkg.test_wsWS published roles</h2><h2>Package app1pkg</h2><ul><li><a href="/api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole">app1pkg.ApiRole</a></li></ul></body></html>`, resp.Body)
	})

	t.Run("read app workspace roles as a sys.Developer", func(t *testing.T) {
		// Generate sys.Developer token
		pp := payloads.PrincipalPayload{
			Login:       "Login",
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: 1,
			Roles:       []payloads.RoleType{{WSID: 1, QName: appdef.QNameRoleDeveloper}},
		}
		tokenDeveloper, err := vit.IssueToken(istructs.AppQName_test1_app1, 100*time.Minute, &pp)
		require.NoError(err)
		// fmt.Printf("Developer token: %s\n", tokenDeveloper)

		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles`, coreutils.WithAuthorizeBy(tokenDeveloper))
		require.NoError(err)
		require.Contains(resp.Body, `<h2>Package sys</h2>`)
		require.Contains(resp.Body, `sys.WorkspaceOwner`)
	})

	t.Run("read workspace with no published roles", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/sys.Workspace/roles`)
		require.NoError(err)
		require.Contains(resp.Body, `No published roles`)
	})

	t.Run("unknown ws", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/pkg.unknown/roles`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"workspace pkg.unknown not found"}`, resp.Body)
	})
}

// [~server.apiv2.role/it.TestQueryProcessor2_SchemasRole~impl]
func TestQueryProcessor2_SchemasWorkspaceRole(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	currencyPath := "/apps/test1/app1/workspaces/{wsid}/docs/app1pkg.Currency"
	initiateJoinWorkspacePath := "/apps/test1/app1/workspaces/{wsid}/commands/sys.InitiateJoinWorkspace"

	t.Run("read app workspace role in JSON", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole`,
			coreutils.WithHeaders("Accept", "application/json"))
		require.NoError(err)
		require.True(strings.HasPrefix(resp.Body, `{`))
		require.Contains(resp.Body, currencyPath)
		require.NotContains(resp.Body, initiateJoinWorkspacePath)
	})
	t.Run("read app workspace role as sys.Developer", func(t *testing.T) {
		// Generate sys.Developer token
		pp := payloads.PrincipalPayload{
			Login:       "Login",
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: 1,
			Roles:       []payloads.RoleType{{WSID: 1, QName: appdef.QNameRoleDeveloper}},
		}
		tokenDeveloper, err := vit.IssueToken(istructs.AppQName_test1_app1, 100*time.Minute, &pp)
		require.NoError(err)
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/sys.Workspace/roles/sys.AuthenticatedUser`,
			coreutils.WithHeaders("Accept", "application/json"), coreutils.WithAuthorizeBy(tokenDeveloper))
		require.NoError(err)
		require.True(strings.HasPrefix(resp.Body, `{`))
		require.NotContains(resp.Body, currencyPath)
		require.Contains(resp.Body, initiateJoinWorkspacePath)
	})
	t.Run("not allowed to read system role with no Developer token", func(t *testing.T) {
		vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/sys.Workspace/roles/sys.AuthenticatedUser`,
			coreutils.WithHeaders("Accept", "application/json"), coreutils.Expect403())
	})
	t.Run("read app workspace role in HTML", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.ApiRole`,
			coreutils.WithHeaders("Accept", "text/html"))
		require.NoError(err)
		require.True(strings.HasPrefix(resp.Body, `<`))
	})
	t.Run("unknown role", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/app1pkg.test_wsWS/roles/app1pkg.UnknownRole`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"role app1pkg.UnknownRole not found in workspace app1pkg.test_wsWS"}`, resp.Body)
	})

	t.Run("unknown ws", func(t *testing.T) {
		resp, err := vit.IFederation.Query(`api/v2/apps/test1/app1/schemas/pkg.unknown/roles/app1pkg.ApiRole`, coreutils.Expect404())
		require.NoError(err)
		require.JSONEq(`{"status":404,"message":"workspace pkg.unknown not found"}`, resp.Body)
	})
}

// [~server.apiv2.docs/it.TestQueryProcessor2_Docs~impl]
func TestQueryProcessor2_Docs(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	_, ids := prepareDailyIdx(require, vit, ws)

	t.Run("read document", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, ids["1"])
		resp, err := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"name":"Awesome food", "sys.ID":%d, "sys.IsActive":true, "sys.QName":"app1pkg.category"}`, ids["1"]), resp.Body)
	})

	t.Run("400 document type not defined", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameODoc1, 123)
		resp, _ := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect400())
		require.JSONEq(`{"status":400,"message":"document or record app1pkg.odoc1 is not defined in Workspace «app1pkg.test_wsWS»"}`, resp.Body)
	})

	t.Run("403 not authorized", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, ids["1"])
		vit.IFederation.Query(path, coreutils.Expect403())
	})

	t.Run("404 not found", func(t *testing.T) {
		path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/%s/%d`, ws.WSID, it.QNameApp1_CDocCategory, 123)
		resp, _ := vit.IFederation.Query(path, coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect404())
		require.JSONEq(`{"status":404,"message":"document app1pkg.category with ID 123 not found"}`, resp.Body)
	})
}

func TestOpenAPI(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	appDef, err := vit.AppDef(istructs.AppQName_test1_app1)
	require.NoError(err)

	schemaMeta := query2.SchemaMeta{
		SchemaTitle:   "Test Schema",
		SchemaVersion: "1.0.0",
		AppName:       appdef.NewAppQName("voedger", "testapp"),
	}

	createOpenApi := func(wsName, role appdef.QName) string {
		writer := new(bytes.Buffer)
		ws := appDef.Workspace(wsName)
		require.NotNil(ws)
		err = query2.CreateOpenAPISchema(writer, ws, role, acl.PublishedTypes, schemaMeta, false)
		require.NoError(err)
		json := writer.String()
		//save to file
		// err = os.WriteFile(fmt.Sprintf("%s.json", role.String()), []byte(json), 0644)
		// require.NoError(err)
		return json
	}

	currencyPath := "/apps/voedger/testapp/workspaces/{wsid}/docs/app1pkg.Currency"
	initiateJoinWorkspacePath := "/apps/voedger/testapp/workspaces/{wsid}/commands/sys.InitiateJoinWorkspace"
	json := createOpenApi(appdef.NewQName("app1pkg", "test_wsWS"), appdef.NewQName("app1pkg", "ApiRole"))
	require.Contains(json, "\"components\": {")
	require.Contains(json, "\"app1pkg.Currency\": {")
	require.Contains(json, "\"paths\": {")
	require.Contains(json, currencyPath)
	require.NotContains(json, initiateJoinWorkspacePath)

	json = createOpenApi(appdef.NewQName("sys", "Workspace"), appdef.NewQName("sys", "AuthenticatedUser"))
	require.Contains(json, "\"components\": {")
	require.Contains(json, initiateJoinWorkspacePath)
	require.NotContains(json, currencyPath)

}

// [~server.apiv2.docs/it.TestQueryProcessor2_CDocs~impl]
func TestQueryProcessor2_CDocs(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws3")
	_, ids := prepareDailyIdx(require, vit, ws)

	t.Run("Read documents", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/cdocs/%s`, ws.WSID, it.QNameApp1_CDocCategory), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[{"name":"Awesome food","sys.ID":%d,"sys.IsActive":true,"sys.QName":"app1pkg.category"}]}`, ids["1"]), resp.Body)
	})
	t.Run("Read documents and use keys constraint", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/cdocs/%s?keys=name,sys.ID`, ws.WSID, it.QNameApp1_CDocCategory), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[{"name":"Awesome food","sys.ID":%d}]}`, ids["1"]), resp.Body)
	})
	t.Run("Read documents and use keys, order, skip and limit constraints", func(t *testing.T) {
		resp, err := vit.IFederation.Query(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/cdocs/%s?keys=sys.ID,Year,Month,Day&order=-Month&skip=6&limit=10`, ws.WSID, it.QNameApp1_CDocDaily), coreutils.WithAuthorizeBy(ws.Owner.Token))
		require.NoError(err)
		require.JSONEq(fmt.Sprintf(`{"results":[
				{"Day":4,"Month":4,"Year":2023,"sys.ID":%[1]d},
				{"Day":3,"Month":4,"Year":2023,"sys.ID":%[2]d},
				{"Day":2,"Month":4,"Year":2023,"sys.ID":%[3]d},
				{"Day":5,"Month":4,"Year":2022,"sys.ID":%[4]d},
				{"Day":4,"Month":4,"Year":2022,"sys.ID":%[5]d},
				{"Day":3,"Month":4,"Year":2022,"sys.ID":%[6]d},
				{"Day":3,"Month":3,"Year":2021,"sys.ID":%[7]d},
				{"Day":4,"Month":3,"Year":2021,"sys.ID":%[8]d},
				{"Day":5,"Month":3,"Year":2022,"sys.ID":%[9]d},
				{"Day":3,"Month":3,"Year":2022,"sys.ID":%[10]d}
		]}`, ids["35"], ids["34"], ids["33"], ids["20"], ids["19"], ids["18"], ids["38"], ids["39"], ids["16"], ids["14"]), resp.Body)
	})
}

// [~server.authnz/it.TestLogin~impl]
func TestQueryProcessor2_AuthLogin(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	//appDef, err := vit.AppDef(istructs.AppQName_test1_app1)

	loginName1 := vit.NextName()
	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)

	vit.SignIn(login1)

	t.Run("Login", func(t *testing.T) {
		body := fmt.Sprintf(`{"login": "%s","password": "%s"}`, login1.Name, login1.Pwd)
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body)
		require.Equal(200, resp.HTTPResp.StatusCode)
		result := make(map[string]interface{})
		err := json.Unmarshal([]byte(resp.Body), &result)
		require.NoError(err)
		require.Equal(3600.0, result["expiresIn"])
		require.Greater(istructs.WSID(result["wsid"].(float64)), login1.PseudoProfileWSID)
		require.NotEmpty(result["principalToken"].(string))
	})

	t.Run("Bad request", func(t *testing.T) {
		cases := []struct {
			bodies   []string
			expected []string
		}{
			{
				bodies:   []string{"", "{}"},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`, `string-field «Password»`},
			},
			{
				bodies: []string{
					`{"password": "pwd"}`,
					fmt.Sprintf(`{"UnknownField": "%s","password": "pwd"}`, login1.Name),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`},
			},
			{
				bodies: []string{
					`{"login": "pwd"}`,
					fmt.Sprintf(`{"login": "%s","UnknownField": "pwd"}`, login1.Name),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Password»`, `validate error code 4`},
			},
			{
				bodies: []string{
					`{"login": 42}`,
				},
				expected: []string{`field \"login\" must be a string`, `field type mismatch`},
			},
			{
				bodies: []string{
					`{"password": 42}`,
				},
				expected: []string{`field \"password\" must be a string`, `field type mismatch`},
			},
			{
				bodies: []string{
					fmt.Sprintf(`{"UnknownField": "%s","password": "%s"}`, login1.Name, "badpwd"),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`},
			},
		}
		for _, c := range cases {
			for _, body := range c.bodies {
				t.Run(body, func(t *testing.T) {
					resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, coreutils.Expect400())
					require.Contains(resp.Body, `"status":400`)
					for _, expected := range c.expected {
						require.Contains(resp.Body, expected)
					}
				})
			}
		}
	})

	t.Run("Login with incorrect password", func(t *testing.T) {
		body := fmt.Sprintf(`{"login": "%s","password": "%s"}`, login1.Name, "badpwd")
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, coreutils.Expect401())
		require.JSONEq(`{"status":401,"message":"login or password is incorrect"}`, resp.Body)
	})

}

// [~server.authnz/it.TestRefresh~impl]
func TestQueryProcessor2_AuthRefresh(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)

	loginName1 := vit.NextName()
	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	prn1 := vit.SignIn(login1)

	t.Run("Refresh", func(t *testing.T) {
		// simulate delay to make the new token be different after referesh
		vit.TimeAdd(time.Minute)
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", coreutils.WithAuthorizeBy(prn1.Token))
		require.Equal(200, resp.HTTPResp.StatusCode)
		result := make(map[string]interface{})
		err := json.Unmarshal([]byte(resp.Body), &result)
		require.NoError(err)
		require.Equal(3600.0, result["expiresIn"])
		require.Equal(istructs.WSID(result["wsid"].(float64)), prn1.ProfileWSID)
		newToken := result["principalToken"].(string)
		require.NotEmpty(newToken)
		require.NotEqual(newToken, prn1.Token)
	})

	t.Run("Empty token", func(t *testing.T) {
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", coreutils.Expect401())
		require.JSONEq(`{"status":401,"message":"authorization header is empty"}`, resp.Body)
	})

	t.Run("Old token", func(t *testing.T) {
		vit.TimeAdd(time.Hour * 2)
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", coreutils.WithAuthorizeBy(prn1.Token), coreutils.Expect401())
		require.JSONEq(`{"status":401,"message":"token expired"}`, resp.Body)
	})

}
