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
func validateObjectIDs(obj *objectType, rawID bool) (ids map[istructs.RecordID]*rowType, err error) {
	ids = make(map[istructs.RecordID]*rowType)

	_ = obj.forEach(func(child *objectType) error {
		id := child.ID()
		if id == istructs.NullRecordID {
			return nil
		}
		if !id.IsRaw() {
			if rawID {
				err = errors.Join(err,
					// ODoc «test.document» sys.ID: id «123456789012345» is not raw
					validateError(ECode_InvalidRecordID,
						ErrRawRecordIDRequired(obj, appdef.SystemField_ID, id)))
			}
		}
		if exists, ok := ids[id]; ok {
			err = errors.Join(err,
				// id «1» used by %v and %v
				ErrRecordIDUniqueViolation(id, exists, child))
		}
		ids[id] = &child.rowType
		return nil
	})

	_ = obj.forEach(func(e *objectType) error {
		for _, fld := range e.fields.RefFields() {
			if id := e.AsRecordID(fld.Name()); id != istructs.NullRecordID {
				target, exists := ids[id]
				if !exists {
					if id.IsRaw() {
						err = errors.Join(err,
							// ODoc «test.document» field «RefField» refers to unknown ID «7»
							validateError(ECode_InvalidRefRecordID, ErrRefIDNotFound(e, fld.Name(), id)))
					}
					continue
				}
				if !fld.Ref(target.QName()) {
					err = errors.Join(err,
						// ODoc «test.document» field «RefField» refers to record ID «1» that has unavailable target ODoc «test.document»
						validateError(ECode_InvalidRefRecordID, ErrWrongRecordIDTarget(e, fld, id, target)))
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
func validateEventCUDsIDs(ev *eventType, ids map[istructs.RecordID]*rowType) (err error) {
	st := make(map[appdef.QName]bool) // singletons unique

	for _, rec := range ev.cud.creates {
		id := rec.ID()
		if id == istructs.NullRecordID {
			// will be error in validateRow
			continue
		}
		if !id.IsRaw() {
			if !ev.Synced() {
				err = errors.Join(err,
					// CDoc «test.document» sys.ID: id «123456789012345» is not raw
					validateError(ECode_InvalidRecordID,
						ErrRawRecordIDRequired(rec, appdef.SystemField_ID, id)))
			}
		}
		if exists, ok := ids[id]; ok {
			err = errors.Join(err,
				// id «1» used by %v and %v
				ErrRecordIDUniqueViolation(id, exists, rec))
		}
		ids[id] = &rec.rowType

		if singleton, ok := rec.typ.(appdef.ISingleton); ok && singleton.Singleton() {
			if _, violated := st[singleton.QName()]; violated {
				err = errors.Join(err,
					// id «%d» used by %v and %v
					validateError(ECode_InvalidRecordID, ErrSingletonViolation(singleton)))
			}
			st[singleton.QName()] = true
		}
	}

	for _, rec := range ev.cud.updates {
		id := rec.changes.ID()
		if id.IsRaw() {
			err = errors.Join(err,
				// updated CRecord «test.CRecord» sys.ID: id «1» should not be raw
				validateError(ECode_InvalidRecordID,
					ErrUnexpectedRawRecordID(rec, appdef.SystemField_ID, id)))
		}
		if exists, violated := ids[id]; violated {
			err = errors.Join(err,
				// id «%d» used by %v and %v
				validateError(ECode_InvalidRecordID,
					ErrRecordIDUniqueViolation(id, exists, rec)))
		}
		ids[id] = &rec.changes.rowType
	}

	checkRefs := func(rec *recordType) (err error) {
		for name, id := range rec.RecordIDs(false) {
			target, ok := ids[id]
			if !ok {
				if id.IsRaw() {
					err = errors.Join(err,
						// WRecord «WRec: test.WRecord» field «Ref» refers to unknown ID «7»
						validateError(ECode_InvalidRefRecordID, ErrRefIDNotFound(rec, name, id)))
				}
				continue
			}
			fld := rec.fieldDef(name)
			if ref, ok := fld.(appdef.IRefField); ok {
				if !ref.Ref(target.QName()) {
					err = errors.Join(err,
						// WRecord «WRec: test.WRecord» field «Ref» refers to record ID «1» that has unavailable target WDoc «test.WDocument»
						validateError(ECode_InvalidRefRecordID, ErrWrongRecordIDTarget(rec, ref, id, target)))
					continue
				}
			}
		}
		return err
	}

	for _, rec := range ev.cud.creates {
		parId := rec.Parent()
		if target, ok := ids[parId]; ok {
			if parentType, ok := ev.appCfg.AppDef.Type(target.QName()).(appdef.IWithContainers); ok {
				cont := parentType.Container(rec.Container())
				if cont == nil {
					err = errors.Join(err,
						// CRecord «CRec: test.CRecord» has parent ID «1» refers to CDoc «test.CDoc», which has no container «Record»
						validateError(ECode_InvalidRefRecordID,
							ErrWrongRecordID("%v has parent ID «%d» refers to %v, which has no container «%s»", rec, parId, target, rec.Container())))
					return
				}
				if cont.QName() != rec.QName() {
					err = errors.Join(err,
						// CRecord «Record: test.CRecord» has parent ID «1» refers to CDoc «test.CDoc», which container «Record» has another QName «test.CRecord1»
						validateError(ECode_InvalidRefRecordID,
							ErrWrongRecordID("%v has parent ID «%d» refers to %s, which container «%s» has another QName «%s»", rec, parId, target, rec.Container(), cont.QName())))
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
			validateErrorf(ECode_InvalidTypeName, errEventArgUseWrongType, ev, ev.argObject.QName(), arg, ErrWrongTypeError))
	} else if ev.argObject.QName() != appdef.NullQName {
		err = errors.Join(err,
			validateObject(&ev.argObject))
	}

	if ev.argUnlObj.QName() != argUnl {
		err = errors.Join(err,
			// event «test.document» unlogged argument uses wrong type «test.object», expected «.»
			validateErrorf(ECode_InvalidTypeName, errEventUnloggedArgUseWrongType, ev, ev.argUnlObj.QName(), argUnl, ErrWrongTypeError))
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

	t := o.typ.(appdef.IWithContainers)

	// validate occurrences
	for _, cont := range t.Containers() {
		n, occurs := cont.Name(), appdef.Occurs(0)
		for range o.Children(n) {
			occurs++
		}
		if minO := cont.MinOccurs(); occurs < minO {
			err = errors.Join(err,
				// ODoc «test.document» container «child» has not enough occurrences (0, minimum 1)
				validateError(ECode_InvalidOccursMin, ErrMinOccursViolated(o, n, occurs, minO)))
		}
		if maxO := cont.MaxOccurs(); occurs > maxO {
			err = errors.Join(err,
				// ODoc «test.document» container «child» has too many occurrences (2, maximum 1)
				validateError(ECode_InvalidOccursMax, ErrMaxOccursViolated(o, n, occurs, maxO)))
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
					validateError(ECode_InvalidChildName, ErrContainerNotFound(child.Container(), o)))
				return
			}

			childQName := child.QName()
			if childQName != cont.QName() {
				err = errors.Join(err,
					// ODoc «test.document» child[0] ORecord «child2: test.record1» has wrong type name, expected «test.record2»
					validateErrorf(ECode_InvalidTypeName, errWrongContainerType, o, idx, child, cont.QName(), ErrWrongTypeError))
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
							validateError(ECode_InvalidRefRecordID,
								ErrWrongRecordID("%v child[%d] %v has wrong parent id «%d», expected «%d»", o, idx, child, parID, objID)))
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
					// field is empty: ODoc «test.document» RequiredField
					validateError(ECode_EmptyData, ErrFieldMissed(row, f)))
				continue
			}
			if !f.IsSys() {
				switch f.DataKind() {
				case appdef.DataKind_RecordID:
					if row.AsRecordID(f.Name()) == istructs.NullRecordID {
						err = errors.Join(err,
							// ORecord «child2: test.record2» required ref field «RequiredRefField» has NullRecordID value
							validateError(ECode_InvalidRefRecordID,
								ErrWrongRecordID("%v required ref field «%s» has NullRecordID value", row, f.Name())))
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
			return validateError(ECode_EEmptyCUDs, ErrCUDsMissed(ev))
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
			validateEventCUD(ev, rec))
	}

	for _, rec := range ev.cud.updates {
		err = errors.Join(err,
			validateEventCUD(ev, &rec.result))
	}

	return err
}

// Validates specified CUD record.
//
// Checks that CUD record has correct (storable) type and content.
func validateEventCUD(ev *eventType, rec *recordType) error {
	switch k := rec.typ.Kind(); k {
	case appdef.TypeKind_GDoc, appdef.TypeKind_ODoc, appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_GRecord, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		return validateRow(&rec.rowType)
	default:
		return validateError(ECode_InvalidTypeKind,
			ErrUnexpectedType("%v in %v CUDs", rec, ev))
	}
}

// # Validates specified view key.
//
// If partialClust specified then clustering columns row may be partially filled
func validateViewKey(key *keyType, partialClust bool) (err error) {
	for _, f := range key.partRow.fields.Fields() {
		if !key.partRow.HasValue(f.Name()) {
			err = errors.Join(err,
				validateError(ECode_EmptyData, ErrFieldMissed(key, f)))
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
							validateError(ECode_EmptyData,
								enrichError(ErrFieldIsEmptyError, "%v has a hole at field «%s»", key, fName)))
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
					validateError(ECode_EmptyData, ErrFieldMissed(key, f)))
			}
		}
	}

	return err
}

// # Validates specified view value
func validateViewValue(value *valueType) (err error) {
	return validateRow(&value.rowType)
}
