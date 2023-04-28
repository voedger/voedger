/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// Validate a definition entities
func (d *def) Validate() (err error) {
	return errors.Join(
		d.validateFields(),
		d.validateContainers(),
	)
}

// Validates definition fields
func (d *def) validateFields() (err error) {
	d.Fields(func(f Field) {
		if !f.IsSys() {
			if !d.Kind().DataKindAvailable(f.DataKind()) {
				err = errors.Join(err,
					fmt.Errorf("%v: field «%s» has unexpected type «%v»: %w", d.QName(), f.Name(), f.DataKind(), ErrInvalidDataKind))
			}
		}
	})

	switch d.Kind() {
	case DefKind_ViewRecord:
		err = errors.Join(err, d.validateViewFields())
	case DefKind_ViewRecord_PartitionKey:
		err = errors.Join(err, d.validateViewPartKeyFields())
	case DefKind_ViewRecord_ClusteringColumns:
		err = errors.Join(err, d.validateViewClustKeyFields())
	}

	return err
}

// Validate view fields unique. See https://dev.heeus.io/launchpad/?r=1#!17003 for particulars
func (d *def) validateViewFields() (err error) {
	findDef := func(contName string, kind DefKind) IDef {
		if cont := d.Container(contName); cont != nil {
			if def := d.app.DefByName(cont.Def()); def != nil {
				if def.Kind() == kind {
					return def
				}
			}
		}
		return nil
	}

	pkDef, ccDef, valDef :=
		findDef(SystemContainer_ViewPartitionKey, DefKind_ViewRecord_PartitionKey),
		findDef(SystemContainer_ViewClusteringCols, DefKind_ViewRecord_ClusteringColumns),
		findDef(SystemContainer_ViewValue, DefKind_ViewRecord_Value)
	if (pkDef == nil) || (ccDef == nil) || (valDef == nil) {
		return nil // extended error will return later; see validateViewContainers() method
	}

	const errWrapFmt = "definition «%v»: view field «%s» unique violated in «%s» and in «%s»: %w"

	pkDef.Fields(func(f Field) {
		if ccDef.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, d.QName(), f.Name(), SystemContainer_ViewPartitionKey, SystemContainer_ViewClusteringCols, ErrNameUniqueViolation))
		}
		if valDef.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, d.QName(), f.Name(), SystemContainer_ViewPartitionKey, SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})
	ccDef.Fields(func(f Field) {
		if valDef.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, d.QName(), f.Name(), SystemContainer_ViewClusteringCols, SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})

	return err
}

// Validates view partition key definition fields
func (d *def) validateViewPartKeyFields() error {
	if d.FieldCount() == 0 {
		return fmt.Errorf("%v: partition key can not to be empty: %w", d.QName(), ErrFieldsMissed)
	}

	// the validity of the field types (fixed width) was checked above in the method validateFields

	return nil
}

// Validates view clustering columns definition fields
func (d *def) validateViewClustKeyFields() (err error) {
	if d.FieldCount() == 0 {
		return fmt.Errorf("%v: clustering columns can not to be empty: %w", d.QName(), ErrFieldsMissed)
	}

	idx, cnt := 0, d.FieldCount()
	d.Fields(func(fld Field) {
		idx++
		if idx == cnt {
			return // last field may be any kind
		}
		if !fld.IsFixedWidth() {
			err = errors.Join(err,
				fmt.Errorf("%v: only last clustering column field can be variable length; not last field «%s» has variable length type «%v»: %w", d.QName(), fld.Name(), fld.DataKind(), ErrInvalidDataKind))
		}
	})

	return err
}

// Validates definition containers
func (d *def) validateContainers() (err error) {
	switch d.Kind() {
	case DefKind_ViewRecord:
		err = d.validateViewContainers()
	default:
		d.Containers(func(c Container) {
			def := d.app.DefByName(c.Def())
			if def != nil {
				if !d.Kind().ContainerKindAvailable(def.Kind()) {
					err = errors.Join(err, fmt.Errorf("%v: kind «%v»: container «%s» kind «%v» is not available: %w", d.QName(), d.Kind(), c.Name(), def.Kind(), ErrInvalidDefKind))
				}
			}
		})
	}

	return err
}

// Validates view definition containers
func (d *def) validateViewContainers() (err error) {
	const viewContCount = 3
	if d.ContainerCount() != viewContCount {
		err = errors.Join(err,
			fmt.Errorf("%v: view records definition must have 3 containers: %w", d.QName(), ErrWrongDefStruct))
	}

	checkCont := func(name string, expectedKind DefKind) {
		cont := d.Container(name)
		if cont == nil {
			err = errors.Join(err,
				fmt.Errorf("%v: view definition misses container «%s»: %w", d.QName(), name, ErrWrongDefStruct))
			return
		}
		if o := cont.MinOccurs(); o != 1 {
			err = errors.Join(err,
				fmt.Errorf("%v: view container «%s» has invalid min occurs value %d, expected 1: %w", d.QName(), name, o, ErrWrongDefStruct))
		}
		if o := cont.MaxOccurs(); o != 1 {
			err = errors.Join(err,
				fmt.Errorf("%v: view container «%s» has invalid max occurs value %d, expected 1: %w", d.QName(), name, o, ErrWrongDefStruct))
		}
		contDef := d.ContainerDef(name)
		if contDef == nil {
			err = errors.Join(err,
				fmt.Errorf("%v: view container «%s» definition not found: %w", d.QName(), name, ErrNameNotFound))
			return
		}
		if contDef.Kind() != expectedKind {
			err = errors.Join(err,
				fmt.Errorf("%v: view container «%s» definition has invalid kind «%v», expected «%v»: %w", d.QName(), name, contDef.Kind(), expectedKind, ErrInvalidDefKind))
		}
	}

	checkCont(SystemContainer_ViewPartitionKey, DefKind_ViewRecord_PartitionKey)
	checkCont(SystemContainer_ViewClusteringCols, DefKind_ViewRecord_ClusteringColumns)
	checkCont(SystemContainer_ViewValue, DefKind_ViewRecord_Value)

	return err
}

type (
	// Definitions validator
	validator struct {
		results map[QName]error
	}
)

func newValidator() *validator {
	return &validator{make(map[QName]error)}
}

// validate specified definition
func (v *validator) validate(def IDef) error {
	if err, ok := v.results[def.QName()]; ok {
		return err
	}

	err := def.Validate()
	v.results[def.QName()] = err

	// resolve externals
	def.Containers(func(cont Container) {
		if cont.Def() == def.QName() {
			return
		}
		contDef := def.App().DefByName(cont.Def())
		if contDef == nil {
			err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown definition «%v»: %w", def.QName(), cont.Name(), cont.Def(), ErrNameNotFound))
			v.results[def.QName()] = err
		}
	})

	return err
}
