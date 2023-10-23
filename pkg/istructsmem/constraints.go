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
	fld.Constraints(func(c appdef.IConstraint) {
		switch c.Kind() {
		case appdef.ConstraintKind_MinLen:
			min := c.Value().(uint16)
			switch fld.DataKind() {
			case appdef.DataKind_string:
				if l := len(value.(string)); l < int(min) {
					// string-field «f1» data constraint «MinLen: 1» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			case appdef.DataKind_bytes:
				if l := len(value.([]byte)); l < int(min) {
					// bytes-field «f1» data constraint «MinLen: 1» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			}
		case appdef.ConstraintKind_MaxLen:
			max := c.Value().(uint16)
			switch fld.DataKind() {
			case appdef.DataKind_string:
				if l := len(value.(string)); l > int(max) {
					// string-field «f1» data constraint «MaxLen: 10» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			case appdef.DataKind_bytes:
				if l := len(value.([]byte)); l > int(max) {
					// bytes-field «f1» data constraint «MaxLen: 10» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			}
		case appdef.ConstraintKind_Pattern:
			r := c.Value().(*regexp.Regexp)
			switch fld.DataKind() {
			case appdef.DataKind_string:
				if s := value.(string); !r.MatchString(s) {
					// string-field «f1» data constraint «Pattern: `^/w+$`» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			case appdef.DataKind_bytes:
				if b := value.([]byte); !r.Match(b) {
					// byte-field «f1» data constraint «Pattern: `^/w+$`» violated
					err = errors.Join(err, fmt.Errorf(errFieldDataConstraintViolatedFmt, fld, c, ErrDataConstraintViolation))
				}
			}
		}
	})
	return err
}
