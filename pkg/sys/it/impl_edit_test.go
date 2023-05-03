/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	airsbp_it "github.com/untillpro/airs-bp3/packages/air/it"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/journal"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestOrdersCmd(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	// Create bill on a table
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

	// Create order on a table
	order := fmt.Sprintf(`
	{
		"args": {
			"sys.ID": 1,
			"id_bill": %d,
			"ord_tableno": %d,
			"ord_datetime": 1639237549,
			"id_untill_users": 100000000000,
			"ord_table_part": "a",
			"working_day":"20211222",
			"order_item": [
				{
					"sys.ID": 2,
					"sys.ParentID": 1,
					"id_orders": 1,
					"rowbeg": 1,
					"kind": 4,
					"quantity": -1,
					"price": 6,
					"original_price": 7
				}
			]
		}
	}`, ID, tableNum)
	resp = hit.PostWS(ws, "c.air.Orders", order, utils.Expect500())
	resp.RequireError(t, "negative total -1.000000 due to order")
}

func TestPBillCmd(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	// Create bill on a table
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

	articleID := CreateArticle(hit, ws)
	waiterID := CreateWaiter(hit, ws)

	// Make payment on the table
	pbill := fmt.Sprintf(`
	{
		"args":{
			"sys.ID":11,
			"id_bill":%d,
			"id_untill_users":%d,
			"pdatetime":1639237570,
			"working_day":"20211222",
			"pbill_payments":[
				{
					"sys.ID":12,
					"sys.ParentID":11,
					"id_pbill":11,
					"id_payments":5000000059,
					"price":200
				}
			],
			"sold_articles":[
				{
					"sys.ID":13,
					"id_pbill":11,
					"sys.ParentID":11,
					"quantity":1,
					"sa_coef":1.0,
					"id_articles":%d,
					"price":200
				}
			]
		}
	}`, ID, waiterID, articleID)
	resp = hit.PostWS(ws, "c.air.Pbill", pbill, utils.Expect500())
	resp.RequireError(t, "negative total -1.000000 due to pbill")
}

func TestSplitBill(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer func() {
		hit.TearDown()
	}()
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	// Create bill on a table
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

	// Create order on a table
	order := fmt.Sprintf(`
								{
									"args": {
										"sys.ID": 1,
										"id_bill": %d,
										"ord_tableno": %d,
										"ord_datetime": 1639237549,
										"id_untill_users": 100000000000,
										"ord_table_part": "a",
										"working_day":"20211222",
										"order_item": [
											{
												"sys.ID": 2,
												"sys.ParentID": 1,
												"id_orders": 1,
												"rowbeg": 1,
												"kind": 4,
												"quantity": 5,
												"price": 6,
												"original_price": 7
											}
										]
									}
								}`, ID, tableNum)
	hit.PostWS(ws, "c.air.Orders", order)

	articleID := CreateArticle(hit, ws)

	// Pay first part
	pbill := fmt.Sprintf(`
								{
									"args":{
										"sys.ID":11,
										"id_bill":%d,
										"id_untill_users":100000000000,
										"pdatetime":1639237570,
										"working_day":"20211222",
										"pbill_payments":[
											{
												"sys.ID":12,
												"sys.ParentID":11,
												"id_pbill":11,
												"id_payments":5000000059,
												"price":200
											}
										],
										"sold_articles":[
											{
												"sys.ID":13,
												"id_pbill":11,
												"sys.ParentID":11,
												"quantity":5,
												"sa_coef":0.49963636363636366,
												"id_articles":%d,
												"price":200
											}
										]
									}
								}`, ID, articleID)
	hit.PostWS(ws, "c.air.Pbill", pbill)

	pdatetime := 1639237570
	// Pay second part
	pbill = fmt.Sprintf(`
								{
									"args":{
										"sys.ID":11,
										"id_bill":%d,
										"id_untill_users":100000000000,
										"pdatetime":%d,
										"working_day":"20211222",
										"pbill_payments":[
											{
												"sys.ID":12,
												"sys.ParentID":11,
												"id_pbill":11,
												"id_payments":5000000059,
												"price":200
											}
										],
										"sold_articles":[
											{
												"sys.ID":13,
												"id_pbill":11,
												"sys.ParentID":11,
												"quantity":5,
												"sa_coef":0.5003636363636363,
												"id_articles":%d,
												"price":200
											}
										]
									}
								}`, ID, pdatetime, articleID)
	offset := hit.PostWS(ws, "c.air.Pbill", pbill).CurrentWLogOffset

	WaitForIndexOffset(hit, ws, journal.QNameViewWLogDates, offset)

	timestamp := hit.Now().UnixMilli()
	body := fmt.Sprintf(`
									{
										"args":{"From":%d,"Till":%d,"EventTypes":"bills"},
										"elements":[{"fields":["Offset","EventTime","Event"]}],
										"orderBy":[{"field":"EventTime", "desc":true}]
									}`, timestamp, timestamp)

	resp = hit.PostWS(ws, "q.sys.Journal", body)

	require.Contains(t, resp.Body, fmt.Sprintf(`\"close_datetime\":%d`, pdatetime))
}

func TestSplitBill_603555(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	bill := fmt.Sprintf(`{"cuds":[{"fields":{
								 "sys.ID": 1,
							     "sys.QName": "untill.bill",
								 "tableno": %d,
								 "id_untill_users": 100000000000,
								 "table_part": "a",
							     "proforma":0,
								 "working_day":"20230227"}}]}`, tableNum)
	ID := hit.PostWS(ws, "c.sys.CUD", bill).NewID()

	order := fmt.Sprintf(`
								{
									"args": {
										"sys.ID": 1,
										"id_bill": %d,
										"ord_tableno": %d,
										"ord_datetime": 1649420458815,
										"id_untill_users": 100000000000,
										"ord_table_part": "a",
										"working_day":"20220408",
										"order_item": [
											{
												"sys.ID": 2,
												"sys.ParentID": 1,
												"id_orders": 1,
												"rowbeg": 1,
												"kind": 1,
												"quantity": 1,
												"price": 45000,
												"original_price": 45000
											},
											{
												"sys.ID": 3,
												"sys.ParentID": 1,
												"id_orders": 1,
												"rowbeg": 0,
												"kind": 2,
												"quantity": 1,
												"price": 10000,
												"original_price": 10000
											},
											{
												"sys.ID": 4,
												"sys.ParentID": 1,
												"id_orders": 1,
												"rowbeg": 1,
												"kind": 1,
												"quantity": 1,
												"price": 24000,
												"original_price": 24000
											},
											{
												"sys.ID": 5,
												"sys.ParentID": 1,
												"id_orders": 1,
												"rowbeg": 1,
												"kind": 1,
												"quantity": 1,
												"price": 24000,
												"original_price": 24000
											}
										]
									}
								}`, ID, tableNum)
	hit.PostWS(ws, "c.air.Orders", order)

	articleID := CreateArticle(hit, ws)

	pbill := fmt.Sprintf(`
								{
									"args":{
										"sys.ID":1,
										"id_bill":%d,
										"id_untill_users":100000000000,
										"pdatetime":1649420496841,
										"working_day":"20220408",
										"sold_articles":[
											{
												"sys.ID":2,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":3,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":24000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":24000
											}
										]
									}
								}`, ID, articleID)
	hit.PostWS(ws, "c.air.Pbill", pbill)

	pbill = fmt.Sprintf(`
								{
									"args":{
										"sys.ID":1,
										"id_bill":%d,
										"id_untill_users":100000000000,
										"pdatetime":1649420497085,
										"working_day":"20220408",
										"sold_articles":[
											{
												"sys.ID":2,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":3,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":24000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.33300970873786406,
												"id_articles":%[2]d,
												"price":24000
											}
										]
									}
								}`, ID, articleID)
	hit.PostWS(ws, "c.air.Pbill", pbill)

	pbill = fmt.Sprintf(`
								{
									"args":{
										"sys.ID":1,
										"id_bill":%d,
										"id_untill_users":100000000000,
										"pdatetime":1649420497300,
										"working_day":"20220408",
										"sold_articles":[
											{
												"sys.ID":2,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.3339805825242718,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":3,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.3339805825242718,
												"id_articles":%[2]d,
												"price":45000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.3339805825242718,
												"id_articles":%[2]d,
												"price":24000
											},
											{
												"sys.ID":4,
												"id_pbill":1,
												"sys.ParentID":1,
												"quantity":1,
												"sa_coef":0.3339805825242718,
												"id_articles":%[2]d,
												"price":24000
											}
										]
									}
								}`, ID, articleID)
	hit.PostWS(ws, "c.air.Pbill", pbill)

	bill = fmt.Sprintf(`{"cuds":[{"fields":{
								 "sys.ID": 1,
							     "sys.QName": "untill.bill",
								 "tableno": %d,
								 "id_untill_users": 100000000000,
								 "table_part": "a",
							     "proforma":0,
								 "working_day":"20230227"}}]}`, tableNum)
	hit.PostWS(ws, "c.sys.CUD", bill)
}

func TestOrderItemWithKindArticleMessage(t *testing.T) {
	hit := it.NewHIT(t, &airsbp_it.SharedConfig_Air)
	hit.TearDown()
	ws := hit.WS(istructs.AppQName_untill_airs_bp, "test_restaurant")
	tableNum := hit.NextNumber()

	// Create bill
	bill := fmt.Sprintf(`{
			  "cuds": [
				{
				  "fields": {
					"sys.ID": 1,
					"sys.QName": "untill.bill",
					"tableno": %d,
					"id_untill_users": 100000000000,
					"table_part": "a",
					"proforma": 0,
					"working_day": "20230227"
				  }
				}
			  ]
			}`, tableNum)
	resp := hit.PostWS(ws, "c.sys.CUD", bill)
	billID := resp.NewID()

	// Create order with order item with kind 'Article message'
	order := fmt.Sprintf(`{
								  "args": {
									"sys.ID": 1,
									"id_bill": %d,
									"ord_tableno": %d,
									"ord_datetime": 1639237549,
									"id_untill_users": 100000000000,
									"ord_table_part": "a",
									"working_day": "20211222",
									"order_item": [
									  {
										"sys.ID": 2,
										"sys.ParentID": 1,
										"id_orders": 1,
										"rowbeg": 1,
										"kind": 1,
										"quantity": 3,
										"price": 600,
										"original_price": 600
									  },
									  {
										"sys.ID": 3,
										"sys.ParentID": 1,
										"id_orders": 1,
										"rowbeg": 1,
										"kind": 3,
										"quantity": 7,
										"price": 0,
										"original_price": 0
									  }
									]
								  }
								}`, billID, tableNum)
	hit.PostWS(ws, "c.air.Orders", order)

	articleID := CreateArticle(hit, ws)

	// Create payment
	pdatetime := 1639237570
	pbill := fmt.Sprintf(`
	{
		"args":{
			"sys.ID":1,
			"id_bill":%d,
			"id_untill_users":100000000000,
			"pdatetime":%d,
			"working_day":"20211222",
			"sold_articles":[
				{
					"sys.ID":2,
					"id_pbill":1,
					"sys.ParentID": 1,
					"quantity":3,
					"sa_coef":1.0,
					"id_articles":%d,
					"price":0
				}
			]
		}
	}`, billID, pdatetime, articleID)
	offset := hit.PostWS(ws, "c.air.Pbill", pbill).CurrentWLogOffset

	WaitForIndexOffset(hit, ws, journal.QNameViewWLogDates, offset)

	timestamp := hit.Now().UnixMilli()
	body := fmt.Sprintf(`
									{
										"args":{"From":%d,"Till":%d,"EventTypes":"bills"},
										"elements":[{"fields":["Offset","EventTime","Event"]}],
										"orderBy":[{"field":"EventTime", "desc":true}]
									}`, timestamp, timestamp)
	resp = hit.PostWS(ws, "q.sys.Journal", body)

	require.Contains(t, resp.Body, fmt.Sprintf(`\"close_datetime\":%d`, pdatetime))
}
