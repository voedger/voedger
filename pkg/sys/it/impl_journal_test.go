/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
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

	expectedEvent := fmt.Sprintf(`
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
