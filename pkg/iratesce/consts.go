/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package iratesce

import (
	"math"
	"time"
)

const (
	// Inf is the infinite rate limit; it allows all events (even if burst is zero).
	Inf = Limit(math.MaxFloat64)

	// InfDuration is the duration returned by Delay when a Reservation is not OK.
	InfDuration = time.Duration(1<<63 - 1)
)
