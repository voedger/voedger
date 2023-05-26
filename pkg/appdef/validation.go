/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

type withValidate interface {
	Validate() error
}

// Definitions validator
type validator struct {
	results map[QName]error
}

func newValidator() *validator {
	return &validator{make(map[QName]error)}
}

// validate specified definition
func (v *validator) validate(def IDef) (err error) {
	if err, ok := v.results[def.QName()]; ok {
		// notest
		return err
	}

	if d, ok := def.(withValidate); ok {
		err = d.Validate()
		v.results[def.QName()] = err
	}

	if d, ok := def.(IContainers); ok {
		// resolve externals
		d.Containers(func(cont IContainer) {
			if cont.Def() == def.QName() {
				return
			}
			contDef := def.App().DefByName(cont.Def())
			if contDef == nil {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown definition «%v»: %w", def.QName(), cont.Name(), cont.Def(), ErrNameNotFound))
				v.results[def.QName()] = err
			}
		})
	}

	return err
}
