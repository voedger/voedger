/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import exttinygo "github.com/voedger/exttinygo"

var Package_untill = struct {
	CDoc_articles CDoc_untill_articles
	ODoc_pbill    ODoc_untill_pbill
}{
	CDoc_articles: CDoc_untill_articles{
		Type: Type{qname: "untill.untill_articles"},
	},
}

/*
TABLE articles INHERITS CDoc (
	article_number int32,
	name varchar(255),
    ...
)
*/

type CDoc_untill_articles struct {
	Type
}

type Value_Table_untill_articles struct{ tv exttinygo.TValue }

func (v *Value_Table_untill_articles) Get_article_number() int32 {
	return v.tv.AsInt32("article_number")
}
func (v *Value_Table_untill_articles) Name() string {
	return v.tv.AsString("name")
}

func (v *CDoc_untill_articles) MustGetValue(id ID) Value_Table_untill_articles {
	kb := exttinygo.KeyBuilder(exttinygo.Record, Package_air.ODoc_ProformaPrinted.qname)
	return Value_Table_untill_articles{tv: exttinygo.MustGetValue(kb)}
}

/*
TABLE pbill INHERITS ODoc (
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

func (v *Value_ODoc_untill_pbill) Get_id_bill() ID {
	return ID(v.tv.AsInt64("id_bill"))
}
func (v *Value_ODoc_untill_pbill) Get_id_untill_users() ID {
	return ID(v.tv.AsInt64("id_untill_users"))
}

func (v *Value_ODoc_untill_pbill) Get_number() ID {
	return ID(v.tv.AsInt64("id_number"))
}

func (v *Value_ODoc_untill_pbill) Get_pbill_item() (res Collection_ORecord_untill_pbill_item) {
	return Collection_ORecord_untill_pbill_item{tv: v.tv.AsValue("pbill_item")}
}

type ORecord_untill_pbill_item struct {
	Type
}

type Value_ORecord_untill_pbill_item struct{ tv exttinygo.TValue }

func (v *Value_ORecord_untill_pbill_item) Get_id_untill_users() ID {
	return ID(v.tv.AsInt64("id_untill_users"))
}

func (v *Value_ORecord_untill_pbill_item) Get_tableno() int32 {
	return v.tv.AsInt32("tableno")
}

func (v *Value_ORecord_untill_pbill_item) Get_rowbeg() int32 {
	return v.tv.AsInt32("rowbeg")
}

type Collection_ORecord_untill_pbill_item struct {
	tv  exttinygo.TValue
	len int
}

func (v *Collection_ORecord_untill_pbill_item) Len() int {
	if v.len == 0 {
		v.len = v.tv.Len() + 1
	}
	return v.len - 1
}

func (v *Collection_ORecord_untill_pbill_item) Get(i int) Value_ORecord_untill_pbill_item {
	return Value_ORecord_untill_pbill_item{tv: v.tv.GetAsValue(i)}
}
