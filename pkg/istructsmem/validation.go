/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Provides validation application structures by single type
type validator struct {
	validators *validators
	typ        appdef.IType
}

func newValidator(validators *validators, t appdef.IType) *validator {
	return &validator{validators, t}
}

// Validate specified document
func (v *validator) validDocument(doc *elementType) error {
	err := v.validElement(doc)

	ids := make(map[istructs.RecordID]*elementType)

	_ = doc.forEach(func(e *elementType) error {
		if id := e.ID(); id.IsRaw() {
			if _, exists := ids[id]; exists {
				err = errors.Join(err,
					// event argument repeatedly uses record ID «1» inside ODoc (test.doc)
					validateErrorf(ECode_InvalidRawRecordID, "%v repeatedly uses raw record ID «%d»: %w", v, id, ErrRecordIDUniqueViolation))
			}
			ids[id] = e
		}
		return nil
	})

	_ = doc.forEach(func(e *elementType) error {
		e.fields.RefFields(func(fld appdef.IRefField) {
			if id := e.AsRecordID(fld.Name()); id.IsRaw() {
				target, exists := ids[id]
				if !exists {
					err = errors.Join(err,
						// ORecord «Price» (sales.PriceRecord) field «Next» refers to unknown record ID «7»
						validateErrorf(ECode_InvalidRefRecordID, "%v field «%s» refers to unknown raw record ID «%d»: %w", e, fld.Name(), id, ErrRecordIDNotFound))
					return
				}
				if !fld.Ref(target.QName()) {
					err = errors.Join(err,
						// ORecord «Price» (sales.PriceRecord) field «Next» refers to record ID «1» that has unavailable target QName «sales.Order»
						validateErrorf(ECode_InvalidRefRecordID, "%v field «%s» refers to raw record ID «%d» that has unavailable target QName «%s»: %w", e, fld.Name(), id, target.QName(), ErrWrongRecordID))
				}
			}
		})
		return nil
	})

	return err
}

// Validate specified element
func (v *validator) validElement(el *elementType) (err error) {
	if el.typ.Kind().HasSystemField(appdef.SystemField_ID) {
		err = v.validRecord(&el.recordType, true)
	} else {
		err = v.validRow(&el.recordType.rowType)
	}

	err = errors.Join(err,
		v.validContainers(el))

	return err
}

// Validates element containers
func (v *validator) validContainers(el *elementType) (err error) {
	t := v.typ.(appdef.IContainers)

	t.Containers(
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
					validateErrorf(ECode_InvalidOccursMin, "%v container «%s» has not enough occurrences (%d, minimum %d): %w", el, cont.Name(), occurs, cont.MinOccurs(), ErrMinOccursViolation))
			}
			if occurs > cont.MaxOccurs() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidOccursMax, "%v container «%s» has too many occurrences (%d, maximum %d): %w", el, cont.Name(), occurs, cont.MaxOccurs(), ErrMaxOccursViolation))
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
					validateErrorf(ECode_EmptyElementName, "%v child[%d] has empty container name: %w", el, idx, ErrNameMissed))
				return
			}
			cont := t.Container(childName)
			if cont == nil {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidElementName, "%v child[%d] has unknown container name «%s»: %w", el, idx, childName, ErrNameNotFound))
				return
			}

			childQName := child.QName()
			if childQName != cont.QName() {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidDefName, "%v child[%d] %v has wrong type name, expected «%v»: %w", el, idx, child, cont.QName(), ErrNameNotFound))
				return
			}

			if child.typ.Kind().HasSystemField(appdef.SystemField_ParentID) {
				parID := child.Parent()
				if parID == istructs.NullRecordID {
					child.setParent(elID) // if child parentID omitted, then restore it
				} else {
					if parID != elID {
						err = errors.Join(err,
							validateErrorf(ECode_InvalidRefRecordID, "%v child[%d] %v has wrong parent id «%d», expected «%d»: %w", el, idx, child, parID, elID, ErrWrongRecordID))
					}
				}
			}

			childValidator := v.validators.validator(childQName)
			if childValidator == nil {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidDefName, "%v refers to unknown type «%v»: %w", child, childQName, ErrNameNotFound))
				return
			}
			err = errors.Join(err,
				childValidator.validElement(child))
		})

	return err
}

// Validates specified record. If rawIDexpected then raw IDs is required
func (v *validator) validRecord(rec *recordType, rawIDexpected bool) (err error) {
	err = v.validRow(&rec.rowType)

	if v.typ.Kind().HasSystemField(appdef.SystemField_ID) {
		if rawIDexpected && !rec.ID().IsRaw() {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRawRecordID, "new %v ID «%d» is not raw: %w", rec, rec.ID(), ErrRawRecordIDExpected))
		}
	}

	return err
}

// Validates specified row
func (v *validator) validRow(row *rowType) (err error) {
	row.fields.Fields(
		func(f appdef.IField) {
			if f.Required() {
				if !row.HasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "%v misses required field «%s»: %w", row, f.Name(), ErrNameNotFound))
					return
				}
				if !f.IsSys() {
					switch f.DataKind() {
					case appdef.DataKind_RecordID:
						if row.AsRecordID(f.Name()) == istructs.NullRecordID {
							err = errors.Join(err,
								validateErrorf(ECode_InvalidRefRecordID, "%v required ref field «%s» has NullRecordID value: %w", row, f.Name(), ErrWrongRecordID))
						}
					}
				}
			}
		})

	return err
}

// Validate specified object
func (v *validator) validObject(obj *elementType) error {
	return v.validElement(obj)
}

// Application types validators
type validators struct {
	appDef     appdef.IAppDef
	validators map[appdef.QName]*validator
}

func newValidators() *validators {
	return &validators{
		validators: make(map[appdef.QName]*validator),
	}
}

// Prepares validators for specified application
func (v *validators) prepare(appDef appdef.IAppDef) {
	v.appDef = appDef
	v.appDef.Types(
		func(t appdef.IType) {
			v.validators[t.QName()] = newValidator(v, t)
		})
}

// Returns validator for specified type
func (v *validators) validator(n appdef.QName) *validator {
	return v.validators[n]
}

// Validate specified event.
//
// Must be called _after_ build() method
func (v *validators) validEvent(ev *eventType) (err error) {

	err = errors.Join(
		v.validEventRawIDs(ev),
		v.validEventArgs(ev),
		v.validEventCUDs(ev),
	)

	return err
}

// Validate event parts: object and secure object
func (v *validators) validEventArgs(ev *eventType) (err error) {
	arg, argUnl, err := ev.argumentNames()
	if err != nil {
		return validateError(ECode_InvalidDefName, err)
	}

	if ev.argObject.QName() != arg {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidDefName, "event command argument «%v» uses wrong type «%v», expected «%v»: %w", ev.name, ev.argObject.QName(), arg, ErrWrongType))
	} else if arg != appdef.NullQName {
		// #!17185: must be ODoc or Object only
		t := v.appDef.Type(arg)
		if (t.Kind() != appdef.TypeKind_ODoc) && (t.Kind() != appdef.TypeKind_Object) {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidTypeKind, "event command argument «%v» type can not to be «%v», expected («%v» or «%v»): %w", arg, t.Kind().TrimString(), appdef.TypeKind_ODoc.TrimString(), appdef.TypeKind_Object.TrimString(), ErrWrongType))
		}
		err = errors.Join(err,
			v.validArgument(&ev.argObject))
	}

	if ev.argUnlObj.QName() != argUnl {
		err = errors.Join(err,
			validateErrorf(ECode_InvalidDefName, "event command un-logged argument «%v» uses wrong type «%v», expected «%v»: %w", ev.name, ev.argUnlObj.QName(), argUnl, ErrWrongType))
	} else if ev.argUnlObj.QName() != appdef.NullQName {
		err = errors.Join(err,
			v.validArgument(&ev.argUnlObj))
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

// Validates raw IDs in specified event.
//
// Checks that raw IDs is unique.
//
// Checks that references to raw IDs is valid, that is target raw ID unknown and has available QName.
//
// For CUDs checks that raw IDs in sys.ParentID field and value in sys.Container are confirmable for target parent.
func (v *validators) validEventRawIDs(ev *eventType) (err error) {
	const (
		errRepeatedRawID        = "%v repeatedly uses raw record ID «%d»: %w"
		errUnknownRawIDRef      = "%v field «%s» refers to unknown raw record ID «%d»: %w"
		errUnavailableTargetRef = "%v field «%s» refers to raw record ID «%d» that has unavailable target QName «%s»: %w"
	)

	ids := make(map[istructs.RecordID]appdef.QName)

	_ = ev.argObject.forEach(func(e *elementType) error {
		if id := e.ID(); id.IsRaw() {
			if _, exists := ids[id]; exists {
				err = errors.Join(err,
					// ORecord «ORec: test.ORecord» repeatedly uses raw record ID «1»
					validateErrorf(ECode_InvalidRawRecordID, errRepeatedRawID, e, id, ErrRecordIDUniqueViolation))
			}
			ids[id] = e.QName()
		}
		return nil
	})

	_ = ev.argObject.forEach(func(e *elementType) error {
		e.fields.RefFields(func(fld appdef.IRefField) {
			if id := e.AsRecordID(fld.Name()); id.IsRaw() {
				target, exists := ids[id]
				if !exists {
					err = errors.Join(err,
						// ORecord «ORec: test.ORecord» field «Next» refers to unknown raw record ID «7»
						validateErrorf(ECode_InvalidRefRecordID, errUnknownRawIDRef, e, fld.Name(), id, ErrRecordIDNotFound))
					return
				}
				if !fld.Ref(target) {
					err = errors.Join(err,
						// ORecord «ORec: sales.PriceRecord» field «Next» refers to raw record ID «1» that has unavailable target QName «test.ODocument»
						validateErrorf(ECode_InvalidRefRecordID, errUnavailableTargetRef, e, fld.Name(), id, target, ErrWrongRecordID))
				}
			}
		})
		return nil
	})

	for _, rec := range ev.cud.creates {
		id := rec.ID()
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				// WRecord «WRec: test.WRecord» repeatedly uses record ID «1»
				validateErrorf(ECode_InvalidRawRecordID, errRepeatedRawID, rec, id, ErrRecordIDUniqueViolation))
		}
		ids[id] = rec.QName()
	}

	checkRefs := func(rec *recordType) (err error) {
		rec.RecordIDs(false,
			func(name string, id istructs.RecordID) {
				if id.IsRaw() {
					target, ok := ids[id]
					if !ok {
						err = errors.Join(err,
							validateErrorf(ECode_InvalidRefRecordID, errUnknownRawIDRef, rec, name, id, ErrRecordIDNotFound))
						return
					}
					switch name {
					case appdef.SystemField_ParentID:
						if parentType, ok := v.appDef.Type(target).(appdef.IContainers); ok {
							cont := parentType.Container(rec.Container())
							if cont == nil {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, "%v has raw parent ID «%d» refers to «%s», which has no container «%s»: %w", rec, id, target, rec.Container(), ErrWrongRecordID))
								return
							}
							if cont.QName() != rec.QName() {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, "%v has raw parent ID «%d» refers to «%s», which container «%s» has another QName «%s»: %w", rec, id, target, rec.Container(), cont.QName(), ErrWrongRecordID))
								return
							}
						}
					default:
						fld := rec.fieldDef(name)
						if ref, ok := fld.(appdef.IRefField); ok {
							if !ref.Ref(target) {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, errUnavailableTargetRef, rec, name, id, target, ErrWrongRecordID))
								return
							}
						}
					}
				}
			})
		return err
	}

	for _, rec := range ev.cud.creates {
		err = errors.Join(err,
			checkRefs(rec))
	}

	for _, rec := range ev.cud.updates {
		err = errors.Join(err,
			checkRefs(&rec.changes))
	}

	return err
}

// Validates specified document or object
func (v *validators) validArgument(obj *elementType) (err error) {
	if obj.QName() == appdef.NullQName {
		return validateErrorf(ECode_EmptyDefName, "element «%s» has empty type name: %w", obj.Container(), ErrNameMissed)
	}

	validator := v.validator(obj.QName())

	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "object refers to unknown type «%v»: %w", obj.QName(), ErrNameNotFound)
	}

	switch validator.typ.Kind() {
	case appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_ODoc, appdef.TypeKind_WDoc:
		return validator.validDocument(obj)
	case appdef.TypeKind_Object:
		return validator.validObject(obj)
	}

	return validateErrorf(ECode_InvalidTypeKind, "object refers to invalid type «%v» kind «%s»: %w", obj.QName(), validator.typ.Kind().TrimString(), ErrUnexpectedTypeKind)
}

// Validates specified CUD
func (v *validators) validCUD(cud *cudType, isSyncEvent bool) (err error) {
	for _, newRec := range cud.creates {
		err = errors.Join(err,
			v.validCUDRecord(newRec, !isSyncEvent))
	}

	err = errors.Join(err,
		v.validCUDsUnique(cud),
		v.validCUDRefRawIDs(cud),
	)

	for _, updRec := range cud.updates {
		err = errors.Join(err,
			v.validCUDRecord(&updRec.result, false))
	}

	return err
}

// Validates IDs in CUD for unique
func (v *validators) validCUDsUnique(cud *cudType) (err error) {
	const errRecIDViolatedWrap = "cud.%s record ID «%d» is used repeatedly: %w"

	ids := make(map[istructs.RecordID]bool)
	singletons := make(map[appdef.QName]istructs.RecordID)

	for _, rec := range cud.creates {
		id := rec.ID()
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				validateErrorf(ECode_InvalidRawRecordID, errRecIDViolatedWrap, "create", id, ErrRecordIDUniqueViolation))
		}
		ids[id] = true

		if cDoc, ok := rec.typ.(appdef.ICDoc); ok && cDoc.Singleton() {
			if id, ok := singletons[cDoc.QName()]; ok {
				err = errors.Join(err,
					validateErrorf(ECode_InvalidRawRecordID, "cud.create repeatedly creates the same singleton «%v» (record ID «%d» and «%d»): %w ", cDoc.QName(), id, rec.id, ErrRecordIDUniqueViolation))
			}
			singletons[cDoc.QName()] = rec.id
		}
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

	rawIDs := make(map[istructs.RecordID]appdef.QName)

	for _, rec := range cud.creates {
		id := rec.ID()
		if id.IsRaw() {
			rawIDs[id] = rec.QName()
		}
	}

	checkRefs := func(rec *recordType, cu string) (err error) {
		rec.RecordIDs(false,
			func(name string, id istructs.RecordID) {
				if id.IsRaw() {
					target, ok := rawIDs[id]
					if !ok {
						err = errors.Join(err,
							validateErrorf(ECode_InvalidRefRecordID, "cud.%s record «%s: %s» field «%s» refers to unknown raw ID «%d»: %w", cu, rec.Container(), rec.QName(), name, id, ErrRecordIDNotFound))
						return
					}
					switch name {
					case appdef.SystemField_ParentID:
						if parentType, ok := v.appDef.Type(target).(appdef.IContainers); ok {
							cont := parentType.Container(rec.Container())
							if cont == nil {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, "cud.%s record «%s: %s» with raw parent ID «%d» refers to «%s», which has no container «%s»: %w", cu, rec.Container(), rec.QName(), id, target, rec.Container(), ErrWrongRecordID))
								return
							}
							if cont.QName() != rec.QName() {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, "cud.%s record «%s: %s» with raw parent ID «%d» refers to «%s» container «%s», which has another QName «%s»: %w", cu, rec.Container(), rec.QName(), id, target, rec.Container(), cont.QName(), ErrWrongRecordID))
								return
							}
						}
					default:
						fld := rec.fieldDef(name)
						if ref, ok := fld.(appdef.IRefField); ok {
							if !ref.Ref(target) {
								err = errors.Join(err,
									validateErrorf(ECode_InvalidRefRecordID, "cud.%s record «%s: %s» field «%s» refers to raw ID «%d» that has unavailable target QName «%s»: %w", cu, rec.Container(), rec.QName(), name, id, target, ErrWrongRecordID))
								return
							}
						}
					}
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
func (v *validators) validViewKey(key *keyType, partialClust bool) (err error) {
	key.partRow.fields.Fields(
		func(f appdef.IField) {
			if !key.partRow.HasValue(f.Name()) {
				err = errors.Join(err,
					validateErrorf(ECode_EmptyData, "view «%v» partition key field «%s» is empty: %w", key.viewName, f.Name(), ErrFieldIsEmpty))
			}
		})

	if !partialClust {
		key.ccolsRow.fields.Fields(
			func(f appdef.IField) {
				if !key.ccolsRow.HasValue(f.Name()) {
					err = errors.Join(err,
						validateErrorf(ECode_EmptyData, "view «%v» clustering columns field «%s» is empty: %w", key.viewName, f.Name(), ErrFieldIsEmpty))
				}
			})
	}

	return err
}

// Validates specified view value
func (v *validators) validViewValue(value *valueType) (err error) {
	validator := v.validator(value.viewName)
	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "view value «%v» type not found: %w", value.viewName, ErrNameNotFound)
	}

	return validator.validRow(&value.rowType)
}

// Validates specified CUD record.
//
// If rawIDexpected then raw IDs is required
func (v *validators) validCUDRecord(rec *recordType, rawIDexpected bool) (err error) {
	if rec.QName() == appdef.NullQName {
		return validateErrorf(ECode_EmptyDefName, "record «%s» has empty type name: %w", rec.Container(), ErrNameMissed)
	}

	validator := v.validator(rec.QName())
	if validator == nil {
		return validateErrorf(ECode_InvalidDefName, "object refers to unknown type «%v»: %w", rec.QName(), ErrNameNotFound)
	}

	switch validator.typ.Kind() {
	case appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_GRecord, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		return validator.validRecord(rec, rawIDexpected)
	}

	return validateErrorf(ECode_InvalidTypeKind, "record «%s» refers to invalid type «%v» kind «%s»: %w", rec.Container(), rec.QName(), validator.typ.Kind().TrimString(), ErrUnexpectedTypeKind)
}
