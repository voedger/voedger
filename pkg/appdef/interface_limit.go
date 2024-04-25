/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type ILimit interface {
	IType
	On() QNames
	Rate() IRate
}

type IWithLimits interface {
	// Returns Limit by name.
	//
	// Returns nil if not found.
	Limit(QName) ILimit

	// Enumerates all limits
	//
	// Limits are enumerated in alphabetical order by QName
	Limits(func(ILimit))
}

type ILimitsBuilder interface {
	// Adds new Limit type with specified name.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists,
	//	 - if no rated objects names specified,
	//	 - if rate is not found.
	AddLimit(name QName, on []QName, rate QName, comment ...string)
}
