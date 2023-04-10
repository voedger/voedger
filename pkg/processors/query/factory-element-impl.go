/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"
	"strings"

	coreutils "github.com/untillpro/voedger/pkg/utils"
)

func NewElement(data coreutils.MapObject) (IElement, error) {
	e := element{}
	path, _, err := data.AsString("path")
	if err != nil {
		return nil, fmt.Errorf("element: %w", err)
	}
	e.path = strings.Split(path, "/")
	if err := fillArray(data, "fields", func(elem interface{}) error {
		resultField, err := NewField(elem)
		if err == nil {
			e.fields = append(e.fields, resultField.(IResultField))
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("element: %w", err)
	}
	if err := fillArray(data, "refs", func(elem interface{}) error {
		refField, err := NewField(elem)
		if err == nil {
			e.refs = append(e.refs, refField.(IRefField))
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("element: %w", err)
	}
	return e, nil
}

func fillArray(data coreutils.MapObject, fieldName string, cb func(elem interface{}) error) error {
	elems, _, err := data.AsObjects(fieldName)
	for _, elem := range elems {
		if err = cb(elem); err != nil {
			break
		}
	}
	return err
}
