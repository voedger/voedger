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
}

func newValidator() *validator {
	return &validator{}
}

// validate specified definition
func (v *validator) validate(def IDef) (err error) {
	if val, ok := def.(withValidate); ok {
		err = val.Validate()
	}

	if fld, ok := def.(IFields); ok {
		// resolve reference fields definitions
		fld.RefFields(func(rf IRefField) {
			for _, n := range rf.Refs() {
				refDef := def.App().DefByName(n)
				if refDef == nil {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to unknown definition «%v»: %w", def.QName(), rf.Name(), n, ErrNameNotFound))
					continue
				}
				if !refDef.Kind().HasSystemField(SystemField_ID) {
					err = errors.Join(err, fmt.Errorf("%v: reference field «%s» refs to non referable definition «%v» kind «%v» without «%s» field: %w", def.QName(), rf.Name(), n, refDef.Kind(), SystemField_ID, ErrInvalidDefKind))
					continue
				}
			}
		})
	}

	if cnt, ok := def.(IContainers); ok {
		// resolve containers definitions
		cnt.Containers(func(cont IContainer) {
			if cont.Def() == def.QName() {
				return
			}
			contDef := def.App().DefByName(cont.Def())
			if contDef == nil {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown definition «%v»: %w", def.QName(), cont.Name(), cont.Def(), ErrNameNotFound))
			}
		})
	}

	return err
}
