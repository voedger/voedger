/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package oldacl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
)

type ACElem struct {
	desc    string
	pattern PatternType
	policy  appdef.PolicyKind
}

type ACL []ACElem

type PatternType struct {
	opKindsPattern    []appdef.OperationKind
	principalsPattern [][]iauthnz.Principal // first dimension is OR, second is AND
	qNamesPattern     []appdef.QName
	fieldsPattern     [][]string // first dimension is OR, second is AND
}
