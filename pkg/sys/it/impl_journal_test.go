/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/journal"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Journal(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	tableNum := vit.NextNumber()
	idUntillUsers := vit.GetAny("app1pkg.untill_users", ws)

	bill := fmt.Sprintf(`{
				"cuds": [{
				  "fields": {
					"sys.ID": 1,
					"sys.QName": "app1pkg.bill",
					"tableno": %d,
					"id_untill_users": %d,
					"table_part": "a",
					"proforma": 3,
					"working_day": "20230228"
				  }
				}]
			}`, tableNum, idUntillUsers)
	resp := vit.PostWS(ws, "c.sys.CUD", bill)
	ID := resp.NewID()
	expectedOffset := resp.CurrentWLogOffset

	WaitForIndexOffset(vit, ws, journal.QNameViewWLogDates, expectedOffset)

	//Read by unix timestamp
	body := fmt.Sprintf(`
	{
		"args":{"From":%d,"Till":%d,"EventTypes":"all"},
		"elements":[{"fields":["Offset","EventTime","Event"]}]
	}`, vit.Now().UnixMilli(), vit.Now().UnixMilli())
	resp = vit.PostWS(ws, "q.sys.Journal", body)

	j := resp.SectionRow()[2].(string)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(j), &m))
	jn, err := json.MarshalIndent(&m, "", "\t")
	require.NoError(err)
	log.Println(string(jn))

	expectedEvent := fmt.Sprintf(`
	{
		"DeviceID": 0,
		"RegisteredAt": %[3]d,
		"Synced": false,
		"SyncedAt": 0,
		"args": {},
		"cuds": [
			{
				"IsNew": true,
				"fields": {
					"age": 0,
					"ayce_time": 0,
					"bill_type": 0,
					"client_phone": "",
					"close_datetime": 0,
					"comments": "",
					"day_failurednumber": 0,
					"day_number": 0,
					"day_suffix": "",
					"description": null,
					"discount": 0,
					"discount_value": 0,
					"extra_fields": null,
					"failurednumber": 0,
					"fiscal_failurednumber": 0,
					"fiscal_number": 0,
					"fiscal_suffix": "",
					"free_comments": "",
					"group_vat_level": 0,
					"hc_folionumber": "",
					"hc_foliosequence": 0,
					"hc_roomnumber": "",
					"id_alter_user": 0,
					"id_bo_service_charge": 0,
					"id_callers_last": 0,
					"id_cardprice": 0,
					"id_clients": 0,
					"id_courses": 0,
					"id_discount_reasons": 0,
					"id_order_type": 0,
					"id_serving_time": 0,
					"id_t2o_groups": 0,
					"id_time_article": 0,
					"id_untill_users": %[4]d,
					"id_user_proforma": 0,
					"ignore_auto_sc": 0,
					"isactive": 0,
					"isdirty": 0,
					"locker": 0,
					"modified": 0,
					"name": "",
					"not_paid": 0,
					"number": 0,
					"number_of_covers": 0,
					"open_datetime": 0,
					"pbill_failurednumber": 0,
					"pbill_number": 0,
					"pbill_suffix": "",
					"proforma": 3,
					"qty_persons": 0,
					"remaining_quantity": 0,
					"reservationid": "",
					"sc_plan": null,
					"sdescription": "",
					"service_charge": 0,
					"service_tax": 0,
					"serving_time_dt": 0,
					"suffix": "",
					"sys.ID": %[1]d,
					"sys.IsActive": true,
					"sys.QName": "app1pkg.bill",
					"table_name": "",
					"table_part": "a",
					"tableno":  %[2]d,
					"take_away": 0,
					"timer_start": 0,
					"timer_stop": 0,
					"tip": 0,
					"total": 0,
					"vars": null,
					"vat_excluded": 0,
					"was_cancelled": 0,
					"working_day": "20230228"
				},
				"sys.ID": %[1]d,
				"sys.QName": "app1pkg.bill"
			}
		],
		"sys.QName": "sys.CUD"
	}`, ID, tableNum, vit.Now().UnixMilli(), idUntillUsers)

	require.Equal(expectedOffset, istructs.Offset(resp.SectionRow()[0].(float64)))
	require.Equal(int64(resp.SectionRow()[1].(float64)), vit.Now().UnixMilli())
	require.JSONEq(expectedEvent, resp.SectionRow()[2].(string))

	//Read by offset
	body = fmt.Sprintf(`
	{
		"args":{"From":%d,"Till":%d,"EventTypes":"all","RangeUnit":"Offset"},
		"elements":[{"fields":["Offset","EventTime","Event"]}]
	}`, expectedOffset, expectedOffset)
	resp = vit.PostWS(ws, "q.sys.Journal", body)

	require.JSONEq(fmt.Sprintf(`
	{
	  "args": {},
	  "cuds": [
		{
		  "fields": {
			"id_untill_users": %[4]d,
			"proforma": 3,
			"sys.ID": %[1]d,
			"sys.IsActive": true,
			"sys.QName": "app1pkg.bill",
			"table_part": "a",
			"tableno": %[2]d,
			"working_day": "20230228"
		  },
		  "IsNew": true,
		  "sys.ID": %[1]d,
		  "sys.QName": "app1pkg.bill"
		}
	  ],
	  "DeviceID": 0,
	  "RegisteredAt": %[3]d,
	  "Synced": false,
	  "SyncedAt": 0,
	  "sys.QName": "sys.CUD"
	}`, ID, tableNum, vit.Now().UnixMilli(), idUntillUsers), resp.SectionRow()[2].(string))

	expectedEvent = fmt.Sprintf(`
		{
			"args": {},
			"cuds": [
			{
				"fields": {
				"id_untill_users": %[4]d,
				"proforma": 3,
				"sys.ID": %[1]d,
				"sys.IsActive": true,
				"sys.QName": "app1pkg.bill",
				"table_part": "a",
				"tableno": %[2]d,
				"working_day": "20230228"
				},
				"IsNew": true,
				"sys.ID": %[1]d,
				"sys.QName": "app1pkg.bill"
			}
			],
			"DeviceID": 0,
			"RegisteredAt": %[3]d,
			"Synced": false,
			"SyncedAt": 0,
			"sys.QName": "sys.CUD"
		}`, ID, tableNum, vit.Now().UnixMilli(), idUntillUsers)

	require.Equal(expectedOffset, istructs.Offset(resp.SectionRow()[0].(float64)))
	require.Equal(int64(resp.SectionRow()[1].(float64)), vit.Now().UnixMilli())
	require.JSONEq(expectedEvent, resp.SectionRow()[2].(string))
}

func TestJournal_read_in_years_range_1(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	idUntillUsers := vit.GetAny("app1pkg.untill_users", ws)

	createBill := func(tableNo int) istructs.Offset {
		bill := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.bill","tableno":%d,"id_untill_users":%d,"table_part":"a","proforma":3,"working_day":"20230227"}}]}`, tableNo, idUntillUsers)
		return vit.PostWS(ws, "c.sys.CUD", bill).CurrentWLogOffset
	}

	startNow := vit.Now()
	startYear := vit.Now().Year()
	nextYear := startYear

	//Create bills at different years
	vit.TimeAdd(365 * 27 * time.Hour)
	createBill(vit.NextNumber())
	nextYear++

	vit.TimeAdd(365 * 27 * time.Hour)
	time1 := vit.Now()
	table1 := vit.NextNumber()
	offset1 := createBill(table1)
	nextYear++

	vit.TimeAdd(365 * 27 * time.Hour)
	time2 := vit.Now()
	table2 := vit.NextNumber()
	offset2 := createBill(table2)
	nextYear++

	vit.TimeAdd(365 * 27 * time.Hour)
	time3 := vit.Now()
	table3 := vit.NextNumber()
	offset3 := createBill(table3)
	nextYear++

	vit.TimeAdd(365 * 27 * time.Hour)
	time4 := vit.Now()
	table4 := vit.NextNumber()
	offset4 := createBill(table4)
	nextYear++

	vit.TimeAdd(365 * 27 * time.Hour)
	offset := createBill(vit.NextNumber())
	nextYear++

	WaitForIndexOffset(vit, ws, journal.QNameViewWLogDates, offset)

	//Read journal
	// endNow := vit.Now()
	from := time.Date(startYear+2, startNow.Month(), startNow.Day(), startNow.Hour(), startNow.Minute(), startNow.Second()+1, startNow.Nanosecond(), time.UTC).UnixMilli()
	till := vit.Now().UnixMilli()
	body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"all"},
				"elements":[{"fields":["Offset","EventTime","Event"]}]
			}`, from, till)

	resp := vit.PostWS(ws, "q.sys.Journal", body)

	require.Equal(float64(offset1), resp.SectionRow()[0])
	require.Equal(float64(time1.UnixMilli()), resp.SectionRow()[1])
	require.Contains(resp.SectionRow()[2], fmt.Sprintf(`"tableno":%d`, table1))
	require.Equal(float64(offset2), resp.SectionRow(1)[0])
	require.Equal(float64(time2.UnixMilli()), resp.SectionRow(1)[1])
	require.Contains(resp.SectionRow(1)[2], fmt.Sprintf(`"tableno":%d`, table2))
	require.Equal(float64(offset3), resp.SectionRow(2)[0])
	require.Equal(float64(time3.UnixMilli()), resp.SectionRow(2)[1])
	require.Contains(resp.SectionRow(2)[2], fmt.Sprintf(`"tableno":%d`, table3))
	require.Equal(float64(offset4), resp.SectionRow(3)[0])
	require.Equal(float64(time4.UnixMilli()), resp.SectionRow(3)[1])
	require.Contains(resp.SectionRow(3)[2], fmt.Sprintf(`"tableno":%d`, table4))
}
