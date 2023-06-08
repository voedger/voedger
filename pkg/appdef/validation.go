/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "errors"

type withValidate interface {
	Validate() error
}

// Definitions validator
type validator struct {
}

func newValidator() *validator {
	return &validator{}
}

// validate specified definition
func (v *validator) validate(def IDef) (err error) {
	if val, ok := def.(withValidate); ok {
		err = val.Validate()
	}

	if _, ok := def.(IFields); ok {
		err = errors.Join(err, validateDefFields(def))
	}

	if _, ok := def.(IContainers); ok {
		err = errors.Join(err, validateDefContainers(def))
	}

	return err
}
