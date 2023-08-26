/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// TODO: move to internal/validate package

// Provides validation application structures by single definition
type validator struct {
	validators *validators
	def        appdef.IDef
}

func newValidator(validators *validators, def appdef.IDef) *validator {
	return &validator{validators, def}
}

// Return readable name of entity to validate.
//
// If entity has only type QName, then the result will be short like `CDoc (sales.BillDocument)`, otherwise it will be complete like `CRecord «Price» (sales.PriceRecord)`
func (v *validator) entName(e interface{}) string {
	ent := v.def.Kind().TrimString()
	name := ""
	typeName := v.def.QName()

	if row, ok := e.(istructs.IRowReader); ok {
		if qName := row.AsQName(appdef.SystemField_QName); qName != appdef.NullQName {
			typeName = qName
			if (qName == v.def.QName()) && v.def.Kind().HasSystemField(appdef.SystemField_Container) {
				if cont := row.AsString(appdef.SystemField_Container); cont != "" {
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

// Validate specified document
func (v *validator) validDocument(doc *elementType) error {
	// TODO: check RecordID refs available for document kind
	return v.validElement(doc, true)
}

// Validate specified element
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

// Validates element containers
func (v *validator) validElementContainers(el *elementType, storable bool) (err error) {
	def, ok := v.def.(appdef.IContainers)
	if !ok {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidDefName, "%s has definition kind «%s» without containers: %w", v.entName(el), v.def.Kind().TrimString(), ErrUnexpectedDefKind))
		return err
	}

	// validates element containers occurs
	def.Containers(
		func(cont appdef.IContainer) {
			occurs := appdef.Occurs(0)
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

	// validate element children
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
			cont := def.Container(childName)
			if cont == nil {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidElementName, "%s child[%d] has unknown container name «%s»: %w", v.entName(el), idx, childName, ErrNameNotFound))
				return
			}

			childQName := child.QName()
			if childQName != cont.QName() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidDefName, "%s child[%d] «%s» has wrong definition name «%v», expected «%v»: %w", v.entName(el), idx, childName, childQName, cont.QName(), ErrNameNotFound))
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
					validateErrorf(ECode_InvalidDefName, "object refers to unknown definition «%v»: %w", childQName, ErrNameNotFound))
				return
			}
			err = errors.Join(err,
				childValidator.validElement(child, storable))
		})

	return err
}

// Validates specified record. If rawIDexpected then raw IDs is required
func (v *validator) validRecord(rec *recordType, rawIDexpected bool) (err error) {
	err = v.validRow(&rec.rowType)

	if v.def.Kind().HasSystemField(appdef.SystemField_ID) {
		if rawIDexpected && !rec.ID().IsRaw() {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRawRecordID, "new %s ID «%d» is not raw: %w", v.entName(rec), rec.ID(), ErrRawRecordIDExpected))
		}
	}

	return err
}

// Validates specified row
func (v *validator) validRow(row *rowType) (err error) {
	v.def.(appdef.IFields).Fields(
		func(f appdef.IField) {
			if f.Required() {
				if !row.HasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "%s misses field «%s» required by definition «%v»: %w", v.entName(row), f.Name(), v.def.QName(), ErrNameNotFound))
				}
			}
		})

	return err
}

// Validate specified object
func (v *validator) validObject(obj *elementType) error {
	return v.validElement(obj, false)
}

// Application definitions validators
type validators struct {
	appDef     appdef.IAppDef
	validators map[appdef.QName]*validator
}

func newValidators() *validators {
	return &validators{
		validators: make(map[appdef.QName]*validator),
	}
}

// Prepares validators for specified application definition
func (v *validators) prepare(appDef appdef.IAppDef) {
	v.appDef = appDef
	v.appDef.Defs(
		func(d appdef.IDef) {
			v.validators[d.QName()] = newValidator(v, d)
		})
}

// Returns validator for specified definition
func (v *validators) validator(n appdef.QName) *validator {
	return v.validators[n]
}

// Validate specified event.
//
// Must be called _after_ build() method
func (v *validators) validEvent(ev *eventType) (err error) {

	err = errors.Join(
		v.validEventObjects(ev),
		v.validEventCUDs(ev),
	)

	return err
}

// Validate event parts: object and secure object
func (v *validators) validEventObjects(ev *eventType) (err error) {
	arg, argUnl, err := ev.argumentNames()
	if err != nil {
		return validateError(ECode_InvalidDefName, err)
	}

	if ev.argObject.QName() != arg {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidDefName, "event command argument «%v» uses wrong definition «%v», expected «%v»: %w", ev.name, ev.argObject.QName(), arg, ErrWrongDefinition))
	} else if arg != appdef.NullQName {
		// #!17185: must be ODoc or Object only
		def := v.appDef.Def(arg)
		if (def.Kind() != appdef.DefKind_ODoc) && (def.Kind() != appdef.DefKind_Object) {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidDefKind, "event command argument «%v» definition can not to be «%v», expected («%v» or «%v»): %w", arg, def.Kind().TrimString(), appdef.DefKind_ODoc.TrimString(), appdef.DefKind_Object.TrimString(), ErrWrongDefinition))
		}
		err = errors.Join(err,
			v.validObject(&ev.argObject))
	}

	if ev.argUnlObj.QName() != argUnl {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidDefName, "event command un-logged argument «%v» uses wrong definition «%v», expected «%v»: %w", ev.name, ev.argUnlObj.QName(), argUnl, ErrWrongDefinition))
	} else if ev.argUnlObj.QName() != appdef.NullQName {
		err = errors.Join(err,
			v.validObject(&ev.argUnlObj))
	}

	return err
}

// Validate event CUD parts: argument CUDs and result CUDs
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
	if obj.QName() == appdef.NullQName {
		return validateErrorf(ECode_EmptyDefName, "element «%s» has empty definition name: %w", obj.Container(), ErrNameMissed)
	}

	validator := v.validator(obj.QName())

	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "object refers to unknown definition «%v»: %w", obj.QName(), ErrNameNotFound)
	}

	switch validator.def.Kind() {
	case appdef.DefKind_GDoc, appdef.DefKind_CDoc, appdef.DefKind_ODoc, appdef.DefKind_WDoc:
		return validator.validDocument(obj)
	case appdef.DefKind_Object:
		return validator.validObject(obj)
	}

	return validateErrorf(ECode_InvalidDefKind, "object refers to invalid definition «%v» kind «%s»: %w", obj.QName(), validator.def.Kind().TrimString(), ErrUnexpectedDefKind)
}

// Validates specified CUD
func (v *validators) validCUD(cud *cudType, allowStorageIDsInCreate bool) (err error) {
	for _, newRec := range cud.creates {
		err = errors.Join(err,
			v.validRecord(newRec, !allowStorageIDsInCreate))
	}

	err = errors.Join(err,
		v.validCUDsUnique(cud),
		v.validCUDRefRawIDs(cud),
	)

	for _, updRec := range cud.updates {
		err = errors.Join(err,
			v.validRecord(&updRec.result, false))
	}

	return err
}

// Validates IDs in CUD for unique
func (v *validators) validCUDsUnique(cud *cudType) (err error) {
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

// Validates specified view key.
//
// If partialClust specified then clustering columns row may be partially filled
func (v *validators) validKey(key *keyType, partialClust bool) (err error) {
	pkDef := key.pkDef()
	if key.partRow.QName() != pkDef {
		return validateErrorf(ECode_InvalidDefName, "wrong view partition key definition «%v», for view «%v» expected «%v»: %w", key.partRow.QName(), key.viewName, pkDef, ErrWrongDefinition)
	}

	ccDef := key.ccDef()
	if key.ccolsRow.QName() != ccDef {
		return validateErrorf(ECode_InvalidDefName, "wrong view clustering columns definition «%v», for view «%v» expected «%v»: %w", key.ccolsRow.QName(), key.viewName, ccDef, ErrWrongDefinition)
	}

	key.partRow.fieldsDef().Fields(
		func(f appdef.IField) {
			if !key.partRow.HasValue(f.Name()) {
				err = errors.Join(err,
					validateErrorf(ECode_EmptyData, "view «%v» partition key «%v» field «%s» is empty: %w", key.viewName, pkDef, f.Name(), ErrFieldIsEmpty))
			}
		})

	if !partialClust {
		key.ccolsRow.fieldsDef().Fields(
			func(f appdef.IField) {
				if !key.ccolsRow.HasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "view «%v» clustering columns «%v» field «%s» is empty: %w", key.viewName, ccDef, f.Name(), ErrFieldIsEmpty))
				}
			})
	}

	return err
}

// Validates specified view value
func (v *validators) validViewValue(value *valueType) (err error) {
	valDef := value.valueDef()
	if value.QName() != valDef {
		return validateErrorf(ECode_InvalidDefName, "wrong view value definition «%v», for view «%v» expected «%v»: %w", value.QName(), value.viewName, valDef, ErrWrongDefinition)
	}

	validator := v.validator(valDef)
	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "view value «%v» definition not found: %w", valDef, ErrNameNotFound)
	}

	return validator.validRow(&value.rowType)
}

// Validates specified record.
//
// If rawIDexpected then raw IDs is required
func (v *validators) validRecord(rec *recordType, rawIDexpected bool) (err error) {
	if rec.QName() == appdef.NullQName {
		return validateErrorf(ECode_EmptyDefName, "record «%s» has empty definition name: %w", rec.Container(), ErrNameMissed)
	}

	validator := v.validator(rec.QName())
	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "object refers to unknown definition «%v»: %w", rec.QName(), ErrNameNotFound)
	}

	switch validator.def.Kind() {
	case appdef.DefKind_GDoc, appdef.DefKind_CDoc, appdef.DefKind_ODoc, appdef.DefKind_WDoc, appdef.DefKind_GRecord, appdef.DefKind_CRecord, appdef.DefKind_ORecord, appdef.DefKind_WRecord:
		return validator.validRecord(rec, rawIDexpected)
	}

	return validateErrorf(ECode_InvalidDefKind, "record «%s» refers to invalid definition «%v» kind «%s»: %w", rec.Container(), rec.QName(), validator.def.Kind().TrimString(), ErrUnexpectedDefKind)
}
