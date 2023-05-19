/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"github.com/voedger/voedger/pkg/istructs"
)

type filter struct {
	acceptAll bool
	fields    map[string]bool
}

func (f *filter) filter(field string) bool {
	if f.acceptAll {
		return true
	}
	_, ok := f.fields[field]
	return ok
}

type keyPart struct {
	name  string
	value []byte
}

type result struct {
	istructs.NullObject
	value string
}

func (o *result) AsString(string) string { return o.value }
