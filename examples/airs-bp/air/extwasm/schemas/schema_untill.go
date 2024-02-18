package schemas

var Untill = struct {
	Articles Untill_articles
}{
	Articles: Untill_articles{
		Entity: Entity{QName: "untill.untill_articles"},
		Fields: struct {
			Article_number string
			Name           string
		}{
			Article_number: "article_number",
			Name:           "name",
		},
	},
}
