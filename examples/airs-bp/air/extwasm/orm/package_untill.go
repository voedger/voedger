/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import exttinygo "github.com/voedger/exttinygo"

var Package_untill = struct {
	CDoc_articles Table_untill_articles
}{
	CDoc_articles: Table_untill_articles{
		Type: Type{qname: "untill.untill_articles"},
		Fields: struct {
			Article_number string
			Name           string
		}{
			Article_number: "article_number",
			Name:           "name",
		},
	},
}

/*
TABLE articles INHERITS CDoc (
	article_number int32,
	name varchar(255),
    ...
)
*/

type Table_untill_articles struct {
	Type
	// Do we need this for air development?
	Fields struct {
		Article_number string
		Name           string
	}
}

type Value_Table_untill_articles struct{ tv exttinygo.TValue }

func (v *Value_Table_untill_articles) Get_article_number() int32 {
	return v.tv.AsInt32("article_number")
}
func (v *Value_Table_untill_articles) Name() string {
	return v.tv.AsString("name")
}

func (v *Table_untill_articles) MustGetValue(id ID) Value_Table_untill_articles {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecords, Package_air.ODoc_ProformaPrinted.qname)
	return Value_Table_untill_articles{tv: exttinygo.MustGetValue(kb)}
}
