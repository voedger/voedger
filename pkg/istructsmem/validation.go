/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

// TODO: move to internal/validate package

// validator provides validation application structures by single schema
type validator struct {
	validators *validators
	schema     schemas.Schema
}

func newValidator(validators *validators, schema schemas.Schema) *validator {
	return &validator{validators, schema}
}

// Return readable name of entity to validate.
// If entity has only type QName, then the result will be short like `CDoc (sales.BillDocument)`, otherwise it will be complete like `CRecord «Price» (sales.PriceRecord)`
func (v *validator) entName(e interface{}) string {
	ent := shemaKindToStr[v.schema.Kind()]
	name := ""
	typeName := v.schema.QName()

	if row, ok := e.(istructs.IRowReader); ok {
		if qName := row.AsQName(schemas.SystemField_QName); qName != schemas.NullQName {
			typeName = qName
			if (qName == v.schema.QName()) && v.schema.Kind().HasSystemField(schemas.SystemField_Container) {
				if cont := row.AsString(schemas.SystemField_Container); cont != "" {
					name = cont
				}
			}
		}
	}

	if name == "" {
		return fmt.Sprintf("%s (%v)", ent, typeName) // short form
	}

	return fmt.Sprintf("%s «%s» (%v)", ent, name, typeName) // complete form
}

// validate specified document
func (v *validator) validDocument(doc *elementType) error {
	// TODO: check RecordID refs available for document kind
	return v.validElement(doc, true)
}

// validate specified element
func (v *validator) validElement(el *elementType, storable bool) (err error) {
	if storable {
		err = v.validRecord(&el.recordType, true)
	} else {
		if e := v.validRow(&el.recordType.rowType); e != nil {
			err = fmt.Errorf("%s has not valid row data: %w", v.entName(el), e)
		}
	}

	err = errors.Join(err,
		v.validElementContainers(el, storable))

	return err
}

// validates element containers
func (v *validator) validElementContainers(el *elementType, storable bool) (err error) {

	err = v.validElementContOccurses(el)

	elID := el.ID()

	idx := -1
	el.EnumElements(
		func(child *elementType) {
			idx++
			childName := child.Container()
			if childName == "" {
				err = errors.Join(err,
					validateErrorf(ECode_EmptyElementName, "%s child[%d] has empty container name: %w", v.entName(el), idx, ErrNameMissed))
				return
			}
			cont := v.schema.Container(childName)
			if cont == nil {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidElementName, "%s child[%d] has unknown container name «%s»: %w", v.entName(el), idx, childName, ErrNameNotFound))
				return
			}

			childQName := child.QName()
			if childQName != cont.Schema() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidSchemaName, "%s child[%d] «%s» has wrong schema name «%v», expected «%v»: %w", v.entName(el), idx, childName, childQName, cont.Schema(), ErrNameNotFound))
				return
			}

			if storable {
				parID := child.Parent()
				if parID == istructs.NullRecordID {
					child.setParent(elID) // if child parentID omitted, then restore it
				} else {
					if parID != elID {
						err = errors.Join(err,
							validateErrorf(ECode_InvalidRefRecordID, "%s child[%d] «%s (%v)» has wrong parent id «%d», expected «%d»: %w", v.entName(el), idx, childName, childQName, elID, parID, ErrWrongRecordID))
					}
				}
			}

			childValidator := v.validators.validator(childQName)
			if childValidator == nil {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidSchemaName, "object refers to unknown schema «%v»: %w", childQName, ErrNameNotFound))
				return
			}

			err = errors.Join(err,
				childValidator.validElement(child, storable))
		})

	return err
}

// Validates element containers occurses
func (v *validator) validElementContOccurses(el *elementType) (err error) {
	v.schema.EnumContainers(
		func(cont schemas.Container) {
			occurs := schemas.Occurs(0)
			el.EnumElements(
				func(child *elementType) {
					if child.Container() == cont.Name() {
						occurs++
					}
				})
			if occurs < cont.MinOccurs() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidOccursMin, "%s container «%s» has not enough occurrences (%d, minimum %d): %w", v.entName(el), cont.Name(), occurs, cont.MinOccurs(), ErrMinOccursViolation))
			}
			if occurs > cont.MaxOccurs() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidOccursMax, "%s container «%s» has too many occurrences (%d, maximum %d): %w", v.entName(el), cont.Name(), occurs, cont.MaxOccurs(), ErrMaxOccursViolation))
			}
		})
	return err
}

// Validates specified record. If rawIDexpected then raw IDs is required
func (v *validator) validRecord(rec *recordType, rawIDexpected bool) (err error) {
	err = v.validRow(&rec.rowType)

	if v.schema.Kind().HasSystemField(schemas.SystemField_ID) {
		if rawIDexpected && !rec.ID().IsRaw() {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRawRecordID, "new %s ID «%d» is not raw: %w", v.entName(rec), rec.ID(), ErrRawRecordIDExpected))
		}
	}

	return err
}

// Validates specified row
func (v *validator) validRow(row *rowType) (err error) {
	v.schema.EnumFields(
		func(f schemas.Field) {
			if f.Required() {
				if !row.hasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "%s misses field «%s» required by schema «%v»: %w", v.entName(row), f.Name(), v.schema.QName(), ErrNameNotFound))
				}
			}
		})

	return err
}

// Validate specified object
func (v *validator) validObject(obj *elementType) error {
	return v.validElement(obj, false)
}

type validators struct {
	schemas    schemas.SchemaCache
	validators map[schemas.QName]*validator
}

func newValidators() *validators {
	v := validators{
		validators: make(map[schemas.QName]*validator),
	}
	return &v
}

// Prepares validator for specified schema cache
func (v *validators) prepare(schemaCache schemas.SchemaCache) {
	v.schemas = schemaCache
	schemaCache.EnumSchemas(
		func(s schemas.Schema) {
			v.validators[s.QName()] = newValidator(v, s)
		})
}

// Returns validator for specified schema
func (v *validators) validator(n schemas.QName) *validator {
	return v.validators[n]
}

// validate specified event. Must be called _after_ build() method
func (v *validators) validEvent(ev *eventType) (err error) {

	err = errors.Join(
		v.validEventObjects(ev),
		v.validEventCUDs(ev),
	)

	return err
}

// validEventObjects validate event parts: object and unlogged object
func (v *validators) validEventObjects(ev *eventType) (err error) {
	arg, argUnl, err := ev.argumentNames()
	if err != nil {
		return validateError(ECode_InvalidSchemaName, err)
	}

	if ev.argObject.QName() != arg {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidSchemaName, "event command argument «%v» uses wrong schema «%v», expected «%v»: %w", ev.name, ev.argObject.QName(), arg, ErrWrongSchema))
	} else if arg != schemas.NullQName {
		// #!17185: must be ODoc or Object only
		schema := v.schemas.Schema(arg)
		if (schema.Kind() != schemas.SchemaKind_ODoc) && (schema.Kind() != schemas.SchemaKind_Object) {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidSchemaKind, "event command argument «%v» schema can not to be «%v», expected («%v» or «%v»): %w", arg, schema.Kind(), schemas.SchemaKind_ODoc, schemas.SchemaKind_Object, ErrWrongSchema))
		}
		err = errors.Join(err,
			v.validObject(&ev.argObject))
	}

	if ev.argUnlObj.QName() != argUnl {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidSchemaName, "event command unlogged argument «%v» uses wrong schema «%v», expected «%v»: %w", ev.name, ev.argUnlObj.QName(), argUnl, ErrWrongSchema))
	} else if ev.argUnlObj.QName() != schemas.NullQName {
		err = errors.Join(err,
			v.validObject(&ev.argUnlObj))
	}

	return err
}

// validEventCUDs validate event CUD parts: argument CUDs and result CUDs
func (v *validators) validEventCUDs(ev *eventType) (err error) {
	if ev.cud.empty() {
		if ev.name == istructs.QNameCommandCUD {
			return validateErrorf(ECode_EEmptyCUDs, "event «%v» must have not empty CUDs: %w", ev.name, ErrCUDsMissed)
		}
		return nil
	}

	return v.validCUD(&ev.cud, ev.sync)
}

// Validates specified document or object
func (v *validators) validObject(obj *elementType) (err error) {
	if obj.QName() == schemas.NullQName {
		return validateErrorf(ECode_EmptySchemaName, "element «%s» has empty schema name: %w", obj.Container(), ErrNameMissed)
	}

	validator := v.validator(obj.QName())

	if validator == nil {
		return validateErrorf(ECode_InvalidSchemaName, "object refers to unknown schema «%v»: %w", obj.QName(), ErrNameNotFound)
	}

	switch validator.schema.Kind() {
	case schemas.SchemaKind_GDoc, schemas.SchemaKind_CDoc, schemas.SchemaKind_ODoc, schemas.SchemaKind_WDoc:
		return validator.validDocument(obj)
	case schemas.SchemaKind_Object:
		return validator.validObject(obj)
	}

	return validateErrorf(ECode_InvalidSchemaKind, "object refers to invalid schema «%v» kind «%v»: %w", obj.QName(), validator.schema.Kind(), ErrUnexpectedShemaKind)
}

// validates specified CUD
func (v *validators) validCUD(cud *cudType, allowStorageIDsInCreate bool) (err error) {
	for _, newRec := range cud.creates {
		err = errors.Join(err,
			v.validRecord(newRec, !allowStorageIDsInCreate))
	}

	err = errors.Join(err,
		v.validCUDIDsUnique(cud),
		v.validCUDRefRawIDs(cud),
	)

	for _, updRec := range cud.updates {
		err = errors.Join(err,
			v.validRecord(&updRec.result, false))
	}

	return err
}

// Validates IDs in CUD for unique
func (v *validators) validCUDIDsUnique(cud *cudType) (err error) {
	const errRecIDViolatedWrap = "cud.%s record ID «%d» is used repeatedly: %w"

	ids := make(map[istructs.RecordID]bool)

	for _, rec := range cud.creates {
		id := rec.ID()
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRecordID, errRecIDViolatedWrap, "create", id, ErrRecordIDUniqueViolation))
		}
		ids[id] = true
	}

	for _, rec := range cud.updates {
		id := rec.changes.ID()
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRecordID, errRecIDViolatedWrap, "update", id, ErrRecordIDUniqueViolation))
		}
		ids[id] = true
	}

	return err
}

// Validates references to raw IDs in specified CUD
func (v *validators) validCUDRefRawIDs(cud *cudType) (err error) {

	rawIDs := make(map[istructs.RecordID]bool)

	for _, rec := range cud.creates {
		id := rec.ID()
		if id.IsRaw() {
			rawIDs[id] = true
		}
	}

	checkRefs := func(rec *recordType, cu string) (err error) {
		rec.RecordIDs(false,
			func(name string, id istructs.RecordID) {
				if id.IsRaw() && !rawIDs[id] {
					err = errors.Join(err,
						validateErrorf(ECode_InvalidRefRecordID, "cud.%s record «%s» field «%s» refers to unknown raw ID «%d»: %w", cu, rec.Container(), name, id, ErrorRecordIDNotFound))
				}
			})
		return err
	}

	for _, rec := range cud.creates {
		err = errors.Join(err,
			checkRefs(rec, "create"))
	}

	for _, rec := range cud.updates {
		err = errors.Join(err,
			checkRefs(&rec.changes, "update"))
	}

	return err
}

// Validates specified view key. If partialClust specified then clustering columns row may be partially filled
func (v *validators) validKey(key *keyType, partialClust bool) (err error) {
	partSchema := key.partKeySchema()
	if key.partRow.QName() != partSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view partition key schema «%v», for view «%v» expected «%v»: %w", key.partRow.QName(), key.viewName, partSchema, ErrWrongSchema)
	}

	clustSchema := key.clustColsSchema()
	if key.clustRow.QName() != clustSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view clustering columns schema «%v», for view «%v» expected «%v»: %w", key.clustRow.QName(), key.viewName, clustSchema, ErrWrongSchema)
	}

	key.partRow.schema.EnumFields(
		func(f schemas.Field) {
			if !key.partRow.hasValue(f.Name()) {
				err = errors.Join(err,
					validateErrorf(ECode_EmptyData, "view «%v» partition key «%v» field «%s» is empty: %w", key.viewName, partSchema, f.Name(), ErrFieldIsEmpty))
			}
		})

	if !partialClust {
		key.clustRow.schema.EnumFields(
			func(f schemas.Field) {
				if !key.clustRow.hasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "view «%v» clustering columns «%v» field «%s» is empty: %w", key.viewName, clustSchema, f.Name(), ErrFieldIsEmpty))
				}
			})
	}

	return err
}

// Validates specified view value
func (v *validators) validViewValue(value *valueType) (err error) {
	valSchema := value.valueSchema()
	if value.QName() != valSchema {
		return validateErrorf(ECode_InvalidSchemaName, "wrong view value schema «%v», for view «%v» expected «%v»: %w", value.QName(), value.viewName, valSchema, ErrWrongSchema)
	}

	validator := v.validator(valSchema)
	if validator == nil {
		return validateErrorf(ECode_InvalidSchemaName, "view value «%v» schema not found: %w", valSchema, ErrNameNotFound)
	}

	return validator.validRow(&value.rowType)
}

// Validates specified record. If rawIDexpected then raw IDs is required
func (v *validators) validRecord(rec *recordType, rawIDexpected bool) (err error) {
	if rec.QName() == schemas.NullQName {
		return validateErrorf(ECode_EmptySchemaName, "record «%s» has empty schema name: %w", rec.Container(), ErrNameMissed)
	}

	validator := v.validator(rec.QName())
	if validator == nil {
		return validateErrorf(ECode_InvalidSchemaName, "object refers to unknown schema «%v»: %w", rec.QName(), ErrNameNotFound)
	}

	switch validator.schema.Kind() {
	case schemas.SchemaKind_GDoc, schemas.SchemaKind_CDoc, schemas.SchemaKind_ODoc, schemas.SchemaKind_WDoc, schemas.SchemaKind_GRecord, schemas.SchemaKind_CRecord, schemas.SchemaKind_ORecord, schemas.SchemaKind_WRecord:
		return validator.validRecord(rec, rawIDexpected)
	}

	return validateErrorf(ECode_InvalidSchemaKind, "record «%s» refers to invalid schema «%v» kind «%v»: %w", rec.Container(), rec.QName(), validator.schema.Kind(), ErrUnexpectedShemaKind)
}
