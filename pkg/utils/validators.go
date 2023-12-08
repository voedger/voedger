/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"golang.org/x/exp/slices"
)

func MatchQName(qNames ...appdef.QName) func(cud istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) bool {
	return func(cud istructs.ICUDRow, _ istructs.WSID, _ appdef.QName) bool {
		return slices.Contains(qNames, cud.QName())
	}
}

func TestMatchQNameFunc(matchFunc istructs.ValidatorMatchFunc, qNames ...appdef.QName) bool {
	for _, qName := range qNames {
		cudRow := &TestObject{Name: qName}
		if !matchFunc(cudRow, istructs.NullWSID, appdef.NullQName) {
			return false
		}
	}
	return len(qNames) > 0
}
