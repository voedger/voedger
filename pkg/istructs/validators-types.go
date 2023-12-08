/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
)

type CUDValidator struct {
	Match    ValidatorMatchFunc
	Validate func(ctx context.Context, appStructs IAppStructs, cudRow ICUDRow, wsid WSID, cmdQName appdef.QName) error
}

type EventValidator func(ctx context.Context, rawEvent IRawEvent, appStructs IAppStructs, wsid WSID) error

type ValidatorMatchFunc func(cud ICUDRow, wsid WSID, cmdQName appdef.QName) bool
