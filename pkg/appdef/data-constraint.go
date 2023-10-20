/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"regexp"
	"strings"
)

// Return new minimum length constraint for string or bytes data types.
//
// # Panics:
//   - if value is greater then MaxFieldLength (1024)
func DC_MinLen(v int, c ...string) IDataConstraint {
	if v > MaxFieldLength {
		panic(fmt.Errorf("minimum length value (%d) is too large, %d is maximum: %w", v, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return newDataConstraint(DataConstraintKind_MinLen, v, c...)
}

// Return new maximum length restriction for string or bytes data types.
//
// Using MaxLen(), you can both limit the minimum length by a smaller value, and increase it to MaxFieldLength (1024)
//
// # Panics:
//   - if value is zero
//   - if value is greater then MaxStringFieldLength (1024)
func DC_MaxLen(v uint16, c ...string) IDataConstraint {
	if v == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleRestricts))
	}
	if v > MaxFieldLength {
		panic(fmt.Errorf("maximum field length value (%d) is too large, %d is maximum: %w", v, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return newDataConstraint(DataConstraintKind_MaxLen, v, c...)
}

// Return new pattern restriction for string or bytes data types.
//
// # Panics:
//   - if value is not valid regular expression
func DC_Pattern(v string, c ...string) IFieldRestrict {
	re, err := regexp.Compile(v)
	if err != nil {
		panic(err)
	}
	return newDataConstraint(DataConstraintKind_Pattern, re, c...)
}

func (c DataTypeConstraints) String() string {
	if len(c) == 0 {
		return ""
	}

	s := make([]string, 0, len(c))
	for i := DataConstraintKind(1); i < DataConstraintKind_Count; i++ {
		if v, ok := c[i]; ok {
			s = append(s, fmt.Sprint(v))
		}
	}

	return strings.Join(s, ", ")
}

// # Implements:
//   - IDataConstraint
type dataConstraint struct {
	comment
	kind  DataConstraintKind
	value any
}

// Creates and returns new data constraint.
func newDataConstraint(k DataConstraintKind, v any, c ...string) IDataConstraint {
	return &dataConstraint{
		comment: makeComment(c...),
		kind:    k,
		value:   v,
	}
}

func (c *dataConstraint) Kind() DataConstraintKind {
	return c.kind
}

func (c *dataConstraint) Value() any {
	return c.value
}
