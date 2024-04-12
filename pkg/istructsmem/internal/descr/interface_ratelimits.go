/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package descr

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

type RateLimit struct {
	Kind                  istructs.RateLimitKind
	Period                time.Duration
	MaxAllowedPerDuration uint32
}
