/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import "github.com/voedger/voedger/pkg/exttinygo"

var Package_untill = struct {
	CDoc_articles CDoc_untill_articles
	ODoc_pbill    ODoc_untill_pbill
	WDoc_bill     WDoc_untill_bill
}{
	CDoc_articles: CDoc_untill_articles{
		Type: Type{qname: "https://github.com/untillpro/airs-scheme/bp3.untill_articles"},
	},
	WDoc_bill: WDoc_untill_bill{
		Type: Type{qname: "https://github.com/untillpro/airs-scheme/bp3.bill"},
	},
}

/*
TABLE bill INHERITS sys.WDoc (
	close_datetime int64,
	table_name varchar(50),
	tableno int32 NOT NULL,
	id_untill_users ref(untill_users) NOT NULL,
	table_part varchar(1) NOT NULL,
)
*/

type WDoc_untill_bill struct {
	Type
}

func (w WDoc_untill_bill) QueryValue(id ID) (Value_Table_untill_bill, bool) {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, w.qname)
	kb.PutInt64(FieldNameSysID, int64(id))
	tv, exists := exttinygo.QueryValue(kb)
	return Value_Table_untill_bill{tv: tv, kb: kb}, exists
}

func (w WDoc_untill_bill) MustGetValue(id ID) Value_Table_untill_bill {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, w.qname)
	kb.PutInt64(FieldNameSysID, int64(id))
	tv := exttinygo.MustGetValue(kb)
	return Value_Table_untill_bill{tv: tv, kb: kb}
}

func (w WDoc_untill_bill) NewIntent(id ID) Intent_WDoc_untill_bill {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, w.qname)
	kb.PutInt64(FieldNameSysID, int64(id))
	return Intent_WDoc_untill_bill{intent: exttinygo.NewValue(kb)}
}

type Value_Table_untill_bill struct {
	tv exttinygo.TValue
	kb exttinygo.TKeyBuilder
}

func (v Value_Table_untill_bill) Get_close_datetime() int64 {
	return v.tv.AsInt64("close_datetime")
}
func (v Value_Table_untill_bill) Get_table_name() string {
	return v.tv.AsString("table_name")
}
func (v Value_Table_untill_bill) Get_tableno() int32 {
	return v.tv.AsInt32("tableno")
}
func (v Value_Table_untill_bill) Get_id_untill_users() Ref {
	return Ref(v.tv.AsInt64("id_untill_users"))
}

func (v Value_Table_untill_bill) NewIntent() Intent_WDoc_untill_bill {
	return Intent_WDoc_untill_bill{intent: exttinygo.NewValue(v.kb)}
}

type Intent_WDoc_untill_bill struct {
	intent exttinygo.TIntent
}

func (i Intent_WDoc_untill_bill) Set_close_datetime(v int64) Intent_WDoc_untill_bill {
	i.intent.PutInt64("close_datetime", v)
	return i
}

/*
TABLE articles INHERITS sys.CDoc (
	article_number int32,
	name varchar(255),
    ...
)
*/

type CDoc_untill_articles struct {
	Type
}

type Value_Table_untill_articles struct{ tv exttinygo.TValue }

type Intent_CDoc_untill_articles struct {
}

func (v Value_Table_untill_articles) Get_article_number() int32 {
	return v.tv.AsInt32("article_number")
}
func (v Value_Table_untill_articles) Name() string {
	return v.tv.AsString("name")
}

func (v CDoc_untill_articles) MustGetValue(id ID) Value_Table_untill_articles {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, v.qname)
	return Value_Table_untill_articles{tv: exttinygo.MustGetValue(kb)}
}

/*
TABLE pbill INHERITS sys.ODoc (
	id_bill int64 NOT NULL,
	id_untill_users ref(untill_users) NOT NULL,
	number int32,
	pbill_item pbill_item
)
*/

type ODoc_untill_pbill struct {
	Type
}

type Value_ODoc_untill_pbill struct{ tv exttinygo.TValue }

func (v Value_ODoc_untill_pbill) Get_id_bill() Ref {
	// !!! Note that Ref is returned rather than ID
	return Ref(v.tv.AsInt64("id_bill"))
}
func (v Value_ODoc_untill_pbill) Get_id_untill_users() Ref {
	return Ref(v.tv.AsInt64("id_untill_users"))
}

func (v Value_ODoc_untill_pbill) Get_number() Ref {
	return Ref(v.tv.AsInt64("id_number"))
}

func (v Value_ODoc_untill_pbill) Get_pbill_item() (res Container_ORecord_untill_pbill_item) {
	return Container_ORecord_untill_pbill_item{tv: v.tv.AsValue("pbill_item")}
}

type ORecord_untill_pbill_item struct {
	Type
}

type Value_ORecord_untill_pbill_item struct{ tv exttinygo.TValue }

func (v Value_ORecord_untill_pbill_item) Get_id_untill_users() Ref {
	return Ref(v.tv.AsInt64("id_untill_users"))
}

func (v Value_ORecord_untill_pbill_item) Get_tableno() int32 {
	return v.tv.AsInt32("tableno")
}

func (v Value_ORecord_untill_pbill_item) Get_rowbeg() int32 {
	return v.tv.AsInt32("rowbeg")
}

type Container_ORecord_untill_pbill_item struct {
	tv  exttinygo.TValue
	len int
}

// !!! Container Len() receiver is a pointer
func (v *Container_ORecord_untill_pbill_item) Len() int {
	if v.len == 0 {
		v.len = v.tv.Len() + 1
	}
	return v.len - 1
}

func (v *Container_ORecord_untill_pbill_item) Get(i int) Value_ORecord_untill_pbill_item {
	return Value_ORecord_untill_pbill_item{tv: v.tv.GetAsValue(i)}
}
