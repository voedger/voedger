/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"iter"
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

	Scopes() iter.Seq[RateScope]
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

// Limit options enumeration
type LimitOption uint8

//go:generate stringer -type=LimitOption -output=stringer_limitoption.go

const (
	// Limit all objects matched by filter.
	// Single bucket for all objects.
	LimitOption_ALL LimitOption = iota

	// Limit each object matched by filter.
	// Separate bucket for each object.
	LimitOption_EACH

	LimitOption_count
)

type ILimit interface {
	IType

	Option() LimitOption

	// Returns limited resources filter.
	Filter() IFilter

	Rate() IRate
}

type ILimitsBuilder interface {
	// Adds new Limit for objects matched by filter.
	//
	// # Filtered objects to limit:
	// 	- If these contain a function (command or query), this limits count of execution.
	// 	- If these contain a structural (record or view record), this limits count of create/update operations.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists,
	//	 - if matched objects can not to be limited,
	//	 - if rate is not found.
	AddLimit(name QName, opt LimitOption, flt IFilter, rate QName, comment ...string)
}
