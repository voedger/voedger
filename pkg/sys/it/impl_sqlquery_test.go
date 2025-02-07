/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/registry"
	"github.com/voedger/voedger/pkg/sys/sqlquery"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_SqlQuery(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	idUntillUsers := vit.GetAny("app1pkg.untill_users", ws)

	findPLogOffsetByWLogOffset := func(wLogOffset istructs.Offset) istructs.Offset {
		type row struct {
			Workspace  istructs.WSID
			PlogOffset istructs.Offset
			WLogOffset istructs.Offset
		}
		body := `{"args":{"Query":"select Workspace, PlogOffset, WLogOffset from sys.plog limit -1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
		for _, element := range resp.Sections[0].Elements {
			r := new(row)
			require.NoError(json.Unmarshal([]byte(element[0][0][0].(string)), r))
			if r.Workspace == ws.WSID && r.WLogOffset == wLogOffset {
				return r.PlogOffset
			}
		}
		panic("PlogOffset not found")
	}

	tableNum := vit.NextNumber()

	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.bill","tableno":%d,"id_untill_users":%d,"table_part":"a","proforma":0,"working_day":"20230227"}}]}`, tableNum, idUntillUsers)
	vit.PostWS(ws, "c.sys.CUD", body)
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.payments","name":"EFT","guid":"0a53b7c6-2c47-491c-ac00-307b8d5ba6f2"}}]}`
	resp := vit.PostWS(ws, "c.sys.CUD", body)

	body = fmt.Sprintf(`{"args":{"Query":"select CUDs from sys.plog where Offset>=%d"},"elements":[{"fields":["Result"]}]}`, findPLogOffsetByWLogOffset(resp.CurrentWLogOffset))
	resp = vit.PostWS(ws, "q.sys.SqlQuery", body)

	require.Contains(resp.SectionRow()[0], "0a53b7c6-2c47-491c-ac00-307b8d5ba6f2")
}

func TestSqlQuery_plog(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	idUntillUsers := vit.GetAny("app1pkg.untill_users", ws)

	pLogSize := 0
	// it is wrong to consider last resp.CurrentWLogOffset as the pLog events amount because pLog contains events from different workspaces
	// currently log of partition 0 contains events from 2 workspaces: pseudo 140737488420870 and newely created 140737488486400
	// following util shows the initial content on pLog of partition 0:
	t.Run("print the pLog content", func(t *testing.T) {
		require := require.New(t)
		body := `{"args":{"Query":"select * from sys.plog limit -1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		for _, intf := range resp.Sections[0].Elements {
			m := map[string]interface{}{}
			require.NoError(json.Unmarshal([]byte(intf[0][0][0].(string)), &m))
			log.Println(int(m["Workspace"].(float64)), m["PlogOffset"], m["WLogOffset"])
		}
		pLogSize = len(resp.Sections[0].Elements)
	})
	// note that we have wlogOffset 7 twice, so the last resp.CurrentWLogOffset is not the amount of events in pLog
	// currently events amount is 13, the last resp.CurrentWLogOffset is 12:
	/*
		140737488420870 1 7
		140737488486400 2 1
		140737488486400 3 2
		140737488486400 4 3
		140737488486400 5 4
		140737488486400 6 5
		140737488486400 7 6
		140737488486400 8 7
		140737488486400 9 8
		140737488486400 10 9
		140737488486400 11 10
		140737488486400 12 11
		140737488486400 13 12
	*/

	for i := 1; i <= 101; i++ {
		tableno := vit.NextNumber()
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":%d,"sys.QName":"app1pkg.bill","tableno":%d,"id_untill_users":%d,"table_part":"a","proforma":0,"working_day":"20230227"}}]}`, i, tableno, idUntillUsers)
		vit.PostWS(ws, "c.sys.CUD", body)
		pLogSize++
	}

	time.Sleep(ProjectionAwaitTime)

	t.Run("Should read events with default Offset and limit", func(t *testing.T) {
		require := require.New(t)
		body := `{"args":{"Query":"select * from sys.plog"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal(istructs.FirstOffset, istructs.Offset(m["PlogOffset"].(float64)))
		require.Len(resp.Sections[0].Elements, sqlquery.DefaultLimit)
	})

	lastPLogOffset := 0

	t.Run("Should read all events", func(t *testing.T) {
		require := require.New(t)
		body := `{"args":{"Query":"select * from sys.plog limit -1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal(istructs.FirstOffset, istructs.Offset(m["PlogOffset"].(float64)))
		require.GreaterOrEqual(len(resp.Sections[0].Elements), pLogSize)

		m = map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)), &m))
		lastPLogOffset = int(m["PlogOffset"].(float64))

	})
	t.Run("Should read one event by limit", func(t *testing.T) {
		require := require.New(t)
		body := `{"args":{"Query":"select * from sys.plog limit 1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 1)
	})
	t.Run("Should read one event by Offset", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.plog where Offset > %d"},"elements":[{"fields":["Result"]}]}`, lastPLogOffset-1)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal(lastPLogOffset, int(m["PlogOffset"].(float64)))
		require.Len(resp.Sections[0].Elements, 1)
	})
	t.Run("Should read two events by Offset", func(t *testing.T) {
		require := require.New(t)
		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.plog where Offset >= %d"},"elements":[{"fields":["Result"]}]}`, lastPLogOffset-1)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 2)

		m := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &m))
		require.Equal(lastPLogOffset-1, int(m["PlogOffset"].(float64)))
		m = map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(resp.SectionRow(1)[0].(string)), &m))
		require.Equal(lastPLogOffset, int(m["PlogOffset"].(float64)))
	})
	t.Run("Should read event with specified Offset", func(t *testing.T) {
		require := require.New(t)
		specifiedOffset := lastPLogOffset - 52
		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.plog where Offset = %d"},"elements":[{"fields":["Result"]}]}`, specifiedOffset)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 1)
		require.Contains(resp.SectionRow()[0], fmt.Sprintf(`"PlogOffset":%d`, specifiedOffset))
	})
	t.Run("Should return error when field not found in def", func(t *testing.T) {
		body := `{"args":{"Query":"select abracadabra from sys.plog"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "field 'abracadabra' not found in def")
	})

	t.Run("select operation is allowed only", func(t *testing.T) {
		body := `{"args":{"Query":"update sys.plog set a = 1"}}`
		vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect400("'select' operation is expected"))
	})
}

func TestSqlQuery_wlog(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	idUntillUsers := vit.GetAny("app1pkg.untill_users", ws)

	var lastWLogOffset istructs.Offset
	for i := 1; i <= 101; i++ {
		tableno := vit.NextNumber()
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":%d,"sys.QName":"app1pkg.bill","tableno":%d,"id_untill_users":%d,"table_part":"a","proforma":0,"working_day":"20230227"}}]}`, i, tableno, idUntillUsers)
		resp := vit.PostWS(ws, "c.sys.CUD", body)
		lastWLogOffset = resp.CurrentWLogOffset
	}
	wLogEventsAmount := int(lastWLogOffset)

	t.Run("Should read events with default Offset and limit", func(t *testing.T) {
		require := require.New(t)

		body := `{"args":{"Query":"select * from sys.wlog"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 100)
	})
	t.Run("Should read all events", func(t *testing.T) {
		require := require.New(t)

		body := `{"args":{"Query":"select * from sys.wlog limit -1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, wLogEventsAmount)
	})
	t.Run("Should read one event by limit", func(t *testing.T) {
		require := require.New(t)

		body := `{"args":{"Query":"select * from sys.wlog limit 1"},"elements":[{"fields":["Result"]}]}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 1)
	})
	t.Run("Should read one event by Offset", func(t *testing.T) {
		require := require.New(t)

		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset > %d"},"elements":[{"fields":["Result"]}]}`, lastWLogOffset-1)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 1)
	})
	t.Run("Should read two events by Offset", func(t *testing.T) {
		require := require.New(t)

		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset >= %d"},"elements":[{"fields":["Result"]}]}`, lastWLogOffset-1)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Len(resp.Sections[0].Elements, 2)
	})
	t.Run("Should return error when field not found in def", func(t *testing.T) {
		body := `{"args":{"Query":"select abracadabra from sys.wlog"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "field 'abracadabra' not found in def")
	})
}

func TestSqlQuery_readLogParams(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("Should return error when limit value not parsable", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog limit 7.1"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, `strconv.ParseInt: parsing "7.1": invalid syntax`)
	})
	t.Run("Should return error when limit value invalid", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog limit -3"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "limit must be greater than -2")
	})
	t.Run("Should return error when Offset value not parsable", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog where Offset >= 2.1"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, `strconv.ParseUint: parsing "2.1": invalid syntax`)
	})
	t.Run("Should return error when Offset value invalid", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog where Offset >= 0"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "offset must be greater than zero")
	})
	t.Run("Should return error when Offset operation not supported", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog where Offset < 2"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "unsupported operation: <")
	})
	t.Run("Should return error when column name not supported", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.plog where something >= 1"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "unsupported column name: something")
	})
	t.Run("Should return error when expression not supported", func(t *testing.T) {
		body := `{"args":{"Query":"select * from sys.wlog where Offset >= 1 and something >= 5"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "unsupported expression: *sqlparser.AndExpr")
	})
}

func TestSqlQuery_records(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.payments","name":"EFT","guid":"guidEFT"}},
					   {"fields":{"sys.ID":2,"sys.QName":"app1pkg.payments","name":"Cash","guid":"guidCash"}},
					   {"fields":{"sys.ID":3,"sys.QName":"app1pkg.pos_emails","description":"invite"}}]}`
	res := vit.PostWS(ws, "c.sys.CUD", body)

	eftId := res.NewID()
	cashId := res.NewIDs["2"]
	emailId := res.NewIDs["3"]

	t.Run("Should read record with all fields by ID", func(t *testing.T) {
		require := require.New(t)
		body = fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.payments where id = %d"},"elements":[{"fields":["Result"]}]}`, eftId)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		resStr := resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)
		require.Contains(resStr, `"sys.QName":"app1pkg.payments"`)
		require.Contains(resStr, fmt.Sprintf(`"sys.ID":%d`, eftId))
		require.Contains(resStr, `"guid":"guidEFT"`)
		require.Contains(resStr, `"name":"EFT"`)
		require.Contains(resStr, `"sys.IsActive":true`)
	})
	t.Run("Should read records with one field by IDs range", func(t *testing.T) {
		require := require.New(t)
		body = fmt.Sprintf(`{"args":{"Query":"select name, sys.IsActive from app1pkg.payments where id in (%d,%d)"}, "elements":[{"fields":["Result"]}]}`, eftId, cashId)
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)

		require.Equal(`{"name":"EFT","sys.IsActive":true}`, resp.SectionRow()[0])
		require.Equal(`{"name":"Cash","sys.IsActive":true}`, resp.SectionRow(1)[0])
	})

	t.Run("errors", func(t *testing.T) {
		cases := map[string]string{
			`{"args":{"Query":"select * from app1pkg.payments where something = 1"}}`:                                      "unsupported column name: something",
			`{"args":{"Query":"select * from app1pkg.payments where id = 2.3"}}`:                                           `parsing "2.3": invalid syntax`,
			`{"args":{"Query":"select * from app1pkg.payments where id in (1.3)"}}`:                                        `parsing "1.3": invalid syntax`,
			`{"args":{"Query":"select * from app1pkg.payments where id >= 2"}}`:                                            "unsupported operation: >=",
			`{"args":{"Query":"select * from app1pkg.payments where id = 2 and something = 2"}}`:                           "unsupported expression: *sqlparser.AndExpr",
			`{"args":{"Query":"select * from app1pkg.payments"}}`:                                                          "'app1pkg.payments' is not a singleton. At least one record ID must be provided",
			fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.payments where id = %d"}}`, emailId):                      fmt.Sprintf("record with ID '%d' has mismatching QName 'app1pkg.pos_emails'", emailId),
			fmt.Sprintf(`{"args":{"Query":"select * from app1pkg.payments where id = %d"}}`, istructs.NonExistingRecordID): fmt.Sprintf("record with ID '%d' not found", istructs.NonExistingRecordID),
			fmt.Sprintf(`{"args":{"Query":"select abracadabra from app1pkg.pos_emails where id = %d"}}`, emailId):          "not found: field «abracadabra» in CDoc «app1pkg.pos_emails»",
			`{"args":{"Query":"select * from app1pkg.payments.2 where id = 2"}}`:                                           "record ID and 'where id ...' clause can not be used in one query",
			`{"args":{"Query":"select sys.QName from app1pkg.test_ws.1"}}`:                                                 "conditions are not allowed to query a singleton",
			`{"args":{"Query":"select sys.QName from app1pkg.test_ws where id = 1"}}`:                                      "conditions are not allowed to query a singleton",
		}

		for query, expectedError := range cases {
			t.Run(expectedError, func(t *testing.T) {
				vit.PostWS(ws, "q.sys.SqlQuery", query, coreutils.Expect400(expectedError))
			})
		}
	})

	t.Run("Should read singleton", func(t *testing.T) {
		require := require.New(t)
		body = `{"args":{"Query":"select sys.QName from app1pkg.test_ws"},"elements":[{"fields":["Result"]}]}`
		restaurant := vit.PostWS(ws, "q.sys.SqlQuery", body).SectionRow(0)

		require.Equal(`{"sys.QName":"app1pkg.test_ws"}`, restaurant[0])
	})
}

func TestSqlQuery_view_records(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.payments","name":"EFT","guid":"guidEFT"}},
					   {"fields":{"sys.ID":2,"sys.QName":"app1pkg.pos_emails","description":"invite"}}]}`
	resp := vit.PostWS(ws, "c.sys.CUD", body)
	paymentsID := resp.NewID()
	lastWLogOffset := resp.CurrentWLogOffset

	t.Run("Should read record with all fields", func(t *testing.T) {
		require := require.New(t)
		body = `{"args":{"Query":"select * from sys.CollectionView where PartKey = 1 and DocQName = 'app1pkg.payments'"}, "elements":[{"fields":["Result"]}]}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body)

		respStr := resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)
		require.Contains(respStr, fmt.Sprintf(`"DocID":%d`, paymentsID))
		require.Contains(respStr, `"DocQName":"app1pkg.payments"`)
		require.Contains(respStr, `"ElementID":0`)
		require.Contains(respStr, fmt.Sprintf(`"offs":%d`, lastWLogOffset))
		require.Contains(respStr, `"PartKey":1`)
		require.Contains(respStr, `"Record":{`)
		require.Contains(respStr, `"sys.QName":"sys.CollectionView"`)
	})
	t.Run("Should return error when operator not supported", func(t *testing.T) {
		body = `{"args":{"Query":"select * from sys.CollectionView where partKey > 1"}}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "unsupported operator: >")
	})
	t.Run("Should return error when expression not supported", func(t *testing.T) {
		body = `{"args":{"Query":"select * from sys.CollectionView where partKey = 1 or docQname = 'app1pkg.payments'"}}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "unsupported expression: *sqlparser.OrExpr")
	})
	t.Run("Should return error when field does not exist in value def", func(t *testing.T) {
		body = `{"args":{"Query":"select abracadabra from sys.CollectionView where PartKey = 1"}}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "field 'abracadabra' does not exist in 'sys.CollectionView' value def")
	})
	t.Run("Should return error when field does not exist in key def", func(t *testing.T) {
		body = `{"args":{"Query":"select * from sys.CollectionView where partKey = 1"}}`
		resp = vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "field 'partKey' does not exist in 'sys.CollectionView' key def")
	})
}

func TestSqlQuery(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("Should return error when script invalid", func(t *testing.T) {
		body := `{"args":{"Query":" "}}`
		vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect400("invalid query format"))
	})
	t.Run("Should return error when source of data unsupported", func(t *testing.T) {
		body := `{"args":{"Query":"select * from git.hub"}}`
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect500())

		resp.RequireError(t, "do not know how to read from the requested git.hub, TypeKind_null")
	})
	t.Run("Should read sys.wlog from other workspace", func(t *testing.T) {
		wsOne := vit.PostWS(ws, "q.sys.SqlQuery", fmt.Sprintf(`{"args":{"Query":"select * from %d.sys.wlog"}}`, ws.Owner.ProfileWSID))
		wsTwo := vit.PostWS(ws, "q.sys.SqlQuery", `{"args":{"Query":"select * from sys.wlog"}}`)

		require.NotEqual(t, len(wsOne.Sections[0].Elements), len(wsTwo.Sections[0].Elements))
	})

	t.Run("403 forbidden on read from non-inited workspace", func(t *testing.T) {
		vit.PostWS(ws, "q.sys.SqlQuery", fmt.Sprintf(`{"args":{"Query":"select * from %d.sys.wlog"}}`, istructs.NonExistingRecordID),
			coreutils.Expect403(processors.ErrWSNotInited.Message))
	})
}

func TestReadFromWLogWithSysRawArg(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	lastOffset := vit.PostWS(ws, "c.app1pkg.TestCmdRawArg", "hello world").CurrentWLogOffset

	body := fmt.Sprintf(`{"args":{"Query":"select * from sys.wlog where Offset > %d"},"elements":[{"fields":["Result"]}]}`, lastOffset-1)
	resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
	res := resp.SectionRow()[0].(string)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(res), &m))
	rawArg := m["ArgumentObject"].(map[string]interface{})["Body"].(string)
	require.Equal("hello world", rawArg)
}

func TestReadFromAnDifferentLocations(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	oneAppWS := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("wsid", func(t *testing.T) {
		// create a record in one workspace of one app
		categoryName := vit.NextName()
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"%s"}}]}`, categoryName)
		categoryID := vit.PostWS(oneAppWS, "c.sys.CUD", body).NewID()

		// create a workspace in another app
		anotherAppWSOwner := vit.GetPrincipal(istructs.AppQName_test1_app2, "login")
		qNameApp2_TestWSKind := appdef.NewQName("app2pkg", "test_ws")
		anotherAppWS := vit.CreateWorkspace(it.WSParams{
			Name:         "anotherAppWS",
			Kind:         qNameApp2_TestWSKind,
			ClusterID:    istructs.CurrentClusterID(),
			InitDataJSON: `{"IntFld":42}`,
		}, anotherAppWSOwner)

		// in the another app use sql to query the record from the first app
		body = fmt.Sprintf(`{"args":{"Query":"select * from test1.app1.%d.app1pkg.category.%d"},"elements":[{"fields":["Result"]}]}`, oneAppWS.WSID, categoryID)
		resp := vit.PostWS(anotherAppWS, "q.sys.SqlQuery", body)
		resStr := resp.SectionRow(len(resp.Sections[0].Elements) - 1)[0].(string)
		require.Contains(resStr, fmt.Sprintf(`"name":"%s"`, categoryName))
	})

	t.Run("app workspace number", func(t *testing.T) {
		// determine the number of the app workspace that stores cdoc.Login "login"
		registryAppStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_sys_registry)
		require.NoError(err)
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login") // from VIT shared config
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, prn.Name, istructs.CurrentClusterID())
		appWSNumber := pseudoWSID.BaseWSID() % istructs.WSID(registryAppStructs.NumAppWorkspaces())

		// for example read cdoc.registry.Login.LoginHash from the app workspace
		loginID := vit.GetCDocLoginID(prn.Login)
		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.registry.a%d.registry.Login where id = %d"},"elements":[{"fields":["Result"]}]}`, appWSNumber, loginID)
		resp := vit.PostWS(oneAppWS, "q.sys.SqlQuery", body)
		loginHash := registry.GetLoginHash(prn.Login.Name)
		require.Contains(resp.SectionRow()[0].(string), fmt.Sprintf(`"LoginHash":"%s"`, loginHash))
	})

	t.Run("login hash", func(t *testing.T) {
		// for example read cdoc.registry.Login.LoginHash from the app workspace determined by the login name
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login") // from VIT shared config
		loginID := vit.GetCDocLoginID(prn.Login)
		body := fmt.Sprintf(`{"args":{"Query":"select * from sys.registry.\"login\".registry.Login where id = %d"},"elements":[{"fields":["Result"]}]}`, loginID)
		resp := vit.PostWS(oneAppWS, "q.sys.SqlQuery", body)
		loginHash := registry.GetLoginHash(prn.Login.Name)
		require.Contains(resp.SectionRow()[0].(string), fmt.Sprintf(`"LoginHash":"%s"`, loginHash))
	})

	t.Run("query forwarding", func(t *testing.T) {
		wsAnother := vit.WS(istructs.AppQName_test1_app1, "test_ws_another")
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		body := fmt.Sprintf(`{"args":{"Query":"select * from %d.sys.wlog"},"elements":[{"fields":["Result"]}]}`, ws.WSID)
		resp := vit.PostWS(wsAnother, "q.sys.SqlQuery", body)
		require.GreaterOrEqual(resp.NumRows(), 2)
		resp.Println()
	})

	t.Run("query forwarding with empty result", func(t *testing.T) {
		wsAnother := vit.WS(istructs.AppQName_test1_app1, "test_ws_another")
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		body := fmt.Sprintf(`{"args":{"Query":"select * from %d.sys.wlog where offset = %d"},"elements":[{"fields":["Result"]}]}`, ws.WSID, istructs.NonExistingRecordID)
		resp := vit.PostWS(wsAnother, "q.sys.SqlQuery", body)
		require.True(resp.IsEmpty())
	})
}

func TestAuthnz(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("doc", func(t *testing.T) {
		body := `{"args":{"Query":"select * from app1pkg.TestDeniedCDoc.123"},"elements":[{"fields":["Result"]}]}`
		vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect403())
	})

	t.Run("field", func(t *testing.T) {
		// denied
		body := `{"args":{"Query":"select DeniedFld2 from app1pkg.TestCDocWithDeniedFields.123"},"elements":[{"fields":["Result"]}]}`
		vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect403())

		// allowed, just expect 400 not found
		body = `{"args":{"Query":"select Fld1 from app1pkg.TestCDocWithDeniedFields.123"},"elements":[{"fields":["Result"]}]}`
		vit.PostWS(ws, "q.sys.SqlQuery", body, coreutils.Expect400("record with ID '123' not found"))
	})
}
