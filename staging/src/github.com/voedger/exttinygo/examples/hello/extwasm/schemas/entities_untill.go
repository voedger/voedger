package schemas

import exttinygo "github.com/voedger/exttinygo"

/*
TABLE articles INHERITS CDoc (
	article_number int32,
	name varchar(255),
    ...
)
*/

type Untill_articles struct {
	Entity
	// Do we need this for air development?
	Fields struct {
		Article_number string
		Name           string
	}
}

type Untill_articles_Value struct{ tv exttinygo.TValue }

func (v *Untill_articles_Value) Article_number() int32 {
	return v.tv.AsInt32("article_number")
}
func (v *Untill_articles_Value) Name() string {
	return v.tv.AsString("name")
}

func (v *Untill_articles) MustGetValue(id ID) Untill_articles_Value {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecords, Air.ProformaPrinted.QName)
	return Untill_articles_Value{tv: exttinygo.MustGetValue(kb)}
}