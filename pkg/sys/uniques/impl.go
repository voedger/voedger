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

func provideUniquesProjectorFunc(appDef appdef.IAppDef) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		return event.CUDs(func(rec istructs.ICUDRow) (err error) {
			if unique, ok := appDef.Def(rec.QName()).(appdef.IUniques); ok {
				if uniqueField := unique.UniqueField(); uniqueField != nil {
					uniqueFieldIsNotNull, _, _ := iterate.FindFirstMap(rec.ModifiedFields, func(s string, _ interface{}) bool {
						return s == uniqueField.Name()
					})
					// update and uniqueFieldIsNotNull -> we're setting the value if the unique field for the first time. In this case let's insert.
					// update and uniqueFieldIsNotNull and had a value already -> impossible here, denied by the validator already
					if rec.IsNew() {
						// check if unique field is null
						// вставлять в любом случае плохо, т.к. вставили с пустым значением, вставили с нулем - конфликт, т.к. во view записались нули, когда вставляли пустое знаечение
						// не будем вставлять, если поле не предоставлено
						if uniqueFieldIsNotNull {
							return insert(rec, uniqueField, st, intents)
						}
					} else {
						// came here -> we're updating fields that are not part of an unique key
						// because it is impossible to modify the value of an unique field according to the current design
						// e.g. updating sys.IsActive
						// but probably we're setting the value for the unqiue field for the first time. In this case let's insert.

						// читать как-то неправильно, т.к. тут мы уже только что применили изменения
						// если мы обновляем и есть значение, то значит мы устанавливаем его впервые, значит insert. Обновляем uniqe field, которое уже имело значение - уже запрещено валидатором
						if uniqueFieldIsNotNull {
							return insert(rec, uniqueField, st, intents)
						}

						// попали сюда -> уникальное поле не имеет значения в запросе
						// надо проверить, а было ли значение раньше
						// если было, то надо update. А если не было - тогда мы обновляем что-то, что не относится к unqiues -> ничего не делаем
						// тогда только update
						// а если записи все-таки не было вообще? тогда ничего не делаем
						// return update(rec, uniqueField, st, intents)

						kb, err := st.KeyBuilder(state.RecordsStorage, rec.QName())
						if err != nil {
							// notest
							return err
						}
						kb.PutRecordID(state.Field_ID, rec.ID())
						storageRec, err := st.MustExist(kb)
						if err != nil {
							// notest
							return err
						}
						storedUniqueFieldHasValue, _ := iterate.FindFirst(storageRec.FieldNames, func(storedUniqueFieldNameThatHasValue string) bool {
							return storedUniqueFieldNameThatHasValue == uniqueField.Name()
						})

						if storedUniqueFieldHasValue {
							// нет значения и было раньше -> возможно, мы обнволяем sys.IsActive
							return update(rec, uniqueField, st, intents)
						}
						// нет значения и не было раньше -> ничего не делаем
						return nil

						// if !storedUniqueFieldHasValue && uniqueFieldIsNotNull {
						// 	return insert(rec, uniqueField, st, intents)
						// }
						// if !storedUniqueFieldHasValue && !uniqueFieldIsNotNull {
						// 	// unique field value is not inited && we're not initing it -> so nothing
						// 	return nil
						// }
						// return update(rec, uniqueField, st, intents)
					}
				}
			}
			return nil
		})
	}
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
	kb, err := st.KeyBuilder(state.RecordsStorage, rec.QName())
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
	kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewUniques)
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
	case appdef.DataKind_string:
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

// func provideCUDUniqueUpdateDenyValidator() func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) error {
// 	return func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
// 		if cudRow.IsNew() {
// 			return nil
// 		}
// 		qName := cudRow.QName()
// 		if uniques, ok := appStructs.AppDef().Def(qName).(appdef.IUniques); ok {
// 			if uniqueField := uniques.UniqueField(); uniqueField != nil {
// 				var storageError error
// 				var storedRow istructs.IRowReader
// 				cudRow.ModifiedFields(func(fieldName string, newValue interface{}) {
// 					if storageError != nil {
// 						// notest
// 						return
// 					}
// 					if fieldName == uniqueField.Name() {
// 						// read the stored record
// 						if storedRow == nil {
// 							storedRow, storageError = appStructs.Records().Get(wsid, true, cudRow.ID())
// 							if storageError != nil {
// 								// notest
// 								return
// 							}
// 						}
// 						uniqueFieldHasStoredValue, _ := iterate.FindFirst(storedRow.FieldNames, func(storedUniqueFieldNameThatHasValue string) bool {
// 							return storedUniqueFieldNameThatHasValue == uniqueField.Name()
// 						})
// 						// no value for the unique field yet -> allow to set the value
// 						if uniqueFieldHasStoredValue {
// 							err = errors.Join(err,
// 								fmt.Errorf("%v: unique field «%s» can not to be changed: %w", qName, fieldName, ErrUniqueFieldUpdateDeny))
// 						}
// 					}
// 				})
// 			}
// 		}
// 		return err
// 	}
// }

type uniqueViewRecord struct {
	exists      bool
	refRecordID istructs.RecordID
}

func provideEventUniqueValidator() func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	return func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
		//                                      key         uvrID
		uniquesState := map[appdef.QName]map[string]*uniqueViewRecord{}
		err := rawEvent.CUDs(func(cudRec istructs.ICUDRow) (err error) {

			// } else {
			// 	if actualRow, err = appStructs.Records().Get(wsid, true, rec.ID()); err != nil { // read current record
			// 		return err
			// 	}
			// }
			qName := cudRec.QName()
			if uniques, ok := appStructs.AppDef().Def(qName).(appdef.IUniques); ok {
				if uniqueField := uniques.UniqueField(); uniqueField != nil {
					cudUniqueFieldHasValue, _ := iterate.FindFirst(cudRec.FieldNames, func(fieldNameThatHasValue string) bool {
						return fieldNameThatHasValue == uniqueField.Name()
					})
					var uniqueKeyValues []byte
					var existingRow istructs.IRecord
					if cudRec.IsNew() {
						if !cudUniqueFieldHasValue {
							return nil
						}
						uniqueKeyValues, err = getUniqueKeyValues(cudRec, uniqueField)
						if err != nil {
							// notest
							return err
						}
					} else {
						/*
							что если мы обновляем строку, но не устанавливаем уникальность и ее раньше не было?
							как узнать, что ее раньше не было?
						*/
						existingRow, err = appStructs.Records().Get(wsid, true, cudRec.ID())
						if err != nil { // need to check activate\deactivate, read current record
							// notest
							return err
						}
						storedUniqueFieldHasValue, _ := iterate.FindFirst(existingRow.FieldNames, func(uniqueFieldNameThatHasStoredValue string) bool {
							return uniqueFieldNameThatHasStoredValue == uniqueField.Name()
						})
						sourceRow := cudRec.(istructs.IRecord)
						if storedUniqueFieldHasValue {
							sourceRow = existingRow
						}
						if !storedUniqueFieldHasValue && !cudUniqueFieldHasValue {
							// had no unique field value before and update something else _. nothing to do
							return nil
						}
						uniqueKeyValues, err = getUniqueKeyValues(sourceRow, uniqueField)
						if err != nil {
							// notest
							return err
						}
					}
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
							return err
						}
						currentUniqueRecord = &uniqueViewRecord{
							exists:      uniqueViewRecordExists,
							refRecordID: currentUniqueRecordID,
						}
						qNameEventUniques[string(uniqueKeyValues)] = currentUniqueRecord
					}

					if cudRec.IsNew() {
						if cudRec.AsBool(appdef.SystemField_IsActive) {
							if currentUniqueRecord.refRecordID == istructs.NullRecordID {
								// inserting a new active record, unique is inactive -> allowed, update its ID in map
								// qNameEventUniques[string(cudUniqueKeyValues)] = rec.ID()
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
							if cudUniqueFieldHasValue {
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

func conflict(qName appdef.QName, conflictingWithID istructs.RecordID) error {
	return coreutils.NewHTTPError(http.StatusConflict, fmt.Errorf("%s: %w with ID %d", qName, ErrUniqueConstraintViolation, conflictingWithID))
}
