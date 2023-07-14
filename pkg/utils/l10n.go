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
	for key, langTranslationMap := range t {
		for lang, translation := range langTranslationMap {
			if err := ctlg.SetString(lang, key, translation); err != nil {
				return nil
			}
		}
	}
	return ctlg
}
