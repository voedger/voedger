/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"errors"
	"fmt"

	"github.com/untillpro/voedger/pkg/istructs"
)

// Validates all schemas in the cache. Schemas used recursively multiple times are validated once.
// The method should be called once after filling the cache with all schemas.
func (cache *SchemasCache) ValidateSchemas() (err error) {
	cache.Prepare()

	validator := newValidator()
	cache.EnumSchemas(func(schema *Schema) {
		err = errors.Join(err, validator.validate(schema))
	})
	return err
}

// Validates the schema. The method should be called from the tests to check the validity of the schemas as they are created.
//
// In most cases, it is preferable to call the ValidateSchemas() method, which will check all the schemas in the cache.
func (sch *Schema) Validate() (err error) {
	return errors.Join(
		sch.validateFields(),
		sch.validateContainers(),
	)
}

// validateFields: validates schema part: fields
func (sch *Schema) validateFields() (err error) {
	sch.Fields(func(fieldName string, kind DataKind) {
		if !IsSysField(fieldName) {
			if !sch.Props().DataKindAvailable(kind) {
				err = errors.Join(err, fmt.Errorf("schema «%v»: field «%s» has unexpected type «%v»: %w", sch.QName(), fieldName, kind, ErrInvalidDataKind))
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
func (sch *Schema) validateViewFields() (err error) {
	findSchema := func(contName string, kind SchemaKind) *Schema {
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

	partSchema.Fields(func(n string, _ DataKind) {
		if clustSchema.Field(n) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), n, istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewClusteringCols, ErrNameUniqueViolation))
		}
		if valueSchema.Field(n) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), n, istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})
	clustSchema.Fields(func(n string, _ DataKind) {
		if valueSchema.Field(n) != nil {
			err = errors.Join(err, fmt.Errorf(errWrapFmt, sch.QName(), n, istructs.SystemContainer_ViewClusteringCols, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation))
		}
	})

	return err
}

// Validates view partition key schema fields
func (sch *Schema) validateViewPartKeyFields() error {
	if sch.FieldCount() == 0 {
		return fmt.Errorf("schema «%v»: partition key can not to be empty: %w", sch.QName(), ErrFieldsMissed)
	}

	// the validity of the field types (fixed width) was checked above in the method validateFields

	return nil
}

// Validates view clustering columns schema fields
func (sch *Schema) validateViewClustKeyFields() error {
	if sch.FieldCount() == 0 {
		return fmt.Errorf("schema «%v»: clustering columns can not to be empty: %w", sch.QName(), ErrFieldsMissed)
	}

	for i := 0; i < sch.FieldCount(); i++ {
		if i < sch.FieldCount()-1 {
			fld := sch.FieldAt(i)
			if !fld.IsFixedWidth() {
				return fmt.Errorf("schema «%v»: only last clustering column field can be variable length; not last field «%s» has variable length type «%v»: %w", sch.QName(), fld.Name(), fld.DataKind(), ErrInvalidDataKind)
			}
		}
	}
	return nil
}

// Validates schema containers
func (sch *Schema) validateContainers() (err error) {
	switch sch.Kind() {
	case istructs.SchemaKind_ViewRecord:
		err = sch.validateViewContainers()
	default:
		sch.Containers(func(name string, schemaName QName) {
			schema := sch.cache.SchemaByName(schemaName)
			if schema != nil {
				if !sch.Props().ContainerKindAvailable(schema.Kind()) {
					err = errors.Join(err, fmt.Errorf("schema «%v» kind «%v»: container «%s» kind «%v» is not available: %w", sch.QName(), sch.Kind(), name, schema.Kind(), ErrInvalidSchemaKind))
				}
			}
		})
	}

	return err
}

// Validates view schema containers
func (sch *Schema) validateViewContainers() (err error) {
	const viewContCount = 3
	if sch.ContainerCount() != viewContCount {
		err = errors.Join(err, fmt.Errorf("schema «%v»: view records schema must contain 3 containers: %w", sch.QName(), ErrWrongSchemaStruct))
	}

	checkCont := func(name string, expectedKind SchemaKind) {
		cont := sch.Container(name)
		if cont == nil {
			err = errors.Join(err, fmt.Errorf("view schema «%v» misses container «%s»: %w", sch.QName(), name, ErrWrongSchemaStruct))
			return
		}
		if o := cont.MinOccurs(); o != 1 {
			err = errors.Join(err, fmt.Errorf("view schema «%v» container «%s» has invalid min occurs value %d, expected 1: %w", sch.QName(), name, o, ErrWrongSchemaStruct))
		}
		if o := cont.MaxOccurs(); o != 1 {
			err = errors.Join(err, fmt.Errorf("view schema «%v» container «%s» has invalid max occurs value %d, expected 1: %w", sch.QName(), name, o, ErrWrongSchemaStruct))
		}
		contSchema := sch.ContainerSchema(name)
		if contSchema == nil {
			err = errors.Join(err, fmt.Errorf("view schema «%v» container «%s» schema not found: %w", sch.QName(), name, ErrNameNotFound))
			return
		}
		if contSchema.Kind() != expectedKind {
			err = errors.Join(err, fmt.Errorf("view schema «%v» container «%s» schema has invalid kind «%v», expected «%v»: %w", sch.QName(), name, contSchema.Kind(), expectedKind, ErrInvalidSchemaKind))
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
func (v *validator) validate(schema *Schema) error {
	if err, ok := v.results[schema.QName()]; ok {
		return err
	}

	err := schema.Validate()
	v.results[schema.QName()] = err

	// resolve externals
	schema.Containers(func(containerName string, schemaName QName) {
		if schemaName == schema.QName() {
			return
		}
		contSchema := schema.cache.SchemaByName(schemaName)
		if contSchema == nil {
			err = errors.Join(err, fmt.Errorf("schema «%v» container «%s» uses unknown schema «%v»: %w", schema.QName(), containerName, schemaName, ErrNameNotFound))
			v.results[schema.QName()] = err
		}
	})

	return err
}
