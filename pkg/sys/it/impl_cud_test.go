/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */
package sys_it

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_CUD(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("create", func(t *testing.T) {
		body := `
			{
				"cuds": [
					{
						"fields": {
							"sys.ID": 1,
							"sys.QName": "app1.articles",
							"name": "cola",
							"article_manual": 1,
							"article_hash": 2,
							"hideonhold": 3,
							"time_active": 4,
							"control_active": 5
						}
					}
				]
			}`
		vit.PostWS(ws, "c.sys.CUD", body).Println()
	})

	var id float64
	t.Run("read using collection", func(t *testing.T) {
		body := `
		{
			"args":{
				"Schema":"app1.articles"
			},
			"elements":[
				{
					"fields": ["name", "control_active", "sys.ID"]
				}
			],
			"orderBy":[{"field":"name"}]
		}`
		resp := vit.PostWS(ws, "q.sys.Collection", body)
		actualName := resp.SectionRow()[0].(string)
		actualControlActive := resp.SectionRow()[1].(float64)
		id = resp.SectionRow()[2].(float64)
		require.Equal("cola", actualName)
		require.Equal(float64(5), actualControlActive)
	})

	t.Run("update", func(t *testing.T) {
		body := fmt.Sprintf(`
		{
			"cuds": [
				{
					"sys.ID": %d,
					"fields": {
						"name": "cola1",
						"article_manual": 11,
						"article_hash": 21,
						"hideonhold": 31,
						"time_active": 41,
						"control_active": 51
					}
				}
			]
		}`, int64(id))
		vit.PostWS(ws, "c.sys.CUD", body)

		body = `
		{
			"args":{
				"Schema":"app1.articles"
			},
			"elements":[
				{
					"fields": ["name", "control_active", "sys.ID"]
				}
			]
		}`
		resp := vit.PostWS(ws, "q.sys.Collection", body)
		actualName := resp.SectionRow()[0].(string)
		actualControlActive := resp.SectionRow()[1].(float64)
		newID := resp.SectionRow()[2].(float64)
		require.Equal("cola1", actualName)
		require.Equal(float64(51), actualControlActive)
		require.Equal(id, newID)

		// CDoc
		body = fmt.Sprintf(`
			{
				"args":{
					"ID": %d
				},
				"elements":[
					{
						"fields": ["Result"]
					}
				]
			}`, int64(id))
		resp = vit.PostWS(ws, "q.sys.CDoc", body)
		jsonBytes := []byte(resp.SectionRow()[0].(string))
		cdoc := map[string]interface{}{}
		require.Nil(json.Unmarshal(jsonBytes, &cdoc))
		log.Println(string(jsonBytes))
		log.Println(cdoc)
	})

	t.Run("404 on update unexisting", func(t *testing.T) {
		body := `
			{
				"cuds": [
					{
						"sys.ID": 100000000001,
						"fields": {}
					}
				]
			}`
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect404())
	})
}

func TestBasicUsage_Init(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `
		{
			"cuds": [
				{
					"fields": {
						"sys.ID": 1000000002,
						"sys.QName": "app1.articles",
						"name": "cola",
						"article_manual": 11,
						"article_hash": 21,
						"hideonhold": 31,
						"time_active": 41,
						"control_active": 51
					}
				}
			]
		}`
	vit.PostWSSys(ws, "c.sys.Init", body)

	body = `
		{
			"args":{
				"Schema":"app1.articles"
			},
			"elements":[
				{
					"fields": ["name", "control_active", "sys.ID"]
				}
			],
			"orderBy":[{"field":"name"}]
		}`
	resp := vit.PostWS(ws, "q.sys.Collection", body)
	actualName := resp.SectionRow()[0].(string)
	actualControlActive := resp.SectionRow()[1].(float64)
	id := resp.SectionRow()[2].(float64)
	require.Equal("cola", actualName)
	require.Equal(float64(51), actualControlActive)
	require.Equal(float64(1000000002), id)
}

func TestBasicUsage_Singletons(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	body := `
		{
			"cuds": [
				{
					"fields": {
						"sys.ID": 1,
						"sys.QName": "app1.Config",
						"Fld1": "42"
					}
				}
			]
		}`
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
	resp := vit.PostProfile(prn, "c.sys.CUD", body)
	require.Empty(resp.NewIDs) // ничего не прошло через ID generator

	// повторное создание -> ошибка
	vit.PostProfile(prn, "c.sys.CUD", body, coreutils.Expect409()).Println()

	// запросим ID через collection
	body = `{
		"args":{ "Schema":"app1.Config" },
		"elements":[{ "fields": ["sys.ID"] }]
	}`
	resp = vit.PostProfile(prn, "q.sys.Collection", body)
	singletonID := int64(resp.SectionRow()[0].(float64))
	log.Println(singletonID)
	require.True(istructs.RecordID(singletonID) >= istructs.FirstSingletonID && istructs.RecordID(singletonID) <= istructs.MaxSingletonID)
}

func TestUnlinkReference(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `
		{
			"cuds": [
				{
					"fields": {
						"sys.ID": 1,
						"sys.QName": "app1.options"
					}
				},
				{
					"fields": {
						"sys.ID": 2,
						"sys.QName": "app1.department",
						"pc_fix_button": 1,
						"rm_fix_button": 1
					}
				},
				{
					"fields": {
						"sys.ID": 3,
						"sys.QName": "app1.department_options",
						"id_options": 1,
						"id_department": 2,
						"sys.ParentID": 2,
						"sys.Container": "department_options",
						"option_type": 1
					}
				}
			]
		}`
	resp := vit.PostWS(ws, "c.sys.CUD", body)

	// unlink department_option from options
	idDep := resp.NewIDs["2"]
	idDepOpts := resp.NewIDs["3"]
	body = fmt.Sprintf(`{"cuds": [{"sys.ID": %d, "fields": {"id_options": %d}}]}`, idDepOpts, istructs.NullRecordID)
	vit.PostWS(ws, "c.sys.CUD", body)

	// read the root department
	body = fmt.Sprintf(`{"args":{"ID": %d},"elements":[{"fields": ["Result"]}]}`, idDep)
	resp = vit.PostWS(ws, "q.sys.CDoc", body)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
	require.Zero(m["department_options"].([]interface{})[0].(map[string]interface{})["id_options"].(float64))
}

func TestRefIntegrity(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("CUDs", func(t *testing.T) {
		t.Skip("wait for https://github.com/voedger/voedger/issues/566")
		body := `{"cuds":[{"fields":{"sys.ID":2,"sys.QName":"app1.department","pc_fix_button": 1,"rm_fix_button": 1, "id_food_group": 123456}}]}`
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect400())

		body = `{"cuds":[{"fields":{"sys.ID": 2, "sys.QName":"app1.cdoc1"}}]}`
		resp := vit.PostWS(ws, "c.sys.CUD", body)
		idCdoc1 := resp.NewIDs["2"]

		body = `{"cuds":[{"fields":{"sys.ID": 2, "sys.QName":"app1.options"}}]}`
		resp = vit.PostWS(ws, "c.sys.CUD", body)
		idOption := resp.NewIDs["2"]

		body = `{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.department","pc_fix_button": 1,"rm_fix_button": 1}}]}`
		resp = vit.PostWS(ws, "c.sys.CUD", body)
		idDep := resp.NewIDs["2"]

		body = `{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2"}}]}`
		vit.PostWS(ws, "c.sys.CUD", body)

		body = `{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field1": 123456}}]}`
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect400())

		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field1": %d}}]}`, idOption)
		vit.PostWS(ws, "c.sys.CUD", body)

		body = `{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field2": 123456}}]}`
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect400())

		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field2": %d}}]}`, idOption)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect400())

		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field2": %d}}]}`, idDep)
		vit.PostWS(ws, "c.sys.CUD", body)

		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field2": %d}}]}`, idCdoc1)
		vit.PostWS(ws, "c.sys.CUD", body)

		body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 2,"sys.QName":"app1.cdoc2","field3": %d}}]}`, idOption)
		vit.PostWS(ws, "c.sys.CUD", body)
	})

	t.Run("cmd args", func(t *testing.T) {
		// InviteID arg is recordID that references an unexisting record
		body := `{"args":{"InviteID":1234567}}`
		vit.PostWS(ws, "c.sys.CancelSentInvite", body, coreutils.Expect400())
	})
}

// https://github.com/voedger/voedger/issues/54
func TestEraseString(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{"cuds":[{"sys.ID": 5000000000400,"fields":{"name":""}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)

	body = `{"args":{"Schema":"app1.air_table_plan"},"elements":[{"fields": ["name","sys.ID"]}],"filters":[{"expr":"eq","args":{"field":"sys.ID","value":5000000000400}}]}`
	resp := vit.PostWS(ws, "q.sys.Collection", body)

	require.Equal(t, "", resp.SectionRow()[0].(string))
}

func TestEraseString1(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	body := `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
	id := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	body = fmt.Sprintf(`{"cuds":[{"sys.ID": %d,"fields":{"name":""}}]}`, id)
	vit.PostWS(ws, "c.sys.CUD", body)

	body = fmt.Sprintf(`{"args":{"Schema":"app1.articles"},"elements":[{"fields": ["name","sys.ID"]}],"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, id)
	resp := vit.PostWS(ws, "q.sys.Collection", body)

	require.Equal(t, "", resp.SectionRow()[0].(string))
}
