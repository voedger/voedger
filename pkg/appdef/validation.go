/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// Validate a schema entities
func (sch *schema) Validate() (err error) {
	return errors.Join(
		sch.validateFields(),
		sch.validateContainers(),
	)
}

// Validates schema fields
func (sch *schema) validateFields() (err error) {
	sch.Fields(func(f Field) {
		if !f.IsSys() {
			if !sch.Kind().DataKindAvailable(f.DataKind()) {
				err = errors.Join(err,
					fmt.Errorf("schema «%v»: field «%s» has unexpected type «%v»: %w", sch.QName(), f.Name(), f.DataKind(), ErrInvalidDataKind))
			}
		}
	})

	switch sch.Kind() {
	case DefKind_ViewRecord:
		err = errors.Join(err, sch.validateViewFields())
	case DefKind_ViewRecord_PartitionKey:
		err = errors.Join(err, sch.validateViewPartKeyFields())
	case DefKind_ViewRecord_ClusteringColumns:
		err = errors.Join(err, sch.validateViewClustKeyFields())
	}

	return err
}

// Validate view fields unique. See https://dev.heeus.io/launchpad/?r=1#!17003 for particulars
func (sch *schema) validateViewFields() (err error) {
	findSchema := func(contName string, kind DefKind) Schema {
		if cont := sch.Container(contName); cont != nil {
			if schema := sch.app.SchemaByName(cont.Schema()); schema != nil {
				if schema.Kind() == kind {
					return schema
				}
			}
		}
		return nil
	}

	partSchema, clustSchema, valueSchema :=
		findSchema(SystemContainer_ViewPartitionKey, DefKind_ViewRecord_PartitionKey),
		findSchema(SystemContainer_ViewClusteringCols, DefKind_ViewRecord_ClusteringColumns),
		findSchema(SystemContainer_ViewValue, DefKind_ViewRecord_Value)
	if (partSchema == nil) || (clustSchema == nil) || (valueSchema == nil) {
		return nil // extended error will return later; see validateViewContainers() method
	}

	const errWrapFmt = "schema «%v»: view field «%s» unique violated in «%s» and in «%s»: %w"

	partSchema.Fields(func(f Field) {
		if clustSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), SystemContainer_ViewPartitionKey, SystemContainer_ViewClusteringCols, ErrNameUniqueViolation))
		}
		if valueSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), SystemContainer_ViewPartitionKey, SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})
	clustSchema.Fields(func(f Field) {
		if valueSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), SystemContainer_ViewClusteringCols, SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})

	return err
}

// Validates view partition key schema fields
func (sch *schema) validateViewPartKeyFields() error {
	if sch.FieldCount() == 0 {
		return fmt.Errorf("schema «%v»: partition key can not to be empty: %w", sch.QName(), ErrFieldsMissed)
	}

	// the validity of the field types (fixed width) was checked above in the method validateFields

	return nil
}

// Validates view clustering columns schema fields
func (sch *schema) validateViewClustKeyFields() (err error) {
	if sch.FieldCount() == 0 {
		return fmt.Errorf("schema «%v»: clustering columns can not to be empty: %w", sch.QName(), ErrFieldsMissed)
	}

	idx, cnt := 0, sch.FieldCount()
	sch.Fields(func(fld Field) {
		idx++
		if idx == cnt {
			return // last field may be any kind
		}
		if !fld.IsFixedWidth() {
			err = errors.Join(err,
				fmt.Errorf("schema «%v»: only last clustering column field can be variable length; not last field «%s» has variable length type «%v»: %w", sch.QName(), fld.Name(), fld.DataKind(), ErrInvalidDataKind))
		}
	})

	return err
}

// Validates schema containers
func (sch *schema) validateContainers() (err error) {
	switch sch.Kind() {
	case DefKind_ViewRecord:
		err = sch.validateViewContainers()
	default:
		sch.Containers(func(c Container) {
			schema := sch.app.SchemaByName(c.Schema())
			if schema != nil {
				if !sch.Kind().ContainerKindAvailable(schema.Kind()) {
					err = errors.Join(err, fmt.Errorf("schema «%v» kind «%v»: container «%s» kind «%v» is not available: %w", sch.QName(), sch.Kind(), c.Name(), schema.Kind(), ErrInvalidDefKind))
				}
			}
		})
	}

	return err
}

// Validates view schema containers
func (sch *schema) validateViewContainers() (err error) {
	const viewContCount = 3
	if sch.ContainerCount() != viewContCount {
		err = errors.Join(err,
			fmt.Errorf("schema «%v»: view records schema must contain 3 containers: %w", sch.QName(), ErrWrongSchemaStruct))
	}

	checkCont := func(name string, expectedKind DefKind) {
		cont := sch.Container(name)
		if cont == nil {
			err = errors.Join(err,
				fmt.Errorf("view schema «%v» misses container «%s»: %w", sch.QName(), name, ErrWrongSchemaStruct))
			return
		}
		if o := cont.MinOccurs(); o != 1 {
			err = errors.Join(err,
				fmt.Errorf("view schema «%v» container «%s» has invalid min occurs value %d, expected 1: %w", sch.QName(), name, o, ErrWrongSchemaStruct))
		}
		if o := cont.MaxOccurs(); o != 1 {
			err = errors.Join(err,
				fmt.Errorf("view schema «%v» container «%s» has invalid max occurs value %d, expected 1: %w", sch.QName(), name, o, ErrWrongSchemaStruct))
		}
		contSchema := sch.ContainerSchema(name)
		if contSchema == nil {
			err = errors.Join(err,
				fmt.Errorf("view schema «%v» container «%s» schema not found: %w", sch.QName(), name, ErrNameNotFound))
			return
		}
		if contSchema.Kind() != expectedKind {
			err = errors.Join(err,
				fmt.Errorf("view schema «%v» container «%s» schema has invalid kind «%v», expected «%v»: %w", sch.QName(), name, contSchema.Kind(), expectedKind, ErrInvalidDefKind))
		}
	}

	checkCont(SystemContainer_ViewPartitionKey, DefKind_ViewRecord_PartitionKey)
	checkCont(SystemContainer_ViewClusteringCols, DefKind_ViewRecord_ClusteringColumns)
	checkCont(SystemContainer_ViewValue, DefKind_ViewRecord_Value)

	return err
}

type (
	// schema validator
	validator struct {
		results map[QName]error
	}
)

func newValidator() *validator {
	return &validator{make(map[QName]error)}
}

// validate specified schema
func (v *validator) validate(schema Schema) error {
	if err, ok := v.results[schema.QName()]; ok {
		return err
	}

	err := schema.Validate()
	v.results[schema.QName()] = err

	// resolve externals
	schema.Containers(func(cont Container) {
		if cont.Schema() == schema.QName() {
			return
		}
		contSchema := schema.App().SchemaByName(cont.Schema())
		if contSchema == nil {
			err = errors.Join(err, fmt.Errorf("schema «%v» container «%s» uses unknown schema «%v»: %w", schema.QName(), cont.Name(), cont.Schema(), ErrNameNotFound))
			v.results[schema.QName()] = err
		}
	})

	return err
}
