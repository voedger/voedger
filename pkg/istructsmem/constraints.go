/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/voedger/voedger/pkg/appdef"
)

// Checks value by field constraints. Return error if constraints violated
func checkConstraints(fld appdef.IField, value interface{}) (err error) {
	switch fld.DataKind() {
	case appdef.DataKind_string:
		err = checkStringConstraints(fld, value)
	case appdef.DataKind_bytes:
		err = checkBytesConstraints(fld, value)
	}
	return err
}

// Checks value by string field constraints. Return error if constraints violated
func checkStringConstraints(fld appdef.IField, value interface{}) (err error) {
	min := uint16(0)
	max := appdef.DefaultFieldMaxLength
	var pat *regexp.Regexp

	fld.Constraints(func(c appdef.IConstraint) {
		switch c.Kind() {
		case appdef.ConstraintKind_MinLen:
			min = c.Value().(uint16)
		case appdef.ConstraintKind_MaxLen:
			max = c.Value().(uint16)
		case appdef.ConstraintKind_Pattern:
			pat = c.Value().(*regexp.Regexp)
		}
	})

	v := value.(string)
	l := len(v)
	if l < int(min) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("MinLen: %d", min), ErrDataConstraintViolation))
	}

	if l > int(max) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("MaxLen: %d", max), ErrDataConstraintViolation))
	}

	if (pat != nil) && (!pat.MatchString(v)) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("Pattern: %v", pat), ErrDataConstraintViolation))
	}

	return err
}

// Checks value by bytes field constraints. Return error if constraints violated
func checkBytesConstraints(fld appdef.IField, value interface{}) (err error) {
	min := uint16(0)
	max := appdef.DefaultFieldMaxLength
	var pat *regexp.Regexp

	fld.Constraints(func(c appdef.IConstraint) {
		switch c.Kind() {
		case appdef.ConstraintKind_MinLen:
			min = c.Value().(uint16)
		case appdef.ConstraintKind_MaxLen:
			max = c.Value().(uint16)
		case appdef.ConstraintKind_Pattern:
			pat = c.Value().(*regexp.Regexp)
		}
	})

	v := value.([]byte)
	l := len(v)
	if l < int(min) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("MinLen: %d", min), ErrDataConstraintViolation))
	}

	if l > int(max) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("MaxLen: %d", max), ErrDataConstraintViolation))
	}

	if (pat != nil) && (!pat.Match(v)) {
		err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("Pattern: %v", pat), ErrDataConstraintViolation))
	}

	return err
}
