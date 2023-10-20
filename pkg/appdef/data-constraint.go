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
func DC_Pattern(v string, c ...string) IDataConstraint {
	re, err := regexp.Compile(v)
	if err != nil {
		panic(err)
	}
	return newDataConstraint(DataConstraintKind_Pattern, re, c...)
}

// # Implements:
//   - IDataConstraints
type dataConstraints struct {
	c map[DataConstraintKind]IDataConstraint
}

// Creates and returns new data constraints.
func makeDataConstraints() dataConstraints {
	return dataConstraints{
		c: make(map[DataConstraintKind]IDataConstraint),
	}
}

func (cc dataConstraints) Count() int {
	return len(cc.c)
}

func (cc dataConstraints) Constraint(kind DataConstraintKind) IDataConstraint {
	if c, ok := cc.c[kind]; ok {
		return c
	}
	return nil
}

func (cc dataConstraints) String() string {
	if len(cc.c) == 0 {
		return ""
	}

	s := make([]string, 0, len(cc.c))
	for i := DataConstraintKind(1); i < DataConstraintKind_Count; i++ {
		if c, ok := cc.c[i]; ok {
			s = append(s, fmt.Sprint(c))
		}
	}

	return strings.Join(s, ", ")
}

// Adds specified constraints. If constraint is already exists, it will be replaced.
//
// # Panics:
//   - if constraint is not supported by data.
func (cc dataConstraints) set(k DataKind, c ...IDataConstraint) {
	for _, c := range c {
		if ok := k.IsSupportedConstraint(c.Kind()); !ok {
			panic(fmt.Errorf("constraint %v is not compatible with %v: %w", c, k, ErrIncompatibleRestricts))
		}
		cc.c[c.Kind()] = c
	}
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

func (c dataConstraint) Kind() DataConstraintKind {
	return c.kind
}

func (c dataConstraint) Value() any {
	return c.value
}

func (c dataConstraint) String() string {
	switch c.kind {
	case DataConstraintKind_Pattern:
		return fmt.Sprintf("%s: `%v`", c.kind.TrimString(), c.value)
	default:
		return fmt.Sprintf("%s: %v", c.kind.TrimString(), c.value)
	}
}
