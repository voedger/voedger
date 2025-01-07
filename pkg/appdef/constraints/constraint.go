/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package constraints

import (
	"math"
	"regexp"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
)

// Return new minimum length constraint for string or bytes data types.
func MinLen(v uint16, c ...string) appdef.IConstraint {
	return datas.NewDataConstraint(appdef.ConstraintKind_MinLen, v, c...)
}

// Return new maximum length restriction for string or bytes data types.
//
// Using MaxLen(), you can both limit the maximum length by a smaller value,
// and increase it to MaxFieldLength (65535).
//
// # Panics:
//   - if value is zero
func MaxLen(v uint16, c ...string) appdef.IConstraint {
	if v == 0 {
		panic(appdef.ErrOutOfBounds("maximum field length value is zero"))
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_MaxLen, v, c...)
}

// Return new pattern restriction for string or bytes data types.
//
// # Panics:
//   - if value is not valid regular expression
func Pattern(v string, c ...string) appdef.IConstraint {
	re, err := regexp.Compile(v)
	if err != nil {
		panic(err)
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_Pattern, re, c...)
}

// Return new minimum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is +infinite
func MinIncl(v float64, c ...string) appdef.IConstraint {
	if math.IsNaN(v) {
		panic(appdef.ErrInvalid("minimum inclusive value is NaN"))
	}
	if math.IsInf(v, 1) {
		panic(appdef.ErrOutOfBounds("minimum inclusive value is positive infinity"))
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_MinIncl, v, c...)
}

// Return new minimum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is +infinite
func MinExcl(v float64, c ...string) appdef.IConstraint {
	if math.IsNaN(v) {
		panic(appdef.ErrInvalid("minimum exclusive value is NaN"))
	}
	if math.IsInf(v, 1) {
		panic(appdef.ErrOutOfBounds("minimum inclusive value is positive infinity"))
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_MinExcl, v, c...)
}

// Return new maximum inclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxIncl(v float64, c ...string) appdef.IConstraint {
	if math.IsNaN(v) {
		panic(appdef.ErrInvalid("maximum inclusive value is NaN"))
	}
	if math.IsInf(v, -1) {
		panic(appdef.ErrOutOfBounds("maximum inclusive value is negative infinity"))
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_MaxIncl, v, c...)
}

// Return new maximum exclusive constraint for numeric data types.
//
// # Panics:
//   - if value is NaN
//   - if value is -infinite
func MaxExcl(v float64, c ...string) appdef.IConstraint {
	if math.IsNaN(v) {
		panic(appdef.ErrInvalid("maximum exclusive value is NaN"))
	}
	if math.IsInf(v, -1) {
		panic(appdef.ErrOutOfBounds("maximum exclusive value is negative infinity"))
	}
	return datas.NewDataConstraint(appdef.ConstraintKind_MaxExcl, v, c...)
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
func Enum[T enumerable](v ...T) appdef.IConstraint {
	l := len(v)
	if l == 0 {
		panic(appdef.ErrMissed("enumeration values slice (%T)", v))
	}
	c := make([]T, 0, l)
	for i := 0; i < l; i++ {
		n := v[i]
		c = append(c, n)
	}
	slices.Sort(c)
	c = slices.Compact(c)
	return datas.NewDataConstraint(appdef.ConstraintKind_Enum, c)
}

// Creates and returns new constraint.
//
// # Panics:
//   - if kind is unknown,
//   - id value is not compatible with kind.
func NewConstraint(kind appdef.ConstraintKind, value any, c ...string) appdef.IConstraint {
	switch kind {
	case appdef.ConstraintKind_MinLen:
		return MinLen(value.(uint16), c...)
	case appdef.ConstraintKind_MaxLen:
		return MaxLen(value.(uint16), c...)
	case appdef.ConstraintKind_Pattern:
		return Pattern(value.(string), c...)
	case appdef.ConstraintKind_MinIncl:
		return MinIncl(value.(float64), c...)
	case appdef.ConstraintKind_MinExcl:
		return MinExcl(value.(float64), c...)
	case appdef.ConstraintKind_MaxIncl:
		return MaxIncl(value.(float64), c...)
	case appdef.ConstraintKind_MaxExcl:
		return MaxExcl(value.(float64), c...)
	case appdef.ConstraintKind_Enum:
		var enum appdef.IConstraint
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
			panic(appdef.ErrUnsupported("enumeration type: %T", value))
		}
		if len(c) > 0 {
			if enum, ok := enum.(*datas.DataConstraint); ok {
				comments.SetComment(&enum.WithComments, c...)
			}
		}
		return enum
	}
	panic(appdef.ErrUnsupported("constraint kind: %v", kind))
}
