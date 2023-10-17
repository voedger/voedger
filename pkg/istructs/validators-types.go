/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
)

type CUDValidator struct {
	// MatchFunc and MatchQNames are both considered
	// MatchFunc could be nil
	MatchFunc func(qName appdef.QName, wsid WSID, cmdQName appdef.QName) bool
	// MatchQNames could be empty
	MatchQNames []appdef.QName
	Validate    func(ctx context.Context, appStructs IAppStructs, cudRow ICUDRow, wsid WSID, cmdQName appdef.QName) error
}

type EventValidator func(ctx context.Context, rawEvent IRawEvent, appStructs IAppStructs, wsid WSID) error
