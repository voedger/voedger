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
	Count  uint
	Period time.Duration
	Scopes []string `json:",omitempty"`
}

type Limit struct {
	Type
	On   []appdef.QName
	Rate appdef.QName
}
