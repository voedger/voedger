/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"regexp"
)

type fieldRestrict uint8

const (
	fieldRestrict_MinLen fieldRestrict = iota + 1
	fieldRestrict_MaxLen
	fieldRestrict_Pattern
)

// # Implements:
//   - IStringFieldRestrict
type fieldRestrictData struct {
	fieldRestrict
	value any
}

// Return new minimum length restriction for string field
//
// # Panics:
//   - if value is greater then MaxStringFieldLength (1024)
func MinLen(value uint16) IStringFieldRestrict {
	if value > MaxStringFieldLength {
		panic(fmt.Errorf("minimum field length value (%d) is too large, %d is maximum: %w", value, MaxStringFieldLength, ErrMaxFieldLengthExceeds))
	}
	return &fieldRestrictData{fieldRestrict_MinLen, &value}
}

// Default string field max length.
//
// Used if MaxLen() restriction is not used.
//
// Using MaxLen (), you can both limit the minimum length by a smaller value, and increase it to MaxStringFieldLength (1024)
const DefaultStringFieldMaxLength = 255

// Return new maximum length restriction for string field
//
// # Panics:
//   - if value is zero
//   - if value is greater then MaxStringFieldLength (1024)
func MaxLen(value uint16) IStringFieldRestrict {
	if value == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleRestricts))
	}
	if value > MaxStringFieldLength {
		panic(fmt.Errorf("maximum field length value (%d) is too large, %d is maximum: %w", value, MaxStringFieldLength, ErrMaxFieldLengthExceeds))
	}
	return &fieldRestrictData{fieldRestrict_MaxLen, &value}
}

// Return new pattern restriction for string field
//
// # Panics:
//   - if value is not valid regular expression
func Pattern(value string) IStringFieldRestrict {
	re, err := regexp.Compile(value)
	if err != nil {
		panic(err)
	}
	return &fieldRestrictData{fieldRestrict_Pattern, re}
}

// # Implements:
//   - IStringFieldRestricts
type fieldRestricts map[fieldRestrict]any

func newFieldRestricts(r ...IStringFieldRestrict) *fieldRestricts {
	f := &fieldRestricts{}
	f.set(r...)
	return f
}

func (r *fieldRestricts) MinLen() uint16 {
	if v, ok := (*r)[fieldRestrict_MinLen]; ok {
		v := v.(*uint16)
		return *v
	}
	return 0
}

func (r *fieldRestricts) MaxLen() uint16 {
	if v, ok := (*r)[fieldRestrict_MaxLen]; ok {
		v := v.(*uint16)
		return *v
	}
	return DefaultStringFieldMaxLength
}

func (r *fieldRestricts) Pattern() *regexp.Regexp {
	if v, ok := (*r)[fieldRestrict_Pattern]; ok {
		return v.(*regexp.Regexp)
	}
	return nil
}

func (r *fieldRestricts) checkCompatibles() {
	if min, max := r.MinLen(), r.MaxLen(); min > max {
		panic(fmt.Errorf("min length (%d) is greater then max length (%d): %w", min, max, ErrIncompatibleRestricts))
	}
}

func (r *fieldRestricts) set(restricts ...IStringFieldRestrict) {
	for i := range restricts {
		if v, ok := restricts[i].(*fieldRestrictData); ok {
			(*r)[v.fieldRestrict] = v.value
		}
	}
	r.checkCompatibles()
}
