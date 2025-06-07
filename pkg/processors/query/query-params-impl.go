/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
)

type queryParams struct {
	elements  []IElement
	filters   []IFilter
	orderBy   []IOrderBy
	startFrom int64
	count     int64
}

func (p queryParams) Elements() []IElement { return p.elements }
func (p queryParams) Filters() []IFilter   { return p.filters }
func (p queryParams) OrderBy() []IOrderBy  { return p.orderBy }
func (p queryParams) StartFrom() int64     { return p.startFrom }
func (p queryParams) Count() int64         { return p.count }

func newQueryParams(data coreutils.MapObject, elementFactory ElementFactory, filterFactory FilterFactory, orderByFactory OrderByFactory, rootFieldsKinds FieldsKinds,
	rootType appdef.IType) (res IQueryParams, err error) {
	qp := queryParams{}
	if err = qp.fillArray(data, "elements", func(elem coreutils.MapObject) error {
		element, err := elementFactory(elem)
		if err == nil {
			qp.elements = append(qp.elements, element)
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("elements: %w", err)
	}
	if err = qp.fillArray(data, "filters", func(elem coreutils.MapObject) error {
		filter, err := filterFactory(elem)
		if err == nil {
			qp.filters = append(qp.filters, filter)
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("filters: %w", err)
	}
	if err = qp.fillArray(data, "orderBy", func(elem coreutils.MapObject) error {
		orderBy, err := orderByFactory(elem)
		if err == nil {
			qp.orderBy = append(qp.orderBy, orderBy)
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("orderBy: %w", err)
	}
	if qp.count, _, err = data.AsInt64("count"); err != nil {
		return nil, err
	}
	if qp.startFrom, _, err = data.AsInt64("startFrom"); err != nil {
		return nil, err
	}
	return qp, qp.validate(rootFieldsKinds, rootType)
}

func (p *queryParams) fillArray(data coreutils.MapObject, fieldName string, cb func(elem coreutils.MapObject) error) error {
	elems, _, err := data.AsObjects(fieldName)
	for _, elemIntf := range elems {
		elem, ok := elemIntf.(map[string]interface{})
		if !ok {
			return fmt.Errorf("each member must be an object: %w", ErrWrongType)
		}
		if err = cb(elem); err != nil {
			break
		}
	}
	return err
}

func (p queryParams) validate(rootFieldsKinds FieldsKinds, rootType appdef.IType) (err error) {
	pathPresent := make(map[string]bool)
	for _, e := range p.elements {
		if pathPresent[e.Path().Name()] {
			return fmt.Errorf("elements: path '%s' must be unique", e.Path().Name())
		}
		pathPresent[e.Path().Name()] = true
	}

	fields := make(map[string]bool)
	for _, e := range p.elements {
		path := e.Path().AsArray()
		if e.Path().IsRoot() {
			// root
			for _, field := range e.ResultFields() {
				if _, ok := rootFieldsKinds[field.Field()]; !ok {
					return fmt.Errorf("elements: root element fields has field '%s' that is unexpected in root fields, please remove it: %w", field.Field(), ErrUnexpected)
				}
				fields[field.Field()] = true
			}
			for _, field := range e.RefFields() {
				fields[field.Key()] = true
			}
			continue
		}
		// nested
		currentType := rootType
		var deepestContainer appdef.IContainer
		for _, nestedName := range path {
			currentContainers, ok := currentType.(appdef.IWithContainers)
			if !ok {
				return fmt.Errorf("elements: table %s has no nested tables but %s nested table is queried: %w", currentType.QName(), nestedName, ErrUnexpected)
			}
			nestedContainer := currentContainers.Container(nestedName)
			if nestedContainer == nil {
				return fmt.Errorf("elements: unknown nested table %s: %w", nestedName, ErrUnexpected)
			}
			currentType = nestedContainer.Type()
			deepestContainer = nestedContainer
		}
		deepestContainerFields := deepestContainer.Type().(appdef.IWithFields)
		for _, field := range e.ResultFields() {
			iField := deepestContainerFields.Field(field.Field())
			if iField == nil {
				return fmt.Errorf("elements: nested element fields has field '%s' that is unexpected among fields of %s, please remove it: %w", field.Field(), deepestContainer.QName(), ErrUnexpected)
			}
			fields[field.Field()] = true
		}
		for _, field := range e.RefFields() {
			fields[field.Key()] = true
		}
	}
	validateFilter := func(filter, field string) (err error) {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("'%s' filter has field '%s' that is absent in root element fields/refs, please add or change it: %w", filter, field, ErrUnexpected)
		}
		return nil
	}
	err = validateFilters(p.filters, validateFilter)
	if err != nil {
		return fmt.Errorf("filters: %w", err)
	}
	return p.validateOrderBy(fields)
}

func validateFilters(filters []IFilter, validateFilter func(filter, field string) (err error)) (err error) {
	for _, f := range filters {
		if err != nil {
			return
		}
		switch filter := f.(type) {
		case *EqualsFilter:
			err = validateFilter(filterKind_Eq, filter.field)
		case *NotEqualsFilter:
			err = validateFilter(filterKind_NotEq, filter.field)
		case *GreaterFilter:
			err = validateFilter(filterKind_Gt, filter.field)
		case *LessFilter:
			err = validateFilter(filterKind_Lt, filter.field)
		case *AndFilter:
			err = validateFilters(filter.filters, validateFilter)
			if err != nil {
				err = fmt.Errorf("'%s' filter: %w", filterKind_And, err)
			}
		case *OrFilter:
			err = validateFilters(filter.filters, validateFilter)
			if err != nil {
				err = fmt.Errorf("'%s' filter: %w", filterKind_Or, err)
			}
		}
	}
	return err
}

func (p queryParams) validateOrderBy(fields map[string]bool) (err error) {
	for _, o := range p.orderBy {
		if _, ok := fields[o.Field()]; !ok {
			return fmt.Errorf("orderBy has field '%s' that is absent in root element fields/refs, please add or change it: %w", o.Field(), ErrUnexpected)
		}
	}
	return nil
}
