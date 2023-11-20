/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideApplyUniques(appDef appdef.IAppDef) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		return event.CUDs(func(rec istructs.ICUDRow) (err error) {
			if unique, ok := appDef.Type(rec.QName()).(appdef.IUniques); ok {
				if uniqueField := unique.UniqueField(); uniqueField != nil {
					uniqueFieldIsNotNull, _, _ := iterate.FindFirstMap(rec.ModifiedFields, func(s string, _ interface{}) bool {
						return s == uniqueField.Name()
					})
					// update and uniqueFieldIsNotNull -> we're setting the value if the unique field for the first time. In this case let's insert.
					// update and uniqueFieldIsNotNull and had a value already -> impossible here, denied by the validator already
					if rec.IsNew() {
						// insert regardless uniqueFieldIsNotNull -> insert with an empty value is the same as if value is 0
						// so will not insert if the value is not provided
						if uniqueFieldIsNotNull {
							return insert(rec, uniqueField, st, intents)
						}
					} else {
						// came here in 2 cases:
						// - we're updating fields that are not part of an unique key (e.g. sys.IsActive)
						// - we're setting the value for the unqiue field for the first time

						if uniqueFieldIsNotNull {
							// update and the value is provided -> we're setting the value for the first time. Let's insert
							// update the value that is already set -> denied by the validator already
							return insert(rec, uniqueField, st, intents)
						}

						// came here -> unique field has no value in the request
						// let's check if the value is inited already
						// inited already -> update, need to check if we're activating\deactivating
						// not inited yet -> has no record in the view, we're updating other fields -> do nothing
						storedUniqueFieldHasValue, err := storedUniqueFieldHasValue(st, rec, uniqueField)
						if err != nil {
							// notest
							return nil
						}
						if storedUniqueFieldHasValue {
							// has record in the view and the value is not provided in the request -> probably we're updating sys.IsActive
							return update(rec, uniqueField, st, intents)
						}
						// no unique field value in the request and was not inited -> do nothing
						return nil
					}
				}
			}
			return nil
		})
	}
}

func storedUniqueFieldHasValue(st istructs.IState, rec istructs.ICUDRow, uniqueField appdef.IField) (bool, error) {
	kb, err := st.KeyBuilder(state.Record, rec.QName())
	if err != nil {
		// notest
		return false, err
	}
	kb.PutRecordID(state.Field_ID, rec.ID())
	storageRec, err := st.MustExist(kb)
	if err != nil {
		// notest
		return false, err
	}
	storedUniqueFieldHasValue, _ := iterate.FindFirst(storageRec.FieldNames, func(storedUniqueFieldNameThatHasValue string) bool {
		return storedUniqueFieldNameThatHasValue == uniqueField.Name()
	})
	return storedUniqueFieldHasValue, nil
}

func insert(rec istructs.ICUDRow, uf appdef.IField, state istructs.IState, intents istructs.IIntents) error {
	uniqueViewRecord, uniqueViewKB, ok, err := getUniqueViewRecord(state, rec, uf)
	if err != nil {
		return err
	}

	var uniqueViewRecordBuilder istructs.IStateValueBuilder
	recIsActive := rec.AsBool(appdef.SystemField_IsActive)
	if !ok {
		if uniqueViewRecordBuilder, err = intents.NewValue(uniqueViewKB); err != nil {
			return err
		}
		cudID := rec.ID()
		if !recIsActive {
			cudID = istructs.NullRecordID
		}
		uniqueViewRecordBuilder.PutRecordID(field_ID, cudID)
	} else {
		if !recIsActive {
			// inserting an inactive record whereas unique exists -> allow, nothing to do
			return nil
		}
		if uniqueViewRecord.AsRecordID(field_ID) == istructs.NullRecordID {
			// insering an active record whereas unique exists for a deactivated record -> allow + update view
			if uniqueViewRecordBuilder, err = intents.UpdateValue(uniqueViewKB, uniqueViewRecord); err != nil {
				return err
			}
			uniqueViewRecordBuilder.PutRecordID(field_ID, rec.ID())
		}
		// note: inserting a new active record whereas uniques exists already - this case is denied already by the validator
		// note: uvrID == recID - impossible, recID can not deuplicate
	}
	return err
}

func update(rec istructs.ICUDRow, uf appdef.IField, st istructs.IState, intents istructs.IIntents) error {
	// read current record
	kb, err := st.KeyBuilder(state.Record, rec.QName())
	if err != nil {
		return err
	}
	kb.PutRecordID(state.Field_ID, rec.ID())
	currentRecord, err := st.MustExist(kb)
	if err != nil {
		return err
	}

	// unique view record
	uniqueViewRecord, uniqueViewKB, _, err := getUniqueViewRecord(st, currentRecord, uf)
	if err != nil {
		return err
	}

	// !uniqueViewRecord.Exists() - impossible: record was inserted -> an unique exists
	// now possible: inserted a record that has no value for a unique field -> no unique at all

	viewValueID := istructs.NullRecordID
	uvrID := uniqueViewRecord.AsRecordID(field_ID)
	if rec.AsBool(appdef.SystemField_IsActive) {
		if uvrID == istructs.NullRecordID {
			// activating the record whereas previous combination was deactivated -> allow, update view
			viewValueID = rec.ID()
		} else {
			// activating the already activated record, unique combination exists for that record -> allow, nothing to do
			return nil
			// note: case when uvrID == rec.ID() is handled already by validator
		}
	} else {
		if rec.ID() != uvrID {
			// deactivating a record whereas unique combination exists for another record -> allow, nothing to do
			return nil
		}
	}
	uniqueViewRecordBuilder, err := intents.UpdateValue(uniqueViewKB, uniqueViewRecord)
	if err != nil {
		return err
	}
	uniqueViewRecordBuilder.PutRecordID(field_ID, viewValueID)
	return nil
}

func getUniqueViewRecord(st istructs.IState, rec istructs.IRowReader, uf appdef.IField) (istructs.IStateValue, istructs.IStateKeyBuilder, bool, error) {
	kb, err := st.KeyBuilder(state.View, qNameViewUniques)
	if err != nil {
		return nil, nil, false, err
	}
	if err := buildUniqueViewKey(kb, rec, uf); err != nil {
		// notest
		return nil, nil, false, err
	}
	sv, ok, err := st.CanExist(kb)
	return sv, kb, ok, err
}

func getUniqueKeyValues(rec istructs.IRowReader, uf appdef.IField) (res []byte, err error) {
	buf := bytes.NewBuffer(nil)

	val := coreutils.ReadByKind(uf.Name(), uf.DataKind(), rec)
	switch uf.DataKind() {
	case appdef.DataKind_string, appdef.DataKind_raw:
		_, err = buf.WriteString(val.(string))
	case appdef.DataKind_bytes:
		_, err = buf.Write(val.([]byte))
	default:
		err = binary.Write(buf, binary.BigEndian, val)
	}

	return buf.Bytes(), err
}

// notest err
func buildUniqueViewKeyByValues(kb istructs.IKeyBuilder, qName appdef.QName, uniqueKeyValues []byte) error {
	kb.PutQName(field_QName, qName)
	kb.PutInt64(field_ValuesHash, coreutils.HashBytes(uniqueKeyValues))
	kb.PutBytes(field_Values, uniqueKeyValues)
	return nil
}

// notest err
func buildUniqueViewKey(kb istructs.IKeyBuilder, rec istructs.IRowReader, uf appdef.IField) error {
	uniqueKeyValues, err := getUniqueKeyValues(rec, uf)
	if err != nil {
		// notest
		return err
	}
	return buildUniqueViewKeyByValues(kb, rec.AsQName(appdef.SystemField_QName), uniqueKeyValues)
}

type uniqueViewRecord struct {
	exists      bool
	refRecordID istructs.RecordID
}

func provideEventUniqueValidator() func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	return func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
		//                                      key         uvrID
		uniquesState := map[appdef.QName]map[string]*uniqueViewRecord{}
		err := rawEvent.CUDs(func(cudRec istructs.ICUDRow) (err error) {
			qName := cudRec.QName()
			if uniques, ok := appStructs.AppDef().Type(qName).(appdef.IUniques); ok {
				if uniqueField := uniques.UniqueField(); uniqueField != nil {
					cudUniqueFieldHasValue, _ := iterate.FindFirst(cudRec.FieldNames, func(fieldNameThatHasValue string) bool {
						return fieldNameThatHasValue == uniqueField.Name()
					})
					var uniqueKeyValues []byte
					sourceRow := cudRec.(istructs.IRecord)
					if cudRec.IsNew() {
						if !cudUniqueFieldHasValue {
							// insert a new record, no unique field value -> do nothing
							return nil
						}
					} else {
						// need to check if unique combination exists already
						storedRow, err := appStructs.Records().Get(wsid, true, cudRec.ID())
						if err != nil {
							// notest
							return err
						}
						storedUniqueFieldHasValue, _ := iterate.FindFirst(storedRow.FieldNames, func(uniqueFieldNameThatHasStoredValue string) bool {
							return uniqueFieldNameThatHasStoredValue == uniqueField.Name()
						})
						if !storedUniqueFieldHasValue && !cudUniqueFieldHasValue {
							// had no unique field value before and update something else _. nothing to do
							return nil
						}
						if storedUniqueFieldHasValue {
							sourceRow = storedRow
						}
					}
					uniqueKeyValues, err = getUniqueKeyValues(sourceRow, uniqueField)
					if err != nil {
						// notest
						return err
					}

					currentUniqueRecord, err := getCurrentUniqueViewRecord(uniquesState, qName, uniqueKeyValues, appStructs, wsid)
					if err != nil {
						// notest
						return err
					}

					if cudRec.IsNew() {
						if cudRec.AsBool(appdef.SystemField_IsActive) {
							if currentUniqueRecord.refRecordID == istructs.NullRecordID {
								// inserting a new active record, unique is inactive -> allowed, update its ID in map
								currentUniqueRecord.refRecordID = cudRec.ID()
								currentUniqueRecord.exists = true // avoid: 1st CUD insert a unique record, 2nd modify the unique value
							} else {
								// inserting a new active record, unique is active -> deny
								return conflict(qName, currentUniqueRecord.refRecordID)
							}
						} else {
							if cudUniqueFieldHasValue {
								currentUniqueRecord.exists = true
								currentUniqueRecord.refRecordID = istructs.NullRecordID
							}
							// insert an inactive record, no unique value -> allow, do nothing
						}
					} else {
						cudRecIsActive := cudRec.AsBool(appdef.SystemField_IsActive)
						if currentUniqueRecord.exists {
							if cudUniqueFieldHasValue && (currentUniqueRecord.refRecordID == cudRec.ID() || currentUniqueRecord.refRecordID == istructs.NullRecordID) {
								return fmt.Errorf("%v: unique field «%s» can not be changed: %w", qName, uniqueField.Name(), ErrUniqueFieldUpdateDeny)
							}
							if currentUniqueRecord.refRecordID == istructs.NullRecordID {
								if cudRecIsActive {
									currentUniqueRecord.refRecordID = cudRec.ID()
								}
							} else {
								if currentUniqueRecord.refRecordID == cudRec.ID() {
									if !cudRecIsActive {
										currentUniqueRecord.refRecordID = istructs.NullRecordID
									}
								} else {
									if cudRecIsActive {
										return conflict(qName, currentUniqueRecord.refRecordID)
									}
								}
							}
						} else {
							// update, no unique view record
							if cudRecIsActive && cudUniqueFieldHasValue {
								currentUniqueRecord.exists = true
								currentUniqueRecord.refRecordID = cudRec.ID()
							}
						}
					}
				}
			}
			return nil
		})
		return err
	}
}

func getCurrentUniqueViewRecord(uniquesState map[appdef.QName]map[string]*uniqueViewRecord, qName appdef.QName, uniqueKeyValues []byte, appStructs istructs.IAppStructs, wsid istructs.WSID) (*uniqueViewRecord, error) {
	// why to accumulate in a map?
	//         id:  field: IsActive: Result:
	// stored: 111: xxx    -
	// …
	// cud(I): 222: xxx    +         - should be ok to insert new record
	// …
	// cud(J): 111:        +         - should be denied to restore old record
	qNameEventUniques, ok := uniquesState[qName]
	if !ok {
		qNameEventUniques = map[string]*uniqueViewRecord{}
		uniquesState[qName] = qNameEventUniques
	}
	currentUniqueRecord, ok := qNameEventUniques[string(uniqueKeyValues)]
	if !ok {
		currentUniqueRecordID, uniqueViewRecordExists, err := getUniqueIDByValues(appStructs, wsid, qName, uniqueKeyValues)
		if err != nil {
			return nil, err
		}
		currentUniqueRecord = &uniqueViewRecord{
			exists:      uniqueViewRecordExists,
			refRecordID: currentUniqueRecordID,
		}
		qNameEventUniques[string(uniqueKeyValues)] = currentUniqueRecord
	}
	return currentUniqueRecord, nil
}

func conflict(qName appdef.QName, conflictingWithID istructs.RecordID) error {
	return coreutils.NewHTTPError(http.StatusConflict, fmt.Errorf("%s: %w with ID %d", qName, ErrUniqueConstraintViolation, conflictingWithID))
}
