/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"
	"math"

	coreutils "github.com/untillpro/voedger/pkg/utils"
)

func NewFilter(data coreutils.MapObject) (IFilter, error) {
	args, ok := data["args"]
	if !ok {
		return nil, fmt.Errorf("filter: field 'args' must be present: %w", ErrNotFound)
	}
	expr, err := data.AsStringRequired("expr")
	if err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	}
	switch expr {
	case filterKind_Eq:
		return newEqualsFilter(args)
	case filterKind_NotEq:
		return newNotEqualsFilter(args)
	case filterKind_Gt:
		return newGreaterFilter(args)
	case filterKind_Lt:
		return newLessFilter(args)
	case filterKind_And:
		return newAndFilter(args)
	case filterKind_Or:
		return newOrFilter(args)
	default:
		return nil, fmt.Errorf("filter: expr: filter '%s' is unknown: %w", expr, ErrWrongType)
	}
}

func newEqualsFilter(args interface{}) (IFilter, error) {
	field, value, err := generalArgs(args)
	if err != nil {
		return nil, filterErr(filterKind_Eq, err)
	}
	epsilon, err := epsilon(args)
	if err != nil {
		return nil, filterErr(filterKind_Eq, err)
	}
	return &EqualsFilter{
		field:   field,
		value:   value,
		epsilon: epsilon,
	}, nil
}

func newNotEqualsFilter(args interface{}) (IFilter, error) {
	field, value, err := generalArgs(args)
	if err != nil {
		return nil, filterErr(filterKind_NotEq, err)
	}
	epsilon, err := epsilon(args)
	if err != nil {
		return nil, filterErr(filterKind_NotEq, err)
	}
	return &NotEqualsFilter{
		field:   field,
		value:   value,
		epsilon: epsilon,
	}, nil
}

func newGreaterFilter(args interface{}) (IFilter, error) {
	field, value, err := generalArgs(args)
	if err != nil {
		return nil, filterErr(filterKind_Gt, err)
	}
	return &GreaterFilter{
		field: field,
		value: value,
	}, nil
}

func newLessFilter(args interface{}) (IFilter, error) {
	field, value, err := generalArgs(args)
	if err != nil {
		return nil, filterErr(filterKind_Lt, err)
	}
	return &LessFilter{
		field: field,
		value: value,
	}, nil
}

func newAndFilter(args interface{}) (IFilter, error) {
	operands, ok := args.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'%s' filter: field 'args' must be an array: %w", filterKind_And, ErrWrongType)
	}
	andFilter := &AndFilter{make([]IFilter, len(operands))}
	for i, operandIntf := range operands {
		operand, ok := operandIntf.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("'%s' filter: each 'args' member must be an object: %w", filterKind_And, ErrWrongType)
		}
		filter, err := NewFilter(operand)
		if err != nil {
			return nil, filterErr(filterKind_And, err)
		}
		andFilter.filters[i] = filter
	}
	return andFilter, nil
}

func newOrFilter(args interface{}) (IFilter, error) {
	operands, ok := args.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'%s' filter: field 'args' must be an array: %w", filterKind_Or, ErrWrongType)
	}
	orFilter := &OrFilter{make([]IFilter, len(operands))}
	for i, operandIntf := range operands {
		operand, ok := operandIntf.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("'%s' filter: each 'args' member must be an object: %w", filterKind_Or, ErrWrongType)
		}
		filter, err := NewFilter(operand)
		if err != nil {
			return nil, filterErr(filterKind_Or, err)
		}
		orFilter.filters[i] = filter
	}
	return orFilter, nil
}

func filterErr(filterKind string, err error) error {
	return fmt.Errorf("'%s' filter: %w", filterKind, err)
}

func generalArgs(args interface{}) (string, interface{}, error) {
	data, ok := args.(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("field 'args' must be an object: %w", ErrWrongType)
	}
	mapObject := coreutils.MapObject(data)
	field, err := mapObject.AsStringRequired("field")
	if err != nil {
		return "", nil, err
	}
	value, ok := data["value"]
	if !ok {
		return "", nil, fmt.Errorf("field 'value' must be present: %w", ErrNotFound)
	}
	return field, value, nil
}

func epsilon(args interface{}) (float64, error) {
	data := args.(map[string]interface{}) // type is already checked by generalArgs()
	mapObject := coreutils.MapObject(data)
	options, _, err := mapObject.AsObject("options")
	if err != nil {
		return 0.0, err
	}
	epsilon, _, err := options.AsFloat64("epsilon")
	if err != nil {
		return 0.0, err
	}
	//TODO (FILTER0001) move it to filter prepare or validate
	//if epsilon == 0 {
	//	return 0, ErrJsonFieldNotFound
	//}
	return epsilon, nil
}

// reference https://floating-point-gui.de/errors/NearlyEqualsTest.java
func nearlyEqual(a, b, epsilon float64) bool {
	absA := math.Abs(a)
	absB := math.Abs(b)
	diff := math.Abs(a - b)
	if a == b {
		return true
	} else if a == 0 || b == 0 || diff < minNormalFloat64 {
		return diff < (epsilon * minNormalFloat64)
	} else {
		return diff/(absA+absB) < epsilon
	}
}

//TODO (FILTER0002) dynamic prepare and validation?
//type baseFilter struct {
//	field       string
//	value       interface{}
//	schemaField istructs.DataKindType
//}
//
//func (f *baseFilter) Prepare(schemaFields utils.SchemaFields) error {
//	schemaField, ok := schemaFields[f.field]
//	if !ok {
//		return fmt.Errorf(errLayout, f.field, ErrSchemaFieldNotFound)
//	}
//	f.schemaField = schemaField
//	return f.validateValue()
//}
//
//func (f *baseFilter) IsMatch(istructs.IObject) (bool, error) {
//	panic("implement me")
//}
//
//func (f *baseFilter) validateValue() error {
//	_, ok := f.value.(float64)
//	if (f.schemaField == istructs.DataKind_int32 || f.schemaField == istructs.DataKind_int64 || f.schemaField == istructs.DataKind_float32) && !ok {
//		return fmt.Errorf(errLayout, f.field, ErrWrongDataType)
//	}
//	return nil
//}
//TODO (FILTER0002)
