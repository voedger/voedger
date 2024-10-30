/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"
	"math"

	"github.com/voedger/voedger/pkg/coreutils"
)

var filterFactories = map[string]func(string, interface{}, coreutils.MapObject) (IFilter, error){
	filterKind_Eq:    newEqualsFilter,
	filterKind_NotEq: newNotEqualsFilter,
	filterKind_Gt:    newGreaterFilter,
	filterKind_Lt:    newLessFilter,
}

func NewFilter(data coreutils.MapObject) (IFilter, error) {
	argsIntf, ok := data["args"]
	if !ok {
		return nil, fmt.Errorf("filter: field 'args' must be present: %w", ErrNotFound)
	}
	filterKind, err := data.AsStringRequired("expr")
	if err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	}

	switch filterKind {
	case filterKind_Eq, filterKind_NotEq, filterKind_Gt, filterKind_Lt:
		filterFactory := filterFactories[filterKind]
		argsMI, ok := argsIntf.(map[string]interface{})
		if !ok {
			return nil, filterErr(filterKind, fmt.Errorf("field 'args' must be an object: %w", ErrWrongType))
		}
		args := coreutils.MapObject(argsMI)
		field, err := args.AsStringRequired("field")
		if err != nil {
			return nil, filterErr(filterKind, err)
		}
		value, ok := args["value"]
		if !ok {
			return nil, filterErr(filterKind, fmt.Errorf("field 'value' must be present: %w", ErrNotFound))
		}
		iFilter, err := filterFactory(field, value, args)
		if err != nil {
			err = filterErr(filterKind, err)
		}
		return iFilter, err
	case filterKind_And, filterKind_Or:
		operands, ok := argsIntf.([]interface{})
		if !ok {
			return nil, filterErr(filterKind, fmt.Errorf("field 'args' must be an array of objects: %w", ErrWrongType))
		}
		if filterKind == filterKind_And {
			return newAndFilter(operands)
		}
		return newOrFilter(operands)
	default:
		return nil, fmt.Errorf("filter: expr: filter '%s' is unknown: %w", filterKind, ErrWrongType)
	}
}

func newEqualsFilter(field string, value interface{}, args coreutils.MapObject) (IFilter, error) {
	epsilon, err := epsilon(args)
	if err != nil {
		return nil, err
	}
	return &EqualsFilter{
		field:   field,
		value:   value,
		epsilon: epsilon,
	}, nil
}

func newNotEqualsFilter(field string, value interface{}, args coreutils.MapObject) (IFilter, error) {
	epsilon, err := epsilon(args)
	if err != nil {
		return nil, err
	}
	return &NotEqualsFilter{
		field:   field,
		value:   value,
		epsilon: epsilon,
	}, nil
}

func newGreaterFilter(field string, value interface{}, _ coreutils.MapObject) (IFilter, error) {
	return &GreaterFilter{
		field: field,
		value: value,
	}, nil
}

func newLessFilter(field string, value interface{}, _ coreutils.MapObject) (IFilter, error) {
	return &LessFilter{
		field: field,
		value: value,
	}, nil
}

func newAndFilter(operands []interface{}) (IFilter, error) {
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

func newOrFilter(operands []interface{}) (IFilter, error) {
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

func epsilon(args coreutils.MapObject) (float64, error) {
	options, _, err := args.AsObject("options")
	if err != nil {
		return 0.0, err
	}
	epsilon, _, err := options.AsFloat64("epsilon")
	if err != nil {
		return 0.0, err
	}
	// TODO (FILTER0001) move it to filter prepare or validate
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
	}
	return diff/(absA+absB) < epsilon
}

//TODO (FILTER0002) dynamic prepare and validation?
//type baseFilter struct {
//	field       string
//	value       interface{}
//	kind        appdef.DataKind
//}
//
//func (f *baseFilter) Prepare(fd utils.FieldsDef) error {
//	kind, ok := fd[f.field]
//	if !ok {
//		return fmt.Errorf(errLayout, f.field, ErrNameNotFound)
//	}
//	f.kind = kind
//	return f.validateValue()
//}
//
//func (f *baseFilter) IsMatch(istructs.IObject) (bool, error) {
//	panic("implement me")
//}
//
//func (f *baseFilter) validateValue() error {
//	_, ok := f.value.(float64)
//	if (f.kind == appdef.DataKind_int32 || f.kind == appdef.DataKind_int64 || f.kind == appdef.DataKind_float32) && !ok {
//		return fmt.Errorf(errLayout, f.field, ErrWrongDataType)
//	}
//	return nil
//}
//TODO (FILTER0002)
