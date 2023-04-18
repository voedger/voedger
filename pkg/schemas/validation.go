/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

func (cache *schemasCache) validateSchemas() (err error) {
	cache.prepare()

	validator := newValidator()
	cache.EnumSchemas(func(schema Schema) {
		err = errors.Join(err, validator.validate(schema))
	})
	return err
}

func (sch *schema) validate() (err error) {
	return errors.Join(
		sch.validateFields(),
		sch.validateContainers(),
	)
}

// Validates schema fields
func (sch *schema) validateFields() (err error) {
	sch.EnumFields(func(f Field) {
		if !f.IsSys() {
			if !sch.Props().DataKindAvailable(f.DataKind()) {
				err = errors.Join(err,
					fmt.Errorf("schema «%v»: field «%s» has unexpected type «%v»: %w", sch.QName(), f.Name(), f.DataKind(), ErrInvalidDataKind))
			}
		}
	})

	switch sch.Kind() {
	case istructs.SchemaKind_ViewRecord:
		errors.Join(err, sch.validateViewFields())
	case istructs.SchemaKind_ViewRecord_PartitionKey:
		errors.Join(err, sch.validateViewPartKeyFields())
	case istructs.SchemaKind_ViewRecord_ClusteringColumns:
		errors.Join(err, sch.validateViewClustKeyFields())
	}

	return err
}

// Validate view fields unique. See https://dev.heeus.io/launchpad/?r=1#!17003 for particulars
func (sch *schema) validateViewFields() (err error) {
	findSchema := func(contName string, kind SchemaKind) Schema {
		if cont := sch.Container(contName); cont != nil {
			if schema := sch.cache.SchemaByName(cont.Schema()); schema != nil {
				if schema.Kind() == kind {
					return schema
				}
			}
		}
		return nil
	}

	partSchema, clustSchema, valueSchema :=
		findSchema(istructs.SystemContainer_ViewPartitionKey, istructs.SchemaKind_ViewRecord_PartitionKey),
		findSchema(istructs.SystemContainer_ViewClusteringCols, istructs.SchemaKind_ViewRecord_ClusteringColumns),
		findSchema(istructs.SystemContainer_ViewValue, istructs.SchemaKind_ViewRecord_Value)
	if (partSchema == nil) || (clustSchema == nil) || (valueSchema == nil) {
		return nil // extended error will return later; see validateViewContainers() method
	}

	const errWrapFmt = "schema «%v»: view field «%s» unique violated in «%s» and in «%s»: %w"

	partSchema.EnumFields(func(f Field) {
		if clustSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewClusteringCols, ErrNameUniqueViolation))
		}
		if valueSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})
	clustSchema.EnumFields(func(f Field) {
		if valueSchema.Field(f.Name()) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), f.Name(), istructs.SystemContainer_ViewClusteringCols, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation))
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
	sch.EnumFields(func(fld Field) {
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
	case istructs.SchemaKind_ViewRecord:
		err = sch.validateViewContainers()
	default:
		sch.EnumContainers(func(c Container) {
			schema := sch.cache.SchemaByName(c.Schema())
			if schema != nil {
				if !sch.Props().ContainerKindAvailable(schema.Kind()) {
					err = errors.Join(err, fmt.Errorf("schema «%v» kind «%v»: container «%s» kind «%v» is not available: %w", sch.QName(), sch.Kind(), c.Name(), schema.Kind(), ErrInvalidSchemaKind))
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

	checkCont := func(name string, expectedKind SchemaKind) {
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
				fmt.Errorf("view schema «%v» container «%s» schema has invalid kind «%v», expected «%v»: %w", sch.QName(), name, contSchema.Kind(), expectedKind, ErrInvalidSchemaKind))
		}
	}

	checkCont(istructs.SystemContainer_ViewPartitionKey, istructs.SchemaKind_ViewRecord_PartitionKey)
	checkCont(istructs.SystemContainer_ViewClusteringCols, istructs.SchemaKind_ViewRecord_ClusteringColumns)
	checkCont(istructs.SystemContainer_ViewValue, istructs.SchemaKind_ViewRecord_Value)

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

	err := schema.validate()
	v.results[schema.QName()] = err

	// resolve externals
	schema.EnumContainers(func(cont Container) {
		if cont.Schema() == schema.QName() {
			return
		}
		contSchema := schema.Cache().SchemaByName(cont.Schema())
		if contSchema == nil {
			err = errors.Join(err, fmt.Errorf("schema «%v» container «%s» uses unknown schema «%v»: %w", schema.QName(), cont.Name(), cont.Schema(), ErrNameNotFound))
			v.results[schema.QName()] = err
		}
	})

	return err
}
