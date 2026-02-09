/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

const (
	maxRetriesOnCASErr      = 2
	maintainIntervalDivisor = 4
	retryIntervalDivisor    = 20
	preCASKillTimeFactor    = 0.75
	killDeadlineFactor      = 0.8
)
