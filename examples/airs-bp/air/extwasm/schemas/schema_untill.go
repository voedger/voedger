/*
 * Copyright (c) 2024-present unTill Software Development Group B. V. 
 * @author Maxim Geraskin
 */

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
