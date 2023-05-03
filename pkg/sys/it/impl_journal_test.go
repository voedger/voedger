/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	airsbp_it "github.com/untillpro/airs-bp3/packages/air/it"
	"github.com/untillpro/airs-bp3/packages/air/ordersdates"
	"github.com/untillpro/airs-bp3/packages/air/pbilldates"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/journal"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Journal(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	bill := fmt.Sprintf(`{
				"cuds": [{
				  "fields": {
					"sys.ID": 1,
					"sys.QName": "untill.bill",
					"tableno": %d,
					"id_untill_users": 100000000000,
					"table_part": "a",
					"proforma": 3,
					"working_day": "20230228"
				  }
				}]
			}`, tableNum)
	resp := hit.PostWS(ws, "c.sys.CUD", bill)
	ID := resp.NewID()
	expectedOffset := resp.CurrentWLogOffset

	WaitForIndexOffset(hit, ws, journal.QNameViewWLogDates, expectedOffset)

	//Read by unix timestamp
	body := fmt.Sprintf(`
	{
		"args":{"From":%d,"Till":%d,"EventTypes":"all"},
		"elements":[{"fields":["Offset","EventTime","Event"]}]
	}`, hit.Now().UnixMilli(), hit.Now().UnixMilli())
	resp = hit.PostWS(ws, "q.sys.Journal", body)

	require.JSONEq(fmt.Sprintf(`
	{
	  "args": {},
	  "cuds": [
		{
		  "fields": {
			"id_untill_users": 100000000000,
			"proforma": 3,
			"sys.ID": %[1]d,
			"sys.IsActive": true,
			"sys.QName": "untill.bill",
			"table_part": "a",
			"tableno": %[2]d,
			"working_day": "20230228"
		  },
		  "IsNew": true,
		  "sys.ID": %[1]d,
		  "sys.QName": "untill.bill"
		}
	  ],
	  "DeviceID": 0,
	  "RegisteredAt": %[3]d,
	  "Synced": false,
	  "SyncedAt": 0,
	  "sys.QName": "sys.CUD"
	}`, ID, tableNum, hit.Now().UnixMilli()), resp.SectionRow()[2].(string))

	expectedEvent := fmt.Sprintf(`
		{
			"args": {},
			"cuds": [
			{
				"fields": {
				"id_untill_users": 100000000000,
				"proforma": 3,
				"sys.ID": %[1]d,
				"sys.IsActive": true,
				"sys.QName": "untill.bill",
				"table_part": "a",
				"tableno": %[2]d,
				"working_day": "20230228"
				},
				"IsNew": true,
				"sys.ID": %[1]d,
				"sys.QName": "untill.bill"
			}
			],
			"DeviceID": 0,
			"RegisteredAt": %[3]d,
			"Synced": false,
			"SyncedAt": 0,
			"sys.QName": "sys.CUD"
		}`, ID, tableNum, hit.Now().UnixMilli())

	require.Equal(int64(resp.SectionRow()[0].(float64)), expectedOffset)
	require.Equal(int64(resp.SectionRow()[1].(float64)), hit.Now().UnixMilli())
	require.JSONEq(expectedEvent, resp.SectionRow()[2].(string))

	//Read by offset
	body = fmt.Sprintf(`
	{
		"args":{"From":%d,"Till":%d,"EventTypes":"all","RangeUnit":"Offset"},
		"elements":[{"fields":["Offset","EventTime","Event"]}]
	}`, expectedOffset, expectedOffset)
	resp = hit.PostWS(ws, "q.sys.Journal", body)

	require.JSONEq(fmt.Sprintf(`
	{
	  "args": {},
	  "cuds": [
		{
		  "fields": {
			"id_untill_users": 100000000000,
			"proforma": 3,
			"sys.ID": %[1]d,
			"sys.IsActive": true,
			"sys.QName": "untill.bill",
			"table_part": "a",
			"tableno": %[2]d,
			"working_day": "20230228"
		  },
		  "IsNew": true,
		  "sys.ID": %[1]d,
		  "sys.QName": "untill.bill"
		}
	  ],
	  "DeviceID": 0,
	  "RegisteredAt": %[3]d,
	  "Synced": false,
	  "SyncedAt": 0,
	  "sys.QName": "sys.CUD"
	}`, ID, tableNum, hit.Now().UnixMilli()), resp.SectionRow()[2].(string))

	expectedEvent = fmt.Sprintf(`
		{
			"args": {},
			"cuds": [
			{
				"fields": {
				"id_untill_users": 100000000000,
				"proforma": 3,
				"sys.ID": %[1]d,
				"sys.IsActive": true,
				"sys.QName": "untill.bill",
				"table_part": "a",
				"tableno": %[2]d,
				"working_day": "20230228"
				},
				"IsNew": true,
				"sys.ID": %[1]d,
				"sys.QName": "untill.bill"
			}
			],
			"DeviceID": 0,
			"RegisteredAt": %[3]d,
			"Synced": false,
			"SyncedAt": 0,
			"sys.QName": "sys.CUD"
		}`, ID, tableNum, hit.Now().UnixMilli())

	require.Equal(int64(resp.SectionRow()[0].(float64)), expectedOffset)
	require.Equal(int64(resp.SectionRow()[1].(float64)), hit.Now().UnixMilli())
	require.JSONEq(expectedEvent, resp.SectionRow()[2].(string))
}

func TestJournal(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()
	hit.SetNow(hit.Now().AddDate(1, 0, 0))

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	t.Run("Should return error when event type is invalid", func(t *testing.T) {
		body := `{"args":{"From":0,"Till":0,"EventTypes":"wrong"},"elements":[{"fields":["EventTime","Event"]}]}`

		resp := hit.PostWS(ws, "q.sys.Journal", body, utils.Expect500())

		resp.RequireError(t, "invalid event type: wrong")
	})
	t.Run("Should filter not 'bills' events", func(t *testing.T) {
		cuds := `{"cuds":[{"fields":{"sys.ID":10000000,"sys.QName":"untill.pos_emails"}}]}`
		hit.PostWSSys(ws, "c.sys.Init", cuds)
		today := hit.Now()
		body := fmt.Sprintf(`{"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["EventTime","Event"]}]}`,
			today.UnixMilli(),
			today.UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(t, "{}", resp.Body)
	})
	t.Run("Should read by air.PbillDates index", func(t *testing.T) {
		require := require.New(t)
		//create bill
		bill := fmt.Sprintf(`
			{
				"cuds": [
					{
						"fields": {
							"sys.ID": 1,
							"sys.QName": "untill.bill",
							"tableno": %d,
							"id_untill_users": 100000000000,
							"table_part": "a",
							"proforma": 3,
							"working_day": "20230227"
						}
					}
				]
			}`, tableNum)
		resp := hit.PostWS(ws, "c.sys.CUD", bill)
		ID := resp.NewID()

		var offset int64

		//create pbills
		timestamps := [2]int64{hit.Now().Add(time.Second * 1).UnixMilli(), hit.Now().Add(time.Second * 2).UnixMilli()}
		for _, timestamp := range timestamps {
			pbill := fmt.Sprintf(`
			{
				"args": {
					"sys.ID": 1,
					"sys.QName": "untill.pbill",
					"working_day": "%s",
					"id_bill": %d,
					"id_untill_users": 100000000000,
					"pdatetime": %d
				}
			}`, hit.Now().Format("20060102"), ID, timestamp)
			offset = hit.PostWS(ws, "c.air.Pbill", pbill).CurrentWLogOffset
		}

		WaitForIndexOffset(hit, ws, pbilldates.QNameViewPbillDates, offset)

		//read journal
		body := fmt.Sprintf(`
				{
					"args":{"From":%d,"Till":%d,"EventTypes":"all","IndexForTimestamps":"air.PbillDates"},
					"elements":[{"fields":["Event"]}]
				}`, hit.Now().UnixMilli(), hit.Now().Add(time.Hour).UnixMilli())
		resp = hit.PostWS(ws, "q.sys.Journal", body)

		require.Len(resp.Sections[0].Elements, 2)
		require.Contains(resp.SectionRow()[0], fmt.Sprintf(`"pdatetime":%d`, timestamps[0]))
		require.Contains(resp.SectionRow(1)[0], fmt.Sprintf(`"pdatetime":%d`, timestamps[1]))
	})
	t.Run("Should read by air.OrdersDates index", func(t *testing.T) {
		require := require.New(t)
		//create bill
		bill := fmt.Sprintf(`
			{
				"cuds": [
					{
						"fields": {
							"sys.ID": 1,
							"sys.QName": "untill.bill",
							"tableno": %d,
							"id_untill_users": 100000000000,
							"table_part": "a",
							"proforma": 3,
							"working_day": "20230227"
						}
					}
				]
			}`, tableNum)
		resp := hit.PostWS(ws, "c.sys.CUD", bill)
		ID := resp.NewID()

		var offset int64

		//create order
		timestamps := [2]int64{hit.Now().Add(time.Second * 1).UnixMilli(), hit.Now().Add(time.Second * 2).UnixMilli()}
		for _, timestamp := range timestamps {
			order := fmt.Sprintf(`{
								  "args": {
									"sys.ID": 1,
									"id_bill": %d,
									"ord_tableno": %d,
									"ord_datetime": %d,
									"id_untill_users": 100000000000,
									"ord_table_part": "a",
									"working_day": "%s"
								  }
								}`, ID, tableNum, timestamp, hit.Now().Format("20060102"))
			offset = hit.PostWS(ws, "c.air.Orders", order).CurrentWLogOffset
		}

		WaitForIndexOffset(hit, ws, ordersdates.QNameViewOrdersDates, offset)

		//read journal
		body := fmt.Sprintf(`
				{
					"args":{"From":%d,"Till":%d,"EventTypes":"all","IndexForTimestamps":"air.OrdersDates"},
					"elements":[{"fields":["Event"]}]
				}`, hit.Now().UnixMilli(), hit.Now().Add(time.Hour).UnixMilli())
		resp = hit.PostWS(ws, "q.sys.Journal", body)

		require.Len(resp.Sections[0].Elements, 2)
		require.Contains(resp.SectionRow()[0], fmt.Sprintf(`"ord_datetime":%d`, timestamps[0]))
		require.Contains(resp.SectionRow(1)[0], fmt.Sprintf(`"ord_datetime":%d`, timestamps[1]))
	})
	t.Run("Should read 'pbills' events", func(t *testing.T) {
		//create bill
		require := require.New(t)
		bill := fmt.Sprintf(`
			{
				"cuds": [
					{
						"fields": {
							"sys.ID": 1,
							"sys.QName": "untill.bill",
							"tableno": %d,
							"id_untill_users": 100000000000,
							"table_part": "a",
							"proforma": 3,
							"working_day": "20230227"
						}
					}
				]
			}`, tableNum)
		resp := hit.PostWS(ws, "c.sys.CUD", bill)
		ID := resp.NewID()
		//create pbill
		pbill := fmt.Sprintf(`
			{
				"args": {
					"sys.ID": 1,
					"sys.QName": "untill.pbill",
					"working_day": "20220210",
					"id_bill": %d,
					"id_untill_users": 100000000000,
					"pdatetime": 1644486499000
				}
			}`, ID)
		offset := hit.PostWS(ws, "c.air.Pbill", pbill).CurrentWLogOffset

		WaitForIndexOffset(hit, ws, pbilldates.QNameViewPbillDates, offset)

		//read journal
		body := `{"args":{"From":1644523200000,"Till":1644523200000,"EventTypes":"pbills","IndexForTimestamps":"air.PbillDates"},
					"elements":[{"fields":["Event"]}]}`
		resp = hit.PostWS(ws, "q.sys.Journal", body)

		require.Contains(resp.SectionRow()[0], `"pdatetime":1644486499000`)
	})
}

func TestJournal_read_in_years_range_1(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()
	hit.SetNow(hit.Now().AddDate(1, 0, 0))

	setTimestamp := func(year int, month time.Month, day int) time.Time {
		now := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		hit.SetNow(now)
		return now
	}

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")

	createBill := func(tableNo int) int64 {
		bill := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.bill","tableno":%d,"id_untill_users":100000000000,"table_part":"a","proforma":3,"working_day":"20230227"}}]}`, tableNo)
		return hit.PostWS(ws, "c.sys.CUD", bill).CurrentWLogOffset
	}

	startYear := hit.Now().Year()
	nextYear := startYear + 1

	//Create bills at different years
	setTimestamp(nextYear, time.August, 17)
	createBill(hit.NextNumber())
	time1 := setTimestamp(nextYear, time.October, 13)
	table1 := hit.NextNumber()
	offset1 := createBill(table1)
	nextYear++
	time2 := setTimestamp(nextYear, time.June, 5)
	table2 := hit.NextNumber()
	offset2 := createBill(table2)
	nextYear++
	time3 := setTimestamp(nextYear, time.July, 7)
	table3 := hit.NextNumber()
	offset3 := createBill(table3)
	nextYear++
	time4 := setTimestamp(nextYear, time.September, 3)
	table4 := hit.NextNumber()
	offset4 := createBill(table4)
	setTimestamp(nextYear, time.November, 5)
	offset := createBill(hit.NextNumber())

	WaitForIndexOffset(hit, ws, journal.QNameViewWLogDates, offset)

	//Read journal
	from := time.Date(startYear+1, time.August, 18, 0, 0, 0, 0, time.UTC).UnixMilli()
	till := time.Date(nextYear, time.November, 4, 0, 0, 0, 0, time.UTC).UnixMilli()
	body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"all"},
				"elements":[{"fields":["Offset","EventTime","Event"]}]
			}`, from, till)

	resp := hit.PostWS(ws, "q.sys.Journal", body)

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

func TestJournal_read_in_years_range_2(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()
	hit.SetNow(hit.Now().AddDate(50, 0, 0))

	setTimestamp := func(year int, month time.Month, day int) {
		now := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		hit.SetNow(now)
	}

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")

	createBill := func(tableNo int) int64 {
		bill := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.bill","tableno":%d,"id_untill_users":100000000000,"table_part":"a","proforma":3,"working_day":"20230227"}}]}`, tableNo)
		return hit.PostWS(ws, "c.sys.CUD", bill).CurrentWLogOffset
	}

	offsets := func(resp *utils.FuncResponse) []int64 {
		oo := make([]int64, 0)
		for _, e := range resp.Sections[0].Elements {
			oo = append(oo, int64(e[0][0][0].(float64)))
		}
		return oo
	}

	concat := func(ss ...[]int64) []int64 {
		r := make([]int64, 0)
		for _, s := range ss {
			r = append(r, s...)
		}
		return r
	}

	firstYear := hit.Now().Year()
	secondYear := firstYear + 1
	firstYearApril := make([]int64, 0, 3)
	firstYearMay := make([]int64, 0, 3)
	secondYearApril := make([]int64, 0, 3)
	secondYearMay := make([]int64, 0, 3)

	setTimestamp(firstYear, time.April, 1)
	firstYearApril = append(firstYearApril, createBill(hit.NextNumber()))
	setTimestamp(firstYear, time.April, 2)
	firstYearApril = append(firstYearApril, createBill(hit.NextNumber()))
	setTimestamp(firstYear, time.April, 3)
	firstYearApril = append(firstYearApril, createBill(hit.NextNumber()))
	setTimestamp(firstYear, time.May, 1)
	firstYearMay = append(firstYearMay, createBill(hit.NextNumber()))
	setTimestamp(firstYear, time.May, 2)
	firstYearMay = append(firstYearMay, createBill(hit.NextNumber()))
	setTimestamp(firstYear, time.May, 3)
	firstYearMay = append(firstYearMay, createBill(hit.NextNumber()))

	setTimestamp(secondYear, time.April, 1)
	secondYearApril = append(secondYearApril, createBill(hit.NextNumber()))
	setTimestamp(secondYear, time.April, 2)
	secondYearApril = append(secondYearApril, createBill(hit.NextNumber()))
	setTimestamp(secondYear, time.April, 3)
	secondYearApril = append(secondYearApril, createBill(hit.NextNumber()))
	setTimestamp(secondYear, time.May, 1)
	secondYearMay = append(secondYearMay, createBill(hit.NextNumber()))
	setTimestamp(secondYear, time.May, 2)
	secondYearMay = append(secondYearMay, createBill(hit.NextNumber()))
	setTimestamp(secondYear, time.May, 3)
	offset := createBill(hit.NextNumber())
	secondYearMay = append(secondYearMay, offset)

	WaitForIndexOffset(hit, ws, journal.QNameViewWLogDates, offset)

	t.Run("Should read all events with overlapped requested years", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear-2, time.April, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(secondYear+2, time.May, 3, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(concat(firstYearApril, firstYearMay, secondYearApril, secondYearMay), offsets(resp))
	})
	t.Run("Should read all events", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear, time.April, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(secondYear, time.May, 3, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(concat(firstYearApril, firstYearMay, secondYearApril, secondYearMay), offsets(resp))
	})
	t.Run("Should read events from first year may and second year april", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear, time.May, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(secondYear, time.April, 3, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(concat(firstYearMay, secondYearApril), offsets(resp))
	})
	t.Run("Should read some events from first year may and some events from second year april", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear, time.May, 2, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(secondYear, time.April, 1, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(append(firstYearMay[1:], secondYearApril[0]), offsets(resp))
	})
	t.Run("Should read some events from first year april and some events from first year may", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear, time.April, 2, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(firstYear, time.May, 1, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(append(firstYearApril[1:], firstYearMay[0]), offsets(resp))
	})
	t.Run("Should read first year may events", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(firstYear, time.May, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(firstYear, time.May, 3, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(firstYearMay, offsets(resp))
	})
	t.Run("Should read second year april events", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`
			{
				"args":{"From":%d,"Till":%d,"EventTypes":"bills"},"elements":[{"fields":["Offset"]}]
			}`,
			time.Date(secondYear, time.April, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(secondYear, time.April, 3, 0, 0, 0, 0, time.UTC).UnixMilli())

		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(secondYearApril, offsets(resp))
	})
	t.Run("Should read all events from first sys.WLogDates record, in other words - workspace initialization events", func(t *testing.T) {
		require := require.New(t)

		body := fmt.Sprintf(`{
										"args":{"Query":"select * from sys.WLogDates where Year = %d"},
										"elements":[{"fields":["Result"]}]
									}`, it.DefaultTestTime.Year())
		jsonStr := hit.PostWS(ws, "q.sys.SqlQuery", body).SectionRow()[0].(string)

		type wLogDate struct {
			First   int `json:"FirstOffset"`
			Last    int `json:"LastOffset"`
			YearDay int `json:"DayOfYear"`
			Year    int `json:"Year"`
		}
		wld := new(wLogDate)
		require.NoError(json.Unmarshal([]byte(jsonStr), wld))

		timestamp := time.Date(wld.Year, time.January, 0, 0, 0, 0, 0, time.UTC).AddDate(0, 0, wld.YearDay).UnixMilli()

		body = fmt.Sprintf(`{"args":{"From":%d,"Till":%d,"EventTypes":"all"},"elements":[{"fields":["Offset"]}]}`, timestamp, timestamp)
		resp := hit.PostWS(ws, "q.sys.Journal", body)

		require.Equal(1, wld.First)
		require.Len(offsets(resp), wld.Last)
	})
}
