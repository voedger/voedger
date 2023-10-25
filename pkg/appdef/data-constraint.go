/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"math"
	"regexp"
)

// Return new minimum length constraint for string or bytes data types.
//
// # Panics:
//   - if value is greater then MaxFieldLength (1024)
func MinLen(v uint16, c ...string) IConstraint {
	if v > MaxFieldLength {
		panic(fmt.Errorf("minimum length value (%d) is too large, %d is maximum: %w", v, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return newDataConstraint(ConstraintKind_MinLen, v, c...)
}

// Return new maximum length restriction for string or bytes data types.
//
// Using MaxLen(), you can both limit the minimum length by a smaller value, and increase it to MaxFieldLength (1024)
//
// # Panics:
//   - if value is zero
//   - if value is greater then MaxStringFieldLength (1024)
func MaxLen(v uint16, c ...string) IConstraint {
	if v == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleConstraints))
	}
	if v > MaxFieldLength {
		panic(fmt.Errorf("maximum field length value (%d) is too large, %d is maximum: %w", v, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return newDataConstraint(ConstraintKind_MaxLen, v, c...)
}

// Return new pattern restriction for string or bytes data types.
//
// # Panics:
//   - if value is not valid regular expression
func Pattern(v string, c ...string) IConstraint {
	re, err := regexp.Compile(v)
	if err != nil {
		panic(err)
	}
	return newDataConstraint(ConstraintKind_Pattern, re, c...)
}

// Return new minimum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is infinite
func MinIncl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("minimum inclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, 0) {
		panic(fmt.Errorf("minimum inclusive value is infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MinIncl, v, c...)
}

// Creates and returns new constraint.
//
// # Panics:
//   - if kind is unknown
func NewConstraint(kind ConstraintKind, value any, c ...string) IConstraint {
	switch kind {
	case ConstraintKind_MinLen:
		return MinLen(value.(uint16), c...)
	case ConstraintKind_MaxLen:
		return MaxLen(value.(uint16), c...)
	case ConstraintKind_Pattern:
		return Pattern(value.(string), c...)
	case ConstraintKind_MinIncl:
		return MinIncl(value.(float64), c...)
	}
	panic(fmt.Errorf("unknown constraint kind: %v", kind))
}

// Returns the constraints for a data type, combined
// with those inherited from all ancestors of the type.
//
// Constraints are collected throughout the data types
// hierarchy, include all ancestors recursively.
// If any constraint is specified by the ancestor,
// but redefined in the descendant, then the constraint
// from the descendant only will included into result.
func ConstraintsWithInherited(data IData) map[ConstraintKind]IConstraint {
	cc := make(map[ConstraintKind]IConstraint)
	for d := data; d != nil; d = d.Ancestor() {
		d.Constraints(func(c IConstraint) {
			k := c.Kind()
			if _, ok := cc[k]; !ok {
				cc[k] = c
			}
		})
	}
	return cc
}

// # Implements:
//   - IDataConstraint
type dataConstraint struct {
	comment
	kind  ConstraintKind
	value any
}

// Creates and returns new data constraint.
func newDataConstraint(k ConstraintKind, v any, c ...string) IConstraint {
	return &dataConstraint{
		comment: makeComment(c...),
		kind:    k,
		value:   v,
	}
}

func (c dataConstraint) Kind() ConstraintKind {
	return c.kind
}

func (c dataConstraint) Value() any {
	return c.value
}

func (c dataConstraint) String() string {
	switch c.kind {
	case ConstraintKind_Pattern:
		return fmt.Sprintf("%s: `%v`", c.kind.TrimString(), c.value)
	default:
		return fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	}
}
