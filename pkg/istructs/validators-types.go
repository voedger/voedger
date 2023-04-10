/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import (
	"context"
)

type CUDValidator struct {
	// MatchFunc and MatchQNames are both considered
	// MatchFunc could be nil
	MatchFunc func(qName QName) bool
	// MatchQNames could be empty
	MatchQNames []QName
	Validate    func(ctx context.Context, appStructs IAppStructs, cudRow ICUDRow, wsid WSID, cmdQName QName) error
}

type EventValidator func(ctx context.Context, rawEvent IRawEvent, appStructs IAppStructs, wsid WSID) error
