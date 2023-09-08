/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

// Checks data by field restricts. Return error if check failed
func checkRestricts(fld appdef.IField, data interface{}) error {
	switch fld.DataKind() {
	case appdef.DataKind_string:
		if f, ok := fld.(appdef.IStringField); ok {
			return checkStringRestricts(f, data.(string))
		}
	case appdef.DataKind_bytes:
		if f, ok := fld.(appdef.IBytesField); ok {
			return checkBytesRestricts(f, data.([]byte))
		}
	}
	return nil
}

// Checks string by field restricts. Return error if check failed
func checkStringRestricts(fld appdef.IStringField, data string) (err error) {
	l := uint16(len(data))
	if m := fld.Restricts().MinLen(); l < m {
		err = errors.Join(err, fmt.Errorf("string for field «%v» is too short (%d), at least %d characters required: %w", fld.Name(), l, m, ErrFieldValueRestricted))
	}
	if m := fld.Restricts().MaxLen(); l > m {
		err = errors.Join(err, fmt.Errorf("string for field «%v» is too long (%d), maximum %d characters allowed: %w", fld.Name(), l, m, ErrFieldValueRestricted))
	}
	if r := fld.Restricts().Pattern(); r != nil {
		if !r.MatchString(data) {
			err = errors.Join(err, fmt.Errorf("string for field «%v» does not match pattern `%v`: %w", fld.Name(), r, ErrFieldValueRestricted))
		}
	}
	return err
}

// Checks bytes by field restricts. Return error if check failed
func checkBytesRestricts(fld appdef.IStringField, data []byte) (err error) {
	l := uint16(len(data))
	if m := fld.Restricts().MinLen(); l < m {
		err = errors.Join(err, fmt.Errorf("bytes for field «%v» is too short (%d), at least %d bytes required: %w", fld.Name(), l, m, ErrFieldValueRestricted))
	}
	if m := fld.Restricts().MaxLen(); l > m {
		err = errors.Join(err, fmt.Errorf("bytes for field «%v» is too long (%d), maximum %d bytes allowed: %w", fld.Name(), l, m, ErrFieldValueRestricted))
	}
	if r := fld.Restricts().Pattern(); r != nil {
		if !r.Match(data) {
			err = errors.Join(err, fmt.Errorf("bytes for field «%v» does not match pattern `%v`: %w", fld.Name(), r, ErrFieldValueRestricted))
		}
	}
	return err
}
