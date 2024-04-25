/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "time"

// Rate scopes enumeration
type RateScope uint8

//go:generate stringer -type=RateScope -output=stringer_ratescope.go
const (
	RateScope_null RateScope = iota

	RateScope_AppPartition
	RateScope_Workspace
	RateScope_User
	RateScope_IP

	RateScope_count
)

type RateScopes []RateScope

var DefaultRateScopes = RateScopes{RateScope_AppPartition}

type (
	RateCount  = uint
	RatePeriod = time.Duration
)

type IRate interface {
	IType
	Count() RateCount
	Period() RatePeriod
	Scopes() RateScopes
}

type IWithRates interface {
	// Returns Rate by name.
	//
	// Returns nil if not found.
	Rate(QName) IRate

	// Enumerates all rates
	//
	// Rates are enumerated in alphabetical order by QName
	Rates(func(IRate))
}

type IRatesBuilder interface {
	// Adds new Rate type with specified name.
	//
	// If no scope is specified, DefaultRateScopes is used.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists,
	//   - if count is zero,
	//   - if period is zero.
	AddRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string)
}
