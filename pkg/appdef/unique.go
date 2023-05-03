/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

type unique struct {
	def    *def
	name   string
	fields []IField
}

func newUnique(def *def, name string, fields []string) *unique {
	u := unique{def, name, make([]IField, 0)}
	for _, f := range fields {
		fld := def.Field(f)
		if fld == nil {
			panic(fmt.Errorf("%v: can not create unique «%s»: field «%s» not found: %w", def.QName(), name, f, ErrNameNotFound))
		}
		u.fields = append(u.fields, fld)
	}
	return &u
}
