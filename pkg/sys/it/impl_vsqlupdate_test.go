/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
)

func TestVSqlUpdate_BasicUsage_UpdateTable(t *testing.T) {
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

func TestVSqlUpdate_BasicUsage_InsertTable(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	categoryName := vit.NextName()
	body := fmt.Sprintf(`{"args": {"Query":"insert test1.app1.%d.app1pkg.category set name = '%s'"}}`, ws.WSID, categoryName)
	resp := vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
		coreutils.WithAuthorizeBy(sysPrn.Token))

	newID := int64(resp.CmdResult["NewID"].(float64))
	body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.category where id = %d"},"elements":[{"fields":["Result"]}]}`, newID)
	resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
	m := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))

	require.Equal(t, map[string]interface{}{
		"cat_external_id":           "",
		"hq_id":                     "",
		"int_fld1":                  float64(0),
		"int_fld2":                  float64(0),
		"ml_name":                   nil,
		"name":                      categoryName,
		appdef.SystemField_ID:       float64(newID),
		appdef.SystemField_IsActive: true,
		appdef.SystemField_QName:    "app1pkg.category",
	}, m)
}

func TestVSqlUpdate_BasicUsage_Corrupted(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	t.Run("wlog", func(t *testing.T) {
		// make an event
		categoryName := vit.NextName()
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
		resp := vit.PostWS(ws, "c.sys.CUD", body)
		wlogOffset := resp.CurrentWLogOffset

		// read the current plog event
		_, expectedPLogEvent := getLastPLogEvent(vit, ws)

		// make wlog event corrupted
		body = fmt.Sprintf(`{"args": {"Query":"update corrupted test1.app1.%d.sys.WLog.%d"}}`, ws.WSID, wlogOffset)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token))

		// check the wlog event is corrupted indeed
		body = fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, wlogOffset)
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		resp.Println()
		checkCorruptedEvent(require, resp)

		// check the according plog event is not touched
		_, actualPLogEvent := getLastPLogEvent(vit, ws)
		require.Equal(expectedPLogEvent, actualPLogEvent)
	})

	t.Run("plog", func(t *testing.T) {
		// make an event
		categoryName := vit.NextName()
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
		resp := vit.PostWS(ws, "c.sys.CUD", body)
		wlogOffset := resp.CurrentWLogOffset

		// determine the last plog offset
		lastPLogOffset, _ := getLastPLogEvent(vit, ws)

		// get the initial wlog event
		expectedWLogEvent := getWLogEvent(vit, ws, wlogOffset)

		// determine the partitionID of the last plog event in the target workspace
		partitionID, err := vit.IAppPartitions.AppWorkspacePartitionID(istructs.AppQName_test1_app1, ws.WSID)
		require.NoError(err)

		// update corrupted plog
		body = fmt.Sprintf(`{"args": {"Query":"update corrupted test1.app1.%d.sys.PLog.%d"}}`, partitionID, lastPLogOffset)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token))

		// check the corrupted plog event
		body = fmt.Sprintf(`{"args":{"Query":"select * from sys.plog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, lastPLogOffset)
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)
		resp.Println()
		checkCorruptedEvent(require, resp)

		// check the according wlog event is not touched
		actualWLogEvent := getWLogEvent(vit, ws, wlogOffset)
		require.Equal(expectedWLogEvent, actualWLogEvent)
	})
}

func getWLogEvent(vit *it.VIT, ws *it.AppWorkspace, wlogOffset istructs.Offset) map[string]interface{} {
	vit.T.Helper()
	// determine the last PLogOffset in the target workspace
	body := fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, wlogOffset)
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	m := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	return m
}

func getLastPLogEvent(vit *it.VIT, ws *it.AppWorkspace) (plogOffset istructs.Offset, event map[string]interface{}) {
	vit.T.Helper()
	// determine the last PLogOffset in the target workspace
	body := `{"args":{"Query":"select * from sys.plog limit -1"},"elements":[{"fields":["Result"]}]}`
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

	m := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)), &m))
	return istructs.Offset(m["PlogOffset"].(float64)), m
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
	lastTest1App1Offset := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset

	// check view values
	body = `{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = 43 and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	res := resp.SectionRow()[0].(string)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(res), &m))
	require.Equal(categoryName, m["Name"].(string))
	require.EqualValues(42, m["Val"].(float64))

	t.Run("basic", func(t *testing.T) {
		// unlogged update
		newName := vit.NextName()
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.CategoryIdx set Name = '%s' where IntFld = 43 and Dummy = 1"}}`, ws.WSID, newName)
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
			"offs":      float64(lastTest1App1Offset),
		}, m)
	})

	t.Run("not full key provided -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.CategoryIdx set Name = 'any' where IntFld = 43"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("Dummy", "is empty"),
		)
	})

	t.Run("update missing record -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.CategoryIdx set Name = 'any' where IntFld = 1 and Dummy = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(fmt.Sprint(istructs.ErrRecordNotFound)), // `record not found`
		)
	})

	t.Run("update unexisting field -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.CategoryIdx set unexistingField = 'any' where IntFld = 43 and Dummy = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(istructsmem.ErrNameNotFoundError.Error(), "app1pkg.CategoryIdx", "unexistingField"),
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
		// unlogged update
		newName := vit.NextName()
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.category.%d set name = '%s', cat_external_id = 'cat value', int_fld1 = 44"}}`, ws.WSID, categoryID, newName)
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

	t.Run("unlogged update unexisting record -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.category.%d set int_fld1 = 44"}}`, ws.WSID, istructs.NonExistingRecordID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(fmt.Sprintf("record ID %d does not exist", istructs.NonExistingRecordID)),
		)
	})

	t.Run("unlogged update unexisting field -> error 400", func(t *testing.T) {
		body = fmt.Sprintf(`{"args": {"Query":"unlogged update test1.app1.%d.app1pkg.category.%d set unknownField = 44"}}`, ws.WSID, categoryID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(istructsmem.ErrNameNotFoundError.Error(), "app1pkg.category", "unknownField"),
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

		// check if there is no such view record
		bodySelect := fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.CategoryIdx where IntFld = %d and Dummy = 1"}, "elements":[{"fields":["Result"]}]}`, intFld)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", bodySelect)
		require.True(resp.IsEmpty())

		// unlogged insert a view record
		newName := vit.NextName()
		body := fmt.Sprintf(`{"args": {"Query":"unlogged insert test1.app1.%d.app1pkg.CategoryIdx set Name = '%s', Val = 123, IntFld = %d, Dummy = 1"}}`, ws.WSID, newName, intFld)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// check view values
		resp = vit.PostWS(ws, "q.sys.SqlQuery", bodySelect)
		res := resp.SectionRow()[0].(string)
		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(res), &m))
		require.Equal(map[string]interface{}{
			"Dummy":     float64(1),
			"IntFld":    float64(intFld),
			"Name":      newName,
			"Val":       float64(123),
			"offs":      float64(0),
			"sys.QName": "app1pkg.CategoryIdx",
		}, m)
	})

	t.Run("not full key proivded -> error", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Query":"unlogged insert test1.app1.%d.app1pkg.CategoryIdx set Name = 'abc', Val = 123, IntFld = 1"}}`, ws.WSID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400("Dummy", "is empty"),
		)
	})

	t.Run("exist already by the key -> error 409 conflict", func(t *testing.T) {
		// insert new
		intFld := 43 + vit.NextNumber()
		body := fmt.Sprintf(`{"args": {"Query":"unlogged insert test1.app1.%d.app1pkg.CategoryIdx set Name = 'abc', Val = 123, IntFld = %d, Dummy = 1"}}`, ws.WSID, intFld)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn.Token))

		// insert the same again -> 409 conflict
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect409("view record already exists"),
		)
	})

	t.Run("set unexisting field -> error 400", func(t *testing.T) {
		newName := vit.NextName()
		intFld := 43 + vit.NextNumber()
		body := fmt.Sprintf(`{"args": {"Query":"unlogged insert test1.app1.%d.app1pkg.CategoryIdx set Name = '%s', Val = 123, IntFld = %d, Dummy = 1, Unexisting = 42"}}`, ws.WSID, newName, intFld)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
			coreutils.WithAuthorizeBy(sysPrn.Token),
			coreutils.Expect400(istructsmem.ErrNameNotFoundError.Error(), "app1pkg.CategoryIdx", "Unexisting"),
		)
	})
}

func TestDirectUpdateManyTypes(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// create a record with fields of different types
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	_, bts := getUniqueNumber(vit)
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocManyTypes","Int":1,"Int64":2,"Float32":3.4,"Float64":5.6,"Str":"str","Bool":true,"Bytes":"%s"}}]}`, bts)
	id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	sysPrn := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)
	num := vit.NextNumber()
	buf := bytes.NewBuffer(nil)
	require.NoError(binary.Write(buf, binary.BigEndian, uint32(num)))
	newBytes := buf.Bytes()
	expectedBytesBase64 := base64.StdEncoding.EncodeToString(newBytes)

	// update the record
	// note: byte field value should be in form 0x<hex>
	body = fmt.Sprintf(`{"args": {"Query":"update test1.app1.%d.app1pkg.DocManyTypes.%d `+
		`set Int = 7, Int64 = 8, Float32 = 9.1, Float64 = 10.2, Str = 'str1', Bool = false, Bytes = 0x%x"}}`, ws.WSID, id, newBytes)
	vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body,
		coreutils.WithAuthorizeBy(sysPrn.Token)).Println()

	// check the updated record
	body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.DocManyTypes where id = %d"},"elements":[{"fields":["Result"]}]}`, id)
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	require.Equal(map[string]interface{}{
		"Bool":         false,
		"Float32":      float64(9.1),
		"Bytes":        expectedBytesBase64,
		"Float64":      float64(10.2),
		"Int":          float64(7),
		"Int64":        float64(8),
		"Str":          "str1",
		"sys.ID":       float64(id),
		"sys.IsActive": true,
		"sys.QName":    "app1pkg.DocManyTypes",
	}, m)
}

func TestUpdateDifferentLocations(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login") // from VIT shared config
	loginID := vit.GetCDocLoginID(prn.Login)

	// read and store current cdoc.Login.WSKindInitializationData
	sysPrn_Test1App1 := vit.GetSystemPrincipal(istructs.AppQName_sys_registry)
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, prn.Name, istructs.CurrentClusterID())
	queryCDocLoginBody := fmt.Sprintf(`{"args":{"Query":"select * from registry.Login.%d"},"elements":[{"fields":["Result"]}]}`, loginID)
	resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.SqlQuery", queryCDocLoginBody, coreutils.WithAuthorizeBy(sysPrn_Test1App1.Token))
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	curentWSKID := m["WSKindInitializationData"].(string)
	sysPrn_ClusterApp := vit.GetSystemPrincipal(istructs.AppQName_sys_cluster)

	rollback := func() {
		// rollback changes to keep the shared config predictable
		curentWSKIDEscaped := fmt.Sprintf("%q", curentWSKID)
		curentWSKIDEscaped = curentWSKIDEscaped[1 : len(curentWSKIDEscaped)-1] // eliminate leading and trailing double quote because the value will be specified in single qoutes
		body := fmt.Sprintf(`{"args": {"Query":"unlogged update sys.registry.\"login\".registry.Login.%d set WSKindInitializationData = '%s'"}}`, loginID, curentWSKIDEscaped)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn_ClusterApp.Token))
	}

	t.Run("hash", func(t *testing.T) {
		defer rollback()

		// unlogged update by login hash
		body := fmt.Sprintf(`{"args": {"Query":"unlogged update sys.registry.\"login\".registry.Login.%d set WSKindInitializationData = 'abc'"}}`, loginID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn_ClusterApp.Token))

		// check the result
		resp = vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.SqlQuery", queryCDocLoginBody, coreutils.WithAuthorizeBy(sysPrn_Test1App1.Token))
		m = map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal("abc", m["WSKindInitializationData"].(string))
	})

	t.Run("app workspace number", func(t *testing.T) {
		defer rollback()

		// determine the number of the app workspace that stores cdoc.Login "login"
		registryAppStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_sys_registry)
		require.NoError(err)
		appWSNumber := pseudoWSID.BaseWSID() % istructs.WSID(registryAppStructs.NumAppWorkspaces())

		// unlogged update by app workspace number
		body := fmt.Sprintf(`{"args": {"Query":"unlogged update sys.registry.a%d.registry.Login.%d set WSKindInitializationData = 'def'"}}`, appWSNumber, loginID)
		vit.PostApp(istructs.AppQName_sys_cluster, clusterapp.ClusterAppWSID, "c.cluster.VSqlUpdate", body, coreutils.WithAuthorizeBy(sysPrn_ClusterApp.Token))

		// check the result
		resp = vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.SqlQuery", queryCDocLoginBody, coreutils.WithAuthorizeBy(sysPrn_Test1App1.Token))
		m = map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal("def", m["WSKindInitializationData"].(string))
	})
}

func TestVSqlUpdateValidateErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// TODO: make test table more readable
	// cases := []struct {query string; expected: []string}…
	//
	cases := map[string]string{
		// common, update table
		"":                       "field is empty",
		" ":                      "invalid query format",
		"update":                 "invalid query format",
		"update s s s":           "invalid query format",
		"select * from sys.plog": "'update' or 'insert' clause expected",
		"wrong op kind test1.app1.42.app1pkg.category.42 set name = 42":          `invalid query format`,
		"update test1.app1.42.app1pkg.category.1":                                "no fields to update",
		"update test1.app1.app1pkg.category.1 set name = 's'":                    "location must be specified",
		"update 42.42.42.wongQName set name = 42":                                "invalid query format",
		"update test1.app1.42.app1pkg.category set name = 42":                    "record ID is not provided",
		"update test1.app1.42.app1pkg.category.1 set name = 42 where sys.ID = 1": "conditions are not allowed on update",
		"update test1.app1.42.app1pkg.category.1 set sys.ID = 1":                 "field sys.ID can not be updated",
		"update test1.app1.42.app1pkg.category.1 set sys.QName = 'sdsd.sds'":     "field sys.QName can not be updated",
		"update test1.app1.42.app1pkg.category.1 set x = 1, x = 2":               "field x specified twice",
		"update test1.app1.42.unknown.table.1 set x = 1, x = 2":                  "qname unknown.table is not found",
		"update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = 0x1":            "hex: odd length hex string",
		"update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = sin(42)":        "unsupported value type",
		"update test1.app1.42.app1pkg.MockCmd set Bytes = 0x00":                  "CDoc or WDoc only expected",
		"update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = null":           "null value is not supported",

		// insert table
		"insert test1.app1.42.app1pkg.CategoryIdx set Val = 42":        "CDoc or WDoc only expected",
		"insert test1.app1.1.app1pkg.MockCmd set Val = 44, Name = 'x'": "CDoc or WDoc only expected",
		"insert test1.app1.42.app1pkg.category":                        "no fields to set",
		"insert test1.app1.42.app1pkg.category set a = 1 where x = 1":  "conditions are not allowed on insert table",
		"insert test1.app1.42.app1pkg.category.1 set a = 1":            "record ID must not be provided on insert table",
		"insert test1.app1.42.app1pkg.category set a = null":           "null value is not supported",

		// update corrupted
		"update corrupted":       "invalid query format",
		"update corrupted s s s": "invalid query format",
		"update corrupted test1.app1.1.sys.PLog.1 set name = 42":              "any params of update corrupted are not allowed",
		"update corrupted test1.app1.1.sys.PLog.1 set name = 42 where x = 1":  "any params of update corrupted are not allowed",
		"update corrupted test1.app1.1.sys.PLog.1 where x = 1":                "syntax error",
		"update corrupted test1.app1.0.sys.WLog.44":                           "wsid must be provided",
		"update corrupted test1.app1.1000.sys.PLog.44":                        "provided partno 1000 is out of 5 declared by app test1/app1",
		"update corrupted test1.app1.1.sys.PLog.-44":                          "invalid query format",
		"update corrupted test1.app1.1.sys.PLog.0":                            "provided offset or ID must not be 0",
		"update corrupted test1.app1.1.sys.PLog":                              "offset must be provided",
		"update corrupted test1.app1.1.app1pkg.category.44":                   "sys.plog or sys.wlog are only allowed",
		"update corrupted unknown.app.1.sys.PLog.44":                          "application not found: unknown/app",
		fmt.Sprintf("update corrupted test1.app1.1.sys.PLog.%d", math.MaxInt): fmt.Sprintf("plog event partition 1 plogoffset %d does not exist", math.MaxInt),
		fmt.Sprintf("update corrupted test1.app1.1.sys.WLog.%d", math.MaxInt): fmt.Sprintf("wlog event partition 1 wlogoffset %d wsid 1 does not exist", math.MaxInt),

		// unlogged update
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = null where IntFld = 43 and Dummy = 1":   "null value is not supported",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = 44 where IntFld = 43 and Dummy = null":  "null value is not supported",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = 44, Name = 'x'":                         "full key must be provided on view unlogged update",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx where x = 1":                                      "syntax error",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx.42 set a = 2 where x = 1":                         "record ID must not be provided on view unlogged update",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x = 1 and x = 1":                  "key field x is specified twice",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = 44 where IntFld = 43 and Dummy is null": "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x = 1 or y = 1":                   "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x > 1":                            "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.category.1 set a = 2 where b = sin(x)":                        "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.category.1 set a = 2 where 1 = 1":                             "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.category.1 set a = 2 where 1 = 1 and 1 = 1":                   "'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported",
		"unlogged update test1.app1.1.app1pkg.category set a = 2":                                           "record ID must be provided on record unlogged update",
		"unlogged update test1.app1.1.app1pkg.category.1 set a = 2 where b = 3":                             "'where' clause is not allowed on record unlogged update",
		"unlogged update test1.app1.1.app1pkg.MockCmd set Val = 44, Name = 'x'":                             "view, CDoc or WDoc only expected",

		// unlogged insert
		"unlogged insert test1.app1.1.app1pkg.CategoryIdx set Val = 44, Name = 'x' where a = 1": "'where' clause is not allowed on view unlogged insert",
		"unlogged insert test1.app1.1.app1pkg.category set Val = 44, Name = 'x'":                "unlogged insert is not allowed for records", //  how to get new ID?
		"unlogged insert test1.app1.1.app1pkg.MockCmd set Val = 44, Name = 'x'":                 "view, CDoc or WDoc only expected",
		"unlogged insert test1.app1.1.app1pkg.CategoryIdx set Val = null":                       "null value is not supported",
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
