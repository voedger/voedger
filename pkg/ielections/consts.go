/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

const (
	// leadership is renewed every 1/4 of leadership duration
	renewalsPerLeadershipDur = 4

	// log every renewal for the first N ticks (early confirmation that maintenance is working)
	maintainLogFirstTicks = 10

	// log a heartbeat every N ticks after the initial period
	maintainLogEveryTicks = 200
)
