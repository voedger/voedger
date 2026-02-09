/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

const (
	// Maximum number of CompareAndSwap retry attempts on error before releasing leadership
	maxRetriesOnCASErr = 2

	// Leadership renewal interval = leadershipDuration / maintainIntervalDivisor
	maintainIntervalDivisor = 4

	// Delay between CompareAndSwap retry attempts = leadershipDuration / retryIntervalDivisor
	retryIntervalDivisor = 20

	// Killer deadline factor before attempting to CompareAndSwap: leadershipStart + leadershipDuration * preCASKillTimeFactor
	preCASKillTimeFactor = 0.75
	
	// Killer deadline factor after successful InsertIfNotExist and CompareAndSwap:
	// leadershipStart + leadershipDuration * killDeadlineFactor
	killDeadlineFactor = 0.8
)
