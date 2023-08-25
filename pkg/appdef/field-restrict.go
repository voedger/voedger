/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"regexp"
	"strings"
)

type fieldRestrict uint8

const (
	fieldRestrict_MinLen fieldRestrict = iota
	fieldRestrict_MaxLen
	fieldRestrict_Pattern

	fieldRestrict_Count
)

func (r fieldRestrict) String() string {
	var fieldRestrict_String = [fieldRestrict_Count]string{"MinLen", "MaxLen", "Pattern"}
	if r < fieldRestrict_Count {
		return fieldRestrict_String[r]
	}
	return fmt.Sprintf("fieldRestrict(%d)", r)
}

// # Implements:
//   - IStringFieldRestrict
//   - IBytesFieldRestrict
type fieldRestrictData struct {
	fieldRestrict
	value any
}

// Return new minimum length restriction for string or bytes field
//
// # Panics:
//   - if value is greater then MaxFieldLength (1024)
func MinLen(value uint16) IFieldRestrict {
	if value > MaxFieldLength {
		panic(fmt.Errorf("minimum field length value (%d) is too large, %d is maximum: %w", value, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return &fieldRestrictData{fieldRestrict_MinLen, value}
}

// Default field max length.
//
// Used if MaxLen() restriction is not used.
//
// Using MaxLen(), you can both limit the minimum length by a smaller value, and increase it to MaxFieldLength (1024)
const DefaultFieldMaxLength = 255

// Return new maximum length restriction for string or bytes field
//
// # Panics:
//   - if value is zero
//   - if value is greater then MaxStringFieldLength (1024)
func MaxLen(value uint16) IFieldRestrict {
	if value == 0 {
		panic(fmt.Errorf("maximum field length value is zero: %w", ErrIncompatibleRestricts))
	}
	if value > MaxFieldLength {
		panic(fmt.Errorf("maximum field length value (%d) is too large, %d is maximum: %w", value, MaxFieldLength, ErrMaxFieldLengthExceeds))
	}
	return &fieldRestrictData{fieldRestrict_MaxLen, value}
}

// Return new pattern restriction for string field
//
// # Panics:
//   - if value is not valid regular expression
func Pattern(value string) IFieldRestrict {
	re, err := regexp.Compile(value)
	if err != nil {
		panic(err)
	}
	return &fieldRestrictData{fieldRestrict_Pattern, re}
}

// # Implements:
//   - IStringFieldRestricts & IBytesFieldRestricts
type fieldRestricts struct {
	f *field
	r map[fieldRestrict]any
}

func newFieldRestricts(f *field, r ...IFieldRestrict) *fieldRestricts {
	rest := &fieldRestricts{f, make(map[fieldRestrict]any)}
	rest.set(r...)
	return rest
}

func (r fieldRestricts) MinLen() uint16 {
	if v, ok := r.r[fieldRestrict_MinLen]; ok {
		return v.(uint16)
	}
	return 0
}

func (r fieldRestricts) MaxLen() uint16 {
	if v, ok := r.r[fieldRestrict_MaxLen]; ok {
		return v.(uint16)
	}
	return DefaultFieldMaxLength
}

func (r fieldRestricts) Pattern() *regexp.Regexp {
	if v, ok := r.r[fieldRestrict_Pattern]; ok {
		return v.(*regexp.Regexp)
	}
	return nil
}

func (r fieldRestricts) String() string {
	if len(r.r) == 0 {
		return ""
	}

	s := make([]string, 0, len(r.r))
	for i := fieldRestrict(0); i < fieldRestrict_Count; i++ {
		if v, ok := r.r[i]; ok {
			switch i {
			case fieldRestrict_Pattern:
				v = fmt.Sprintf("`%v`", v)
			}
			s = append(s, fmt.Sprintf("%v: %v", i, v))
		}
	}

	return strings.Join(s, ", ")
}

func (r fieldRestricts) checkCompatibles() {
	/* Uncomment when there are restricts for numeric fields

	for i := range r.r {
		switch i {
		case fieldRestrict_MinLen, fieldRestrict_MaxLen, fieldRestrict_Pattern:
			if k := r.f.DataKind(); (k != DataKind_string) && (k != DataKind_bytes) {
				panic(fmt.Errorf("%v restrict is not compatible with %s-field «%s»: %w", i, k.TrimString(), r.f.Name(), ErrIncompatibleRestricts))
			}
		}
	}

	*/

	if min, max := r.MinLen(), r.MaxLen(); min > max {
		panic(fmt.Errorf("min length (%d) is greater then max length (%d): %w", min, max, ErrIncompatibleRestricts))
	}
}

func (r *fieldRestricts) set(restricts ...IFieldRestrict) {
	for i := range restricts {
		if v, ok := restricts[i].(*fieldRestrictData); ok {
			r.r[v.fieldRestrict] = v.value
		}
	}
	r.checkCompatibles()
}
