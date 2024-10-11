/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"math"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// # Validates specified event.
//
// Checks IDs used in event.
//
// Checks event arguments.
//
// Checks event CUDs.
func validateEvent(ev *eventType) error {
	return errors.Join(
		validateEventIDs(ev),
		validateEventArgs(ev),
		validateEventCUDs(ev),
	)
}

// # Validates IDs in specified event.
//
// Checks that IDs used for is unique (both for argument and CUDs, both for create and update).
//
// Checks that ID references to created entities is valid, that target ID is known and has available QName.
//
// For CUDs.Create() checks that IDs in sys.ParentID field and value in sys.Container are confirmable for target parent.
func validateEventIDs(ev *eventType) error {

	ids, err := validateObjectIDs(&ev.argObject, !ev.Synced())

	err = errors.Join(err,
		validateEventCUDsIDs(ev, ids))

	return err
}

// # Validates IDs in specified object.
//
// Checks that IDs is unique.
//
// If `rawID` specified, then checks that raw IDs is only used.
//
// Checks that ID references to created entities is valid, that is target ID known and has available QName.
//
// Returns IDs map and error(s) if any.
func validateObjectIDs(obj *objectType, rawID bool) (ids map[istructs.RecordID]appdef.QName, err error) {
	ids = make(map[istructs.RecordID]appdef.QName)

	_ = obj.forEach(func(e *objectType) error {
		id := e.ID()
		if id == istructs.NullRecordID {
			return nil
		}
		if !id.IsRaw() {
			if rawID {
				err = errors.Join(err,
					// ODoc «test.document» should use raw record ID (not «123456789012345») in created ORecord «Rec: test.ORecord»
					validateErrorf(ECode_InvalidRecordID, errRequiredRawID, obj, id, e, ErrRawRecordIDRequired))
			}
		}
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				// ODoc «test.document» repeatedly uses record ID «1» in ORecord «child: test.record1»
				validateErrorf(ECode_InvalidRecordID, errRepeatedID, obj, id, e, ErrRecordIDUniqueViolation))
		}
		ids[id] = e.QName()
		return nil
	})

	_ = obj.forEach(func(e *objectType) error {
		for _, fld := range e.fields.RefFields() {
			if id := e.AsRecordID(fld.Name()); id != istructs.NullRecordID {
				target, exists := ids[id]
				if !exists {
					if id.IsRaw() {
						err = errors.Join(err,
							// ODoc «test.document» field «RefField» refers to unknown record ID «7»
							validateErrorf(ECode_InvalidRefRecordID, errUnknownIDRef, e, fld.Name(), id, ErrRecordIDNotFound))
					}
					continue
				}
				if !fld.Ref(target) {
					err = errors.Join(err,
						// ODoc «test.document» field «RefField» refers to record ID «1» that has unavailable target QName «test.document»
						validateErrorf(ECode_InvalidRefRecordID, errUnavailableTargetRef, e, fld.Name(), id, target, ErrWrongRecordID))
				}
			}
		}
		return nil
	})

	return ids, err
}

// # Validates IDs in CUDs of specified event.
//
// Checks that IDs is unique both for creates and updates and not intersects with passed arguments ids.
//
// If event is not synced then checks that only raw IDs is used in CUD.Create().
//
// If singletons is created in the event, then it checks that no the singleton is created twice.
//
// Checks that raw IDs in CUD.Update() is not used.
//
// Checks that ID references to created entities is valid, that target ID is known and has available QName.
//
// Checks for CUD.Create() that IDs in sys.ParentID field and value in sys.Container are confirmable for target parent.
func validateEventCUDsIDs(ev *eventType, ids map[istructs.RecordID]appdef.QName) (err error) {
	st := make(map[appdef.QName]istructs.RecordID) // singletons unique

	for _, rec := range ev.cud.creates {
		id := rec.ID()
		if id == istructs.NullRecordID {
			// will be error in validateRow
			continue
		}
		if !id.IsRaw() {
			if !ev.Synced() {
				err = errors.Join(err,
					// event «sys.CUD» should use raw record ID (not «123456789012345») in created CRecord «Rec: test.CRecord»
					validateErrorf(ECode_InvalidRecordID, errRequiredRawID, ev, id, rec, ErrRawRecordIDRequired))
			}
		}
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				// event «sys.CUD» repeatedly uses record ID «1» in CRecord «CRec: test.CRecord»
				validateErrorf(ECode_InvalidRecordID, errRepeatedID, ev, id, rec, ErrRecordIDUniqueViolation))
		}
		ids[id] = rec.QName()

		if singleton, ok := rec.typ.(appdef.ISingleton); ok && singleton.Singleton() {
			if id, ok := st[singleton.QName()]; ok {
				err = errors.Join(err,
					// event «sys.CUD» repeatedly creates the same singleton «test.CDoc» (raw record ID «1» and «2»)
					validateErrorf(ECode_InvalidRecordID, errRepeatedSingletonCreation, ev, singleton, id, rec.id, ErrRecordIDUniqueViolation))
			}
			st[singleton.QName()] = rec.id
		}
	}

	for _, rec := range ev.cud.updates {
		id := rec.changes.ID()
		if id.IsRaw() {
			err = errors.Join(err,
				// event «sys.CUD» unexpectedly uses raw record ID «1» in updated CRRecord «CRec: test.CRecord»
				validateErrorf(ECode_InvalidRecordID, errUnexpectedRawID, ev, id, rec, ErrRawRecordIDUnexpected))
		}
		if _, exists := ids[id]; exists {
			err = errors.Join(err,
				// event «sys.CUD» repeatedly uses record ID «1» in CRecord «CRec: test.CRecord»
				validateErrorf(ECode_InvalidRecordID, errRepeatedID, ev, id, rec, ErrRecordIDUniqueViolation))
		}
		ids[id] = rec.changes.QName()
	}

	checkRefs := func(rec *recordType) (err error) {
		for name, id := range rec.RecordIDs(false) {
			target, ok := ids[id]
			if !ok {
				if id.IsRaw() {
					err = errors.Join(err,
						// WRecord «WRec: test.WRecord» field «Ref» refers to unknown record ID «7»
						validateErrorf(ECode_InvalidRefRecordID, errUnknownIDRef, rec, name, id, ErrRecordIDNotFound))
				}
				continue
			}
			fld := rec.fieldDef(name)
			if ref, ok := fld.(appdef.IRefField); ok {
				if !ref.Ref(target) {
					err = errors.Join(err,
						// WRecord «WRec: test.WRecord» field «Ref» refers to record ID «1» that has unavailable target QName «test.WDocument»
						validateErrorf(ECode_InvalidRefRecordID, errUnavailableTargetRef, rec, name, id, target, ErrWrongRecordID))
					continue
				}
			}
		}
		return err
	}

	for _, rec := range ev.cud.creates {
		parId := rec.Parent()
		if target, ok := ids[parId]; ok {
			if parentType, ok := ev.appCfg.AppDef.Type(target).(appdef.IContainers); ok {
				cont := parentType.Container(rec.Container())
				if cont == nil {
					err = errors.Join(err,
						// CRecord «CRec: test.CRecord» has parent ID «1» refers to «test.CDoc», which has no container «Record»
						validateErrorf(ECode_InvalidRefRecordID, errParentHasNoContainer, rec, parId, target, rec.Container(), ErrWrongRecordID))
					return
				}
				if cont.QName() != rec.QName() {
					err = errors.Join(err,
						// CRecord «Record: test.CRecord» has parent ID «1» refers to «test.CDoc», which container «Record» has another QName «test.CRecord1»
						validateErrorf(ECode_InvalidRefRecordID, errParentContainerOtherType, rec, parId, target, rec.Container(), cont.QName(), ErrWrongRecordID))
					return
				}
			}
		}

		err = errors.Join(err,
			checkRefs(rec))
	}

	for _, rec := range ev.cud.updates {
		err = errors.Join(err,
			checkRefs(&rec.changes))
	}

	return err
}

// # Validates event arguments.
//
// Checks that event argument and unlogged argument has correct type and content.
func validateEventArgs(ev *eventType) (err error) {
	arg, argUnl, err := ev.argumentNames()
	if err != nil {
		return validateError(ECode_InvalidTypeName, err)
	}

	if ev.argObject.QName() != arg {
		err = errors.Join(err,
			// event «test.document» argument uses wrong type «test.record1», expected «test.document»
			validateErrorf(ECode_InvalidTypeName, errEventArgUseWrongType, ev, ev.argObject.QName(), arg, ErrWrongType))
	} else if ev.argObject.QName() != appdef.NullQName {
		err = errors.Join(err,
			validateObject(&ev.argObject))
	}

	if ev.argUnlObj.QName() != argUnl {
		err = errors.Join(err,
			// event «test.document» unlogged argument uses wrong type «test.object», expected «.»
			validateErrorf(ECode_InvalidTypeName, errEventUnloggedArgUseWrongType, ev, ev.argUnlObj.QName(), argUnl, ErrWrongType))
	} else if ev.argUnlObj.QName() != appdef.NullQName {
		err = errors.Join(err,
			validateObject(&ev.argUnlObj))
	}

	return err
}

// Validates object with children in containers.
//
// Checks that all required fields is filled.
//
// Checks that min and max occurrences of containers is not violated.
//
// Checks that parent ID and container name is correct for children.
//
// Recursively validates children.
func validateObject(o *objectType) (err error) {
	err = validateRow(&o.rowType)

	t := o.typ.(appdef.IContainers)

	// validate occurrences
	for _, cont := range t.Containers() {
		occurs := appdef.Occurs(0)
		for range o.Children(cont.Name()) {
			occurs++
		}
		if occurs < cont.MinOccurs() {
			err = errors.Join(err,
				// ODoc «test.document» container «child» has not enough occurrences (0, minimum 1)
				validateErrorf(ECode_InvalidOccursMin, errContainerMinOccursViolated, o, cont.Name(), occurs, cont.MinOccurs(), ErrMinOccursViolation))
		}
		if occurs > cont.MaxOccurs() {
			err = errors.Join(err,
				// ODoc «test.document» container «child» has too many occurrences (2, maximum 1)
				validateErrorf(ECode_InvalidOccursMax, errContainerMaxOccursViolated, o, cont.Name(), occurs, cont.MaxOccurs(), ErrMaxOccursViolation))
		}
	}

	// validate children
	objID := o.ID()

	idx := -1
	o.allChildren(
		func(child *objectType) {
			idx++
			cont := t.Container(child.Container())
			if cont == nil {
				err = errors.Join(err,
					// ODoc «test.document» child[0] has unknown container name «child»
					validateErrorf(ECode_InvalidChildName, errUnknownContainerName, o, idx, child.Container(), ErrNameNotFound))
				return
			}

			childQName := child.QName()
			if childQName != cont.QName() {
				err = errors.Join(err,
					// ODoc «test.document» child[0] ORecord «child2: test.record1» has wrong type name, expected «test.record2»
					validateErrorf(ECode_InvalidTypeName, errWrongContainerType, o, idx, child, cont.QName(), ErrWrongType))
				return
			}

			if exists, required := child.typ.Kind().HasSystemField(appdef.SystemField_ParentID); exists {
				// ORecord, let's check parent ID
				parID := child.Parent()
				if parID == istructs.NullRecordID {
					if required {
						// if child parentID omitted, then restore it
						child.setParent(objID)
					}
				} else {
					if parID != objID {
						err = errors.Join(err,
							// ODoc «test.document» child[0] ORecord «child: test.record1» has wrong parent id «2», expected «1»
							validateErrorf(ECode_InvalidRefRecordID, errWrongParentID, o, idx, child, parID, objID, ErrWrongRecordID))
					}
				}
			}

			err = errors.Join(err,
				validateObject(child)) // recursive call
		})

	return err
}

// Validates row fields.
//
// Checks that all required fields are filled.
// For required ref fields checks that they are filled with non null IDs.
func validateRow(row *rowType) (err error) {
	for _, f := range row.fields.Fields() {
		if f.Required() {
			if !row.HasValue(f.Name()) {
				err = errors.Join(err,
					// ODoc «test.document» misses required field «RequiredField»
					validateErrorf(ECode_EmptyData, errEmptyRequiredField, row, f.Name(), ErrNameNotFound))
				continue
			}
			if !f.IsSys() {
				switch f.DataKind() {
				case appdef.DataKind_RecordID:
					if row.AsRecordID(f.Name()) == istructs.NullRecordID {
						err = errors.Join(err,
							// ORecord «child2: test.record2» required ref field «RequiredRefField» has NullRecordID value
							validateErrorf(ECode_InvalidRefRecordID, errNullInRequiredRefField, row, f.Name(), ErrWrongRecordID))
					}
				}
			}
		}
	}
	return err
}

// Validate event CUDs.
//
// Checks that sys.CUD command has not empty CUDs.
//
// Checks that CUDs has correct type and content.
func validateEventCUDs(ev *eventType) (err error) {
	if ev.cud.empty() {
		if ev.name == istructs.QNameCommandCUD {
			return validateErrorf(ECode_EEmptyCUDs, errCUDsMissed, ev, ErrCUDsMissed)
		}
		return nil
	}

	if len(ev.cud.creates) > math.MaxUint16 {
		return validateErrorf(ECode_TooManyCreates, "creates number must not be more than %d", math.MaxUint16)
	}

	if len(ev.cud.updates) > math.MaxUint16 {
		return validateErrorf(ECode_TooManyUpdates, "updates number must not be more than %d", math.MaxUint16)
	}

	for _, rec := range ev.cud.creates {
		err = errors.Join(err,
			validateEventCUD(ev, rec, "Create"))
	}

	for _, rec := range ev.cud.updates {
		err = errors.Join(err,
			validateEventCUD(ev, &rec.result, "Update"))
	}

	return err
}

// Validates specified CUD record.
//
// Checks that CUD record has correct (storable) type and content.
func validateEventCUD(ev *eventType, rec *recordType, part string) error {
	switch rec.typ.Kind() {
	case appdef.TypeKind_GDoc, appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_GRecord, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		return validateRow(&rec.rowType)
	default:
		// event «sys.CUD» CUD.Create() [record ID «1»] ORec «test.ORecord» has invalid type kind: %w"
		return validateErrorf(ECode_InvalidTypeKind, errInvalidTypeKindInCUD, ev, part, rec.ID(), rec, ErrUnexpectedTypeKind)
	}
}

// # Validates specified view key.
//
// If partialClust specified then clustering columns row may be partially filled
func validateViewKey(key *keyType, partialClust bool) (err error) {
	for _, f := range key.partRow.fields.Fields() {
		if !key.partRow.HasValue(f.Name()) {
			err = errors.Join(err,
				validateErrorf(ECode_EmptyData, "view «%v» partition key field «%s» is empty: %w", key.viewName, f.Name(), ErrFieldIsEmpty))
		}
	}

	ccFields := key.ccolsRow.fields.Fields()
	if partialClust {
		for i, f := range ccFields {
			fName := f.Name()
			if !key.ccolsRow.HasValue(fName) {
				for j := i + 1; j < len(ccFields); j++ {
					if key.ccolsRow.HasValue(ccFields[j].Name()) {
						err = errors.Join(err,
							validateErrorf(ECode_EmptyData, "view «%v» clustering columns has a hole at field «%s»: %w", key.viewName, fName, ErrFieldIsEmpty))
						break
					}
				}
				break
			}
		}
	} else {
		for _, f := range ccFields {
			if !key.ccolsRow.HasValue(f.Name()) {
				err = errors.Join(err,
					validateErrorf(ECode_EmptyData, "view «%v» clustering columns field «%s» is empty: %w", key.viewName, f.Name(), ErrFieldIsEmpty))
			}
		}
	}

	return err
}

// # Validates specified view value
func validateViewValue(value *valueType) (err error) {
	return validateRow(&value.rowType)
}
