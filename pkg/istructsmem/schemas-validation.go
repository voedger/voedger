/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/untillpro/voedger/pkg/istructs"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

// ValidateSchemas: validates all schemas in the cache. Schemas used recursively multiple times are validated once.
// The method should be called once after filling the cache with all schemas.
func (cache *SchemasCacheType) ValidateSchemas() (err error) {
	validator := newValidator()
	for _, sch := range cache.schemas {
		if err = validator.validate(sch); err != nil {
			return err
		}
	}
	return nil
}

// Validate: validates the schema. If the resolveExternals parameter is set, it validates not only the schema itself, but also all other schemas used by it.
// The method should be called from the tests to check the validity of the schemas as they are created.
// For the final code, it is preferable to call the ValidateSchemas() method, which will check all the schemas in the cache.
func (sch *SchemaType) Validate(resolveExternals bool) (err error) {
	if sch.name == istructs.NullQName {
		return fmt.Errorf("schema name missed: %w", ErrNameMissed)
	}

	if ok, err := validQName(sch.name); !ok {
		return err
	}

	if resolveExternals {
		return newValidator().validate(sch)
	}

	if err = sch.validateFields(); err != nil {
		return err
	}
	if err = sch.validateContainers(); err != nil {
		return err
	}
	if err = sch.validateSingleton(); err != nil {
		return err
	}

	return nil
}

// Validate: validates view schemas
func (view *ViewSchemaType) Validate() (err error) {
	return view.Schema().Validate(true)
}

// validateFields: validates schema part: fields
func (sch *SchemaType) validateFields() (err error) {
	for n, fld := range sch.fields {
		if n == "" {
			return fmt.Errorf("schema «%v»: %w", sch.name, ErrNameMissed)
		}
		if sysField(n) {
			continue
		}
		if ok, err := validIdent(n); !ok {
			return fmt.Errorf("schema «%v» field name «%s» error: %w", sch.name, n, err)
		}
		if !availableFieldKind(sch.kind, fld.kind) {
			return fmt.Errorf("schema «%v»: field «%s» has unexpected type «%v»: %w", sch.name, n, fld.kind, coreutils.ErrFieldTypeMismatch)
		}
		if fld.verifiable {
			if len(fld.verify) == 0 {
				return fmt.Errorf("schema «%v»: verified field «%s» has no verification kind: %w", sch.name, n, ErrVerificationKindMissed)
			}
		}
	}

	switch sch.kind {
	case istructs.SchemaKind_ViewRecord:
		if err := sch.validateViewFields(); err != nil {
			return err
		}
	case istructs.SchemaKind_ViewRecord_PartitionKey:
		if err := sch.validateViewPartKeyFields(); err != nil {
			return err
		}
	case istructs.SchemaKind_ViewRecord_ClusteringColumns:
		if err := sch.validateViewClustKeyFields(); err != nil {
			return err
		}
	}

	return nil
}

// validateViewFields: validate view fields unique. See https://dev.heeus.io/launchpad/?r=1#!17003 for particulars
func (sch *SchemaType) validateViewFields() error {
	findSchema := func(contName string, kind istructs.SchemaKindType) *SchemaType {
		if cont, ok := sch.containers[contName]; ok {
			if schema := sch.cache.schemaByName(cont.schema); schema != nil {
				if schema.kind == kind {
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

	for n := range partSchema.fields {
		if _, ok := clustSchema.fields[n]; ok {
			return fmt.Errorf(errWrapFmt, sch.name, n, istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewClusteringCols, ErrNameUniqueViolation)
		}
		if _, ok := valueSchema.fields[n]; ok {
			return fmt.Errorf(errWrapFmt, sch.name, n, istructs.SystemContainer_ViewPartitionKey, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation)
		}
	}

	for n := range clustSchema.fields {
		if _, ok := valueSchema.fields[n]; ok {
			return fmt.Errorf(errWrapFmt, sch.name, n, istructs.SystemContainer_ViewClusteringCols, istructs.SystemContainer_ViewValue, ErrNameUniqueViolation)
		}
	}

	return nil
}

// validateViewPartKeyFields: validates view partition key schema fields
func (sch *SchemaType) validateViewPartKeyFields() error {
	if len(sch.fields) == 0 {
		return fmt.Errorf("schema «%v»: partition key can not to be empty: %w", sch.name, coreutils.ErrFieldsMissed)
	}

	// the validity of the field types (fixed width) was checked above in the method validateFields

	return nil
}

// validateViewClustKeyFields: validates view clustering columns key schema fields
func (sch *SchemaType) validateViewClustKeyFields() error {
	if len(sch.fields) == 0 {
		return fmt.Errorf("schema «%v»: clustering key can not to be empty: %w", sch.name, coreutils.ErrFieldsMissed)
	}

	for i, n := range sch.fieldsOrder {
		fld := sch.fields[n]
		if i < len(sch.fieldsOrder)-1 {
			if !fixedFldKind(fld.kind) {
				return fmt.Errorf("schema «%v»: only last field in clustering key can be variable length; not last field «%s» has variable length type «%v»: %w", sch.name, n, dataKindToStr[fld.kind], coreutils.ErrFieldTypeMismatch)
			}
		}
	}
	return nil
}

// validateContainers: validates schema part: containers
func (sch *SchemaType) validateContainers() (err error) {
	switch sch.kind {
	case istructs.SchemaKind_ViewRecord:
		if err := sch.validateViewContainers(); err != nil {
			return err
		}
	}

	if len(sch.containers) == 0 {
		return nil
	}
	if !availableContainers(sch.kind) {
		return fmt.Errorf("schema «%v»: containers in schema kind «%s» is deprecated: %w", sch.name, shemaKindToStr[sch.kind], ErrContainersUnavailable)
	}

	for n, cont := range sch.containers {
		err = cont.validate()
		if err != nil {
			return fmt.Errorf("schema «%v»: container «%s» is not valid: %w", sch.name, n, err)
		}
		contSchema := sch.cache.schemaByName(cont.schema)
		if contSchema != nil {
			if !availableContainerKind(sch.kind, contSchema.kind) {
				return fmt.Errorf("schema «%v»: container «%s» is not available schema kind «%s»: %w", sch.name, n, shemaKindToStr[contSchema.kind], ErrWrongSchemaStruct)
			}
		}
	}
	return nil
}

// validateSingleton: validates schema cdoc singleton
func (sch *SchemaType) validateSingleton() (err error) {
	if !sch.singleton.enabled {
		return nil
	}
	if sch.kind != istructs.SchemaKind_CDoc {
		return fmt.Errorf("schema «%v»: singleton available for «%s» schemas only, not for «%s»: %w", sch.name, shemaKindToStr[istructs.SchemaKind_CDoc], shemaKindToStr[sch.kind], ErrWrongSchemaStruct)
	}
	return nil
}

// validateViewContainers: validates view schema part: containers
func (sch *SchemaType) validateViewContainers() (err error) {
	const viewContCount = 3
	if len(sch.containers) != viewContCount {
		return fmt.Errorf("schema «%v»: view records schema must contain %d containers: %w", sch.name, viewContCount, ErrWrongSchemaStruct)
	}

	checkCont := func(name string, kind istructs.SchemaKindType) error {
		cont, ok := sch.containers[name]
		if !ok {
			return fmt.Errorf(errViewMissesContainerWrap, sch.name, name, ErrWrongSchemaStruct)
		}
		if cont.minOccurs != 1 {
			return fmt.Errorf("view schema «%v» container «%s» must have min occurs «1»: %w", sch.name, name, ErrWrongSchemaStruct)
		}
		if cont.maxOccurs != 1 {
			return fmt.Errorf("view schema «%v» container «%s» must have max occurs «1»: %w", sch.name, name, ErrWrongSchemaStruct)
		}
		contSchema := sch.cache.schemaByName(cont.schema)
		if contSchema == nil {
			return fmt.Errorf(errSchemaNotFoundWrap, cont.schema, ErrNameNotFound)
		}
		if contSchema.kind != kind {
			return fmt.Errorf("view schema «%v» container «%s» schema «%v» must be kind «%s», but has «%s»: %w", sch.name, name, cont.schema, shemaKindToStr[kind], shemaKindToStr[contSchema.kind], ErrUnexpectedShemaKind)
		}
		return nil
	}

	if err := checkCont(istructs.SystemContainer_ViewPartitionKey, istructs.SchemaKind_ViewRecord_PartitionKey); err != nil {
		return err
	}
	if err := checkCont(istructs.SystemContainer_ViewClusteringCols, istructs.SchemaKind_ViewRecord_ClusteringColumns); err != nil {
		return err
	}
	if err := checkCont(istructs.SystemContainer_ViewValue, istructs.SchemaKind_ViewRecord_Value); err != nil {
		return err
	}

	return nil
}

type (
	// validationResultType: type to store validation result for single scheme
	validationResultType struct {
		finished bool
		err      error
	}

	// validatorType: schema validator
	validatorType struct {
		results map[istructs.QName]*validationResultType
	}
)

func newValidator() *validatorType {
	return &validatorType{make(map[istructs.QName]*validationResultType)}
}

// validate: validate specified schema
func (v *validatorType) validate(schema *SchemaType) (err error) {
	res, ok := v.results[schema.name]
	if ok {
		if res.finished {
			return res.err
		}
		return nil
	}

	res = &validationResultType{}
	finish := func() { res.finished = true }
	defer finish()

	v.results[schema.name] = res
	res.err = schema.Validate(false)

	if res.err != nil {
		return res.err
	}

	// resolve externals
	for _, cont := range schema.containers {
		if cont.schema == schema.name {
			continue
		}
		contSchema, ok := schema.cache.schemas[cont.schema]
		if !ok {
			res.err = fmt.Errorf("schema «%v»: container «%s» used unknown schema «%v»: %w", schema.name, cont.name, cont.schema, ErrNameNotFound)
			break
		}

		err = v.validate(contSchema)
		if err != nil {
			res.err = fmt.Errorf("schema «%v»: container «%s» used not valid schema «%v»: %w", schema.name, cont.name, cont.schema, err)
			break
		}
	}

	return res.err
}

// validate: validate container occurses
func (cont *ContainerType) validate() (err error) {
	if cont.name == "" {
		return fmt.Errorf("empty container name: %w", ErrNameMissed)
	}
	if !sysContainer(cont.name) {
		if ok, err := validIdent(cont.name); !ok {
			return fmt.Errorf("container name «%s» error: %w", cont.name, err)
		}
	}
	if cont.schema == istructs.NullQName {
		return fmt.Errorf("empty container «%s» schema name: %w", cont.name, ErrNameMissed)
	}
	err = validateOccurs(cont.minOccurs, cont.maxOccurs)
	if err != nil {
		return fmt.Errorf("container «%s» occurs [%d…%d] failed: %w", cont.name, cont.minOccurs, cont.maxOccurs, err)
	}
	return nil
}
