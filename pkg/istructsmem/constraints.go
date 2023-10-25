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
		err = checkCharsConstraints(fld, value.(string))
	case appdef.DataKind_bytes:
		err = checkCharsConstraints(fld, value.([]byte))
	case appdef.DataKind_int32:
		err = checkNumberConstraints(fld, value.(int32))
	case appdef.DataKind_int64:
		err = checkNumberConstraints(fld, value.(int64))
	case appdef.DataKind_float32:
		err = checkNumberConstraints(fld, value.(float32))
	case appdef.DataKind_float64:
		err = checkNumberConstraints(fld, value.(float64))
	}
	return err
}

// Checks string ot bytes value by field constraints. Return error if constraints violated
type chars interface{ string | []byte }

func checkCharsConstraints[T chars](fld appdef.IField, value T) (err error) {
	max := false

	fld.Constraints(func(c appdef.IConstraint) {
		switch c.Kind() {
		case appdef.ConstraintKind_MinLen:
			if len(value) < int(c.Value().(uint16)) {
				err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
			}
		case appdef.ConstraintKind_MaxLen:
			if len(value) > int(c.Value().(uint16)) {
				err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
			}
			max = true
		case appdef.ConstraintKind_Pattern:
			if pat := c.Value().(*regexp.Regexp); pat != nil {
				switch fld.DataKind() {
				case appdef.DataKind_string:
					if !pat.MatchString(string(value)) {
						err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
					}
				case appdef.DataKind_bytes:
					if !pat.Match([]byte(value)) {
						err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
					}
				}
			}
		}
	})

	if !max {
		if len(value) > int(appdef.DefaultFieldMaxLength) {
			err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, fmt.Sprintf("default MaxLen: %d", appdef.DefaultFieldMaxLength), ErrDataConstraintViolation))
		}
	}

	return err
}

// Checks value by number field constraints. Return error if constraints violated
type number = interface {
	int32 | int64 | float32 | float64
}

func checkNumberConstraints[T number](fld appdef.IField, value T) (err error) {
	fld.Constraints(func(c appdef.IConstraint) {
		switch c.Kind() {
		case appdef.ConstraintKind_MinIncl:
			if float64(value) < c.Value().(float64) {
				err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
			}
		case appdef.ConstraintKind_MinExcl:
			if float64(value) <= c.Value().(float64) {
				err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
			}
		}
	})

	return err
}
