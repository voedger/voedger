/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "time"

type RateScope int8

const (
	RateScope_nul RateScope = iota
	RateScope_AppPartition
	RateScope_Workspace
	RateScope_User
	RateScope_IP
	RateScope_count
)

type RateScopes []RateScope

type IRate interface {
	IType
	Count() int
	Period() time.Duration
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
	// # Panics:
	//   - if name is empty or invalid,
	//   - if Rate with the same name already exists,
	//   - if count is less than 1,
	//   - if period is zero or negative.
	AddRate(name QName, count int, period time.Duration, scopes []RateScope, comment ...string) IRatesBuilder
}
