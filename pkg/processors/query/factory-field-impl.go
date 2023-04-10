/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import "fmt"

func NewField(data interface{}) (IField, error) {
	switch result := data.(type) {
	case string:
		return resultField{result}, nil
	case []interface{}:
		return newRefField(result)
	default:
		return nil, fmt.Errorf("must be a sting or an array of strings: %w", ErrWrongType)
	}
}

type resultField struct {
	field string
}

func (f resultField) Field() string { return f.field }

func newRefField(data []interface{}) (IField, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("field 'ref' parameters length must be 2 but got %d: %w", len(data), ErrWrongLength)
	}
	validate := func(intf interface{}) (string, error) {
		v, ok := intf.(string)
		if !ok {
			return "", fmt.Errorf("field 'ref' parameter must a string: %w", ErrWrongType)
		}
		return v, nil
	}
	field, err := validate(data[0])
	if err != nil {
		return nil, err
	}
	ref, err := validate(data[1])
	if err != nil {
		return nil, err
	}
	f := refField{
		field: field,
		ref:   ref,
	}
	f.key = fmt.Sprintf("%s/%s", f.field, f.ref)
	return f, nil
}

type refField struct {
	field string
	ref   string
	key   string
}

func (f refField) Field() string    { return f.field }
func (f refField) RefField() string { return f.ref }
func (f refField) Key() string      { return f.key }
