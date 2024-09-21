/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

type Translations map[string]map[language.Tag]string

func GetCatalogFromTranslations(t Translations) catalog.Catalog {
	ctlg := catalog.NewBuilder()
	for toBeTranslated, langTranslationMap := range t {
		for lang, translation := range langTranslationMap {
			if err := ctlg.SetString(lang, toBeTranslated, translation); err != nil {
				panic(err)
			}
		}
	}
	return ctlg
}
