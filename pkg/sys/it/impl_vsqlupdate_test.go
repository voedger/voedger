/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps/sys/clusterapp"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestVSqlUpdate_BasicUsage_Simple(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	categoryName := vit.NextName()
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
	categoryID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	newName := vit.NextName()
	body = fmt.Sprintf(`{"args": {"Query":"update test1.app1.%d.app1pkg.category.%d set name = '%s'"}}`, ws.WSID, categoryID, newName)
	vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
		coreutils.WithAuthorizeBy(sysPrn.Token)).Println()

	// check the value is update in another app and another wsid
	body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.category where id = %d"},"elements":[{"fields":["Result"]}]}`, categoryID)
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	resStr := resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)
	require.Contains(t, resStr, fmt.Sprintf(`"name":"%s"`, newName))
}

func TestVSqlUpdate_BasicUsage_Corrupted_PLog(t *testing.T) {

}

func TestVSqlUpdate_BasicUsage_Corrupted_WLog(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	categoryName := vit.NextName()
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
	resp := vit.PostWS(ws, "c.sys.CUD", body)
	wlogOffset := resp.CurrentWLogOffset
	vit.PostWS(ws, "c.sys.CUD", body)

	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	t.Run("wlog", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"update corrupted test1.app1.%d.sys.WLog.%d"}}`, ws.WSID, wlogOffset)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token))

		body = fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, wlogOffset)
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		resp.Println()
		checkCorruptedEvent(require, resp)
	})

	t.Run("plog", func(t *testing.T) {

		// determine the last PLogOffset in the target workspace
		body := `{"args":{"Query":"select * from sys.plog limit -1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)), &m))
		lastPLogOffset := int(m["PlogOffset"].(float64))

		// determine the partitionID of the last event in the target workspace
		partitionID, err := vit.IAppPartitions.AppWorkspacePartitionID(istructs.AppQName_test1_app1, ws.WSID)
		require.NoError(err)

		// update corrupted plog
		body = fmt.Sprintf(`{"args": {"Query":"update corrupted test1.app1.%d.sys.PLog.%d"}}`, partitionID, lastPLogOffset)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token))

		// check the corrupted event
		body = fmt.Sprintf(`{"args":{"Query":"select * from sys.plog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, lastPLogOffset)
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		resp.Println()
		checkCorruptedEvent(require, resp)
	})
}

func checkCorruptedEvent(require *require.Assertions, resp *coreutils.FuncResponse) {
	res := resp.SectionRow()[0].(string)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(res), &m))
	require.Empty(m["ArgumentObject"].(map[string]interface{}))
	require.Empty(m["CUDs"].([]interface{}))
	require.Zero(m["DeviceID"].(float64))
	errEvent := m["Error"].(map[string]interface{})
	require.Equal(istructsmem.ErrCorruptedData.Error(), errEvent["ErrStr"].(string))
	require.NotEmpty(errEvent["OriginalEventBytes"].(string)) // base64 here
	require.Equal(istructs.QNameForCorruptedData.String(), errEvent["QNameFromParams"].(string))
	require.False(errEvent["ValidEvent"].(bool))
}

func TestVSqlUpdate_BasicUsage_DirectUpdate_View(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	// insert a cdoc
	// p.ap1pkg.ApplyCategoryIdx will insert the single hardcoded record view.CategoryIdx(Name = category.Name, IntFld = 43, Dummy = 1, Val = 42) (see shared_cfgs.go)
	categoryName := vit.NextName()
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
	vit.PostWS(ws, "c.sys.CUD", body)

	// check view values
	body = `{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = 43 and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	res := resp.SectionRow()[0].(string)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(res), &m))
	require.Equal(categoryName, m["Name"].(string))
	require.EqualValues(42, m["Val"].(float64))

	t.Run("basic", func(t *testing.T) {
		// direct update
		newName := vit.NextName()
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.CategoryIdx set Name = '%s' where IntFld = 43 and Dummy = 1"}}`, ws.WSID, newName)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// check values are updated
		body = `{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = 43 and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		res = resp.SectionRow()[0].(string)
		m = map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(res), &m))
		require.Equal(map[string]interface{}{
			"Dummy":     float64(1),  // key (hardcoded by the projector)
			"IntFld":    float64(43), // key (hardcoded by the projector)
			"Name":      newName,     // new value
			"Val":       float64(42), // old value (hardcoded by the projector)
			"sys.QName": "app1pkg.CategoryIdx",
		}, m)
	})

	t.Run("not full key provided -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.CategoryIdx set Name = 'any' where IntFld = 43"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("Dummy", "is empty"),
		)
	})

	t.Run("update missing record -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.CategoryIdx set Name = 'any' where IntFld = 1 and Dummy = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("record cannot be found"),
		)
	})

	t.Run("update unexisting field -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.CategoryIdx set unexistingField = 'any' where IntFld = 43 and Dummy = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("unexistingField", "is not found"),
		)
	})
}

func TestVSqlUpdate_BasicUsage_DirectUpdate_Record(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// insert a doc
	categoryName := vit.NextName()
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s", "hq_id":"hq value","int_fld1":42,"int_fld2":43}}]}`, categoryName)
	categoryID := vit.PostWS(ws, "c.sys.CUD", body).NewID()
	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	t.Run("basic", func(t *testing.T) {
		// direct update
		newName := vit.NextName()
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.category.%d set name = '%s', cat_external_id = 'cat value', int_fld1 = 44"}}`, ws.WSID, categoryID, newName)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// check new state
		body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.category where sys.ID = %d"}, "elements":[{"fields":["Result"]}]}`, categoryID)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
		res := resp.SectionRow()[0].(string)
		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(res), &m))
		require.Equal(map[string]interface{}{
			"name":                      newName,     // new value
			"cat_external_id":           "cat value", // new value
			"int_fld1":                  float64(44), // new value
			"int_fld2":                  float64(43), // old value
			"hq_id":                     "hq value",  // old value
			"ml_name":                   nil,         // old value (was not set)
			appdef.SystemField_QName:    "app1pkg.category",
			appdef.SystemField_ID:       float64(categoryID),
			appdef.SystemField_IsActive: true,
		}, m)
	})

	t.Run("direct update unexisting record -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.category.%d set int_fld1 = 44"}}`, ws.WSID, istructs.NonExistingRecordID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(fmt.Sprintf("record ID %d does not exist", istructs.NonExistingRecordID)),
		)
	})

	t.Run("direct update unexisting field -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"direct update test1.app1.%d.app1pkg.category.%d set unknownField = 44"}}`, ws.WSID, categoryID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("unknownField", "is not found"),
		)
	})
}

func TestVSqlUpdate_BasicUsage_DirectInsert(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	t.Run("basic", func(t *testing.T) {
		intFld := 43 + vit.NextNumber()

		// check if there is not view record
		body := fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = %d and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`, intFld)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
		require.True(resp.IsEmpty())

		// direct insert a view record
		newName := vit.NextName()
		body = fmt.Sprintf(`{"args": {"Query":"direct insert test1.app1.%d.app1pkg.CategoryIdx set Name = '%s', Val = 123, IntFld = %d, Dummy = 1"}}`, ws.WSID, newName, intFld)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// check view values
		body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = %d and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`, intFld)
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		res := resp.SectionRow()[0].(string)
		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(res), &m))
		require.Equal(map[string]interface{}{
			"Dummy":     float64(1),
			"IntFld":    float64(intFld),
			"Name":      newName,
			"Val":       float64(123),
			"sys.QName": "app1pkg.CategoryIdx",
		}, m)
	})

	t.Run("not full key proivded -> error", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Query":"direct insert test1.app1.%d.app1pkg.CategoryIdx set Name = 'abc', Val = 123, IntFld = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("Dummy", "is empty"),
		)
	})

	t.Run("exist already by the key -> error 409 conflict", func(t *testing.T) {
		// insert new
		intFld := 43 + vit.NextNumber()
		body := fmt.Sprintf(`{"args": {"Query":"direct insert test1.app1.%d.app1pkg.CategoryIdx set Name = 'abc', Val = 123, IntFld = %d, Dummy = 1"}}`, ws.WSID, intFld)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// insert the same again -> 409 conflict
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect409("view record already exists"),
		)
	})
}

func TestVSqlUpdateErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	cases := map[string]string{
		// update
		"":             "misses required field",
		" ":            "invalid query forma",
		"update":       "invalid query format",
		"update s s s": "invalid query format",
		"update test1.app1.42.app1pkg.category.1":                                "no fields to update",
		"update 42.42.42.wongQName set name = 42":                                "invalid query format",
		"wrong op kind test1.app1.42.app1pkg.category.42 set name = 42":          "wrong update kind",
		"update test1.app1.42.app1pkg.category set name = 42":                    "record ID is not provided",
		"update test1.app1.42.app1pkg.category.1 set name = 42 where sys.ID = 1": "conditions are not allowed on update",
		"update test1.app1.42.app1pkg.category.1 set sys.ID = 1":                 "field sys.ID can not be updated",
		"update test1.app1.42.app1pkg.category.1 set sys.QName = 'sdsd.sds'":     "field sys.QName can not be updated",
		"update test1.app1.42.app1pkg.category.1 set x = 1, x = 2":               "field x specified twice",

		// update corrupted
		"update corrupted":       "invalid query format",
		"update corrupted s s s": "invalid query format",
		"update corrupted test1.app1.1.sys.PLog.1 set name = 42":             "any params of update corrupted are not allowed",
		"update corrupted test1.app1.1.sys.PLog.1 set name = 42 where x = 1": "any params of update corrupted are not allowed",
		"update corrupted test1.app1.1.sys.PLog.1 where x = 1":               "syntax error",
		"update corrupted test1.app1.0.sys.WLog.44":                          "wsid must be provided",
		"update corrupted test1.app1.1000.sys.PLog.44":                       "provided partno 1000 is out of 10 declared by app test1/app1",
		"update corrupted test1.app1.1.sys.PLog.-44":                         "invalid query format",
		"update corrupted test1.app1.1.sys.PLog.0":                           "offset must be provided",
		"update corrupted test1.app1.1.sys.PLog":                             "offset must be provided",
		"update corrupted test1.app1.1.app1pkg.category.44":                  "sys.plog or sys.wlog are only allowed",
		"update corrupted unknown.app.1.sys.PLog.44":                         "application not found: unknown/app",

		// update direct
		"direct update test1.app1.1.app1pkg.CategoryIdx set Val = 44, Name = 'x'":       "full key must be provided on view direct update",
		"direct update test1.app1.1.app1pkg.CategoryIdx where x = 1":                    "syntax error",
		"direct update test1.app1.1.app1pkg.CategoryIdx.42 set a = 2 where x = 1":       "record ID must not be provided on view direct update",
		"direct update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x = 1 or y = 1": "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"direct update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x > 1":          "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
	}
	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)
	for sql, expectedError := range cases {
		t.Run(expectedError, func(t *testing.T) {
			body := fmt.Sprintf(`{"args": {"Query":"%s"}}`, sql)
			vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
				coreutils.WithAuthorizeBy(sysPrn.Token),
				coreutils.Expect400(expectedError),
			).Println()
		})
	}
}
