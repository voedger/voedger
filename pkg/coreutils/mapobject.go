/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

type MapObject map[string]interface{}

func (m MapObject) AsString(name string) (val string, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return "", false, nil
	case string:
		return v, true, nil
	default:
		return "", true, fmt.Errorf(`field "%s" must be a string: %w`, name, ErrFieldTypeMismatch)
	}
}

func (m MapObject) AsStringRequired(name string) (val string, err error) {
	val, ok, err := m.AsString(name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf(`field "%s" missing: %w`, name, ErrFieldsMissed)
	}
	return val, nil
}

func (m MapObject) AsObject(name string) (val MapObject, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return nil, false, nil
	case map[string]interface{}:
		return MapObject(v), true, nil
	default:
		return nil, true, fmt.Errorf(`field "%s" must be an object: %w`, name, ErrFieldTypeMismatch)
	}
}

func (m MapObject) AsInt64(name string) (val int64, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return 0, false, nil
	case json.Number:
		int64Intf, err := ClarifyJSONNumber(v, appdef.DataKind_int64)
		if err != nil {
			return 0, false, err
		}
		return int64Intf.(int64), true, nil
	case int64:
		return v, true, nil
	default:
		return 0, true, fmt.Errorf(`field "%s" must be json.Number: %w`, name, ErrFieldTypeMismatch)
	}
}

func (m MapObject) AsObjects(name string) (val []interface{}, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return nil, false, nil
	case []interface{}:
		return v, true, nil
	default:
		return nil, true, fmt.Errorf(`field "%s" must be an array of objects: %w`, name, ErrFieldTypeMismatch)
	}
}

func (m MapObject) AsFloat64(name string) (val float64, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return 0, false, nil
	case json.Number:
		float64Intf, err := ClarifyJSONNumber(v, appdef.DataKind_float64)
		if err != nil {
			return 0, false, err
		}
		return float64Intf.(float64), true, nil
	default:
		return 0, true, fmt.Errorf(`field "%s" must be json.Number: %w`, name, ErrFieldTypeMismatch)
	}
}

func (m MapObject) AsBoolean(name string) (val bool, ok bool, err error) {
	switch v := m[name].(type) {
	case nil:
		return false, false, nil
	case bool:
		return v, true, nil
	default:
		return false, true, fmt.Errorf(`field "%s" must be a boolean: %w`, name, ErrFieldTypeMismatch)
	}
}
