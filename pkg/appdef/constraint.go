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
	"slices"
)

// Return new minimum length constraint for string, bytes or raw data types.
func MinLen(v uint16, c ...string) IConstraint {
	return newDataConstraint(ConstraintKind_MinLen, v, c...)
}

// Return new maximum length restriction for string bytes or raw data types.
//
// Using MaxLen(), you can both limit the minimum length by a smaller value,
// and increase it to MaxFieldLength (1024) for string and bytes fields
// and to MaxRawFieldLength (65535) for raw data fields.
//
// # Panics:
//   - if value is zero
func MaxLen(v uint16, c ...string) IConstraint {
	if v == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleConstraints))
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
//   - if value is +infinite
func MinIncl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("minimum inclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, 1) {
		panic(fmt.Errorf("minimum inclusive value is positive infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MinIncl, v, c...)
}

// Return new minimum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is +infinite
func MinExcl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("minimum exclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, 1) {
		panic(fmt.Errorf("minimum inclusive value is positive infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MinExcl, v, c...)
}

// Return new maximum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxIncl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("maximum inclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, -1) {
		panic(fmt.Errorf("maximum inclusive value is negative infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MaxIncl, v, c...)
}

// Return new maximum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxExcl(v float64, c ...string) IConstraint {
	if math.IsNaN(v) {
		panic(fmt.Errorf("maximum exclusive value is NaN: %w", ErrIncompatibleConstraints))
	}
	if math.IsInf(v, -1) {
		panic(fmt.Errorf("maximum exclusive value is negative infinity: %w", ErrIncompatibleConstraints))
	}
	return newDataConstraint(ConstraintKind_MaxExcl, v, c...)
}

type enumerable interface {
	string | int32 | int64 | float32 | float64
}

// Return new enumeration constraint for char or numeric data types.
//
// Enumeration values must be one of the following types:
//   - string
//   - int32
//   - int64
//   - float32
//   - float64
//
// Passed values will be sorted and duplicates removed before placing
// into returning constraint.
//
// # Panics:
//   - if enumeration values list is empty
func Enum[T enumerable](v ...T) IConstraint {
	l := len(v)
	if l == 0 {
		panic(fmt.Errorf("enumeration values slice (%T) is empty: %w", v, ErrIncompatibleConstraints))
	}
	c := make([]T, 0, l)
	for i := 0; i < l; i++ {
		n := v[i]
		c = append(c, n)
	}
	slices.Sort(c)
	c = slices.Compact(c)
	return newDataConstraint(ConstraintKind_Enum, c)
}

// Creates and returns new constraint.
//
// # Panics:
//   - if kind is unknown,
//   - id value is not compatible with kind.
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
	case ConstraintKind_MinExcl:
		return MinExcl(value.(float64), c...)
	case ConstraintKind_MaxIncl:
		return MaxIncl(value.(float64), c...)
	case ConstraintKind_MaxExcl:
		return MaxExcl(value.(float64), c...)
	case ConstraintKind_Enum:
		var enum IConstraint
		switch v := value.(type) {
		case []string:
			enum = Enum(v...)
		case []int32:
			enum = Enum(v...)
		case []int64:
			enum = Enum(v...)
		case []float32:
			enum = Enum(v...)
		case []float64:
			enum = Enum(v...)
		default:
			panic(fmt.Errorf("unsupported enumeration type: %T", value))
		}
		if len(c) > 0 {
			enum.(ICommentBuilder).SetComment(c...)
		}
		return enum
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

func (c dataConstraint) String() (s string) {
	const (
		maxLen   = 64
		ellipsis = `…`
	)

	switch c.kind {
	case ConstraintKind_Pattern:
		s = fmt.Sprintf("%s: `%v`", c.kind.TrimString(), c.value)
	case ConstraintKind_Enum:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	default:
		s = fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	}
	if len(s) > maxLen {
		s = s[:maxLen-1] + ellipsis
	}
	return s
}
