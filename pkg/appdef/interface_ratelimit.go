/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"time"
)

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

var DefaultRateScopes = []RateScope{RateScope_AppPartition}

type (
	RateCount  = uint
	RatePeriod = time.Duration
)

type IRate interface {
	IType
	Count() RateCount
	Period() RatePeriod
	Scopes() []RateScope
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

type ILimit interface {
	IType
	On() QNames
	Rate() IRate
}

type ILimitsBuilder interface {
	// Adds new Limit type with specified name.
	//
	// # Limited object names
	//
	// on which limit is applied, must be specified.
	// If these contain a function (command or query), this limits count of execution.
	// If these contain a structural (record or view record), this limits count of create/update operations.
	// Object names can contain `QNameANY` or one of `QNameAny×××` names.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists,
	//	 - if no limited objects names specified,
	//	 - if rate is not found.
	AddLimit(name QName, on []QName, rate QName, comment ...string)
}
