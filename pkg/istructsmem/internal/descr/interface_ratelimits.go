/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package descr

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

type Rate struct {
	Type
	Count  appdef.RateCount
	Period time.Duration
	Scopes []string `json:",omitempty"`
}

type Limit struct {
	Type
	Ops    []string
	Filter LimitFilter
	Rate   appdef.QName
}

type LimitFilter struct {
	Option string // ALL or EACH
	Filter
}
