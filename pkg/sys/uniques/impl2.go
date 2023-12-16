/*
 * Copyright (c) 2020-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package uniques

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideApplyUniques2(appDef appdef.IAppDef) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		return iterate.ForEachError(event.CUDs, func(rec istructs.ICUDRow) error {
			iUniques, ok := appDef.Type(rec.QName()).(appdef.IUniques)
			if !ok {
				return nil
			}
			err := iterate.ForEachError(iUniques.Uniques, func(unique appdef.IUnique) error {
				orderedUniqueFields := getOrderedUniqueFields(appDef, rec, unique)
				if rec.IsNew() {
					return insert2(st, rec, intents, orderedUniqueFields)
				}
				return update2(st, rec, orderedUniqueFields)
			})
			return err
		})
	}
}

func getOrderedUniqueFields(appDef appdef.IAppDef, rec istructs.ICUDRow, unique appdef.IUnique) (orderedUniqueFields orderedUniqueFields) {
	recType := appDef.Type(rec.QName())
	recSchemaFields := recType.(appdef.IFields)
	recSchemaFields.Fields(func(schemaField appdef.IField) {
		for _, uniqueFieldDesc := range unique.Fields() {
			if uniqueFieldDesc.Name() == schemaField.Name() {
				orderedUniqueFields = append(orderedUniqueFields, schemaField)
			}
		}
	})
	return orderedUniqueFields
}

func update2(st istructs.IState, rec istructs.ICUDRow, orderedUniqueFields orderedUniqueFields) error {
	// check modified fields
	// case when we're updating unique fields is already dropped by the validator
	// so came here -> we're updating anything but unique fields
	// let's check activation\deactivation

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
	// we're updating -> unique view record exists
	uniqueViewRecord, uniqueViewKB, _, err := getUniqueViewRecord2(st, currentRecord, orderedUniqueFields)
	if err != nil {
		return err
	}
	if uniqueViewRecord.AsRecordID(field_ID) == istructs.NullRecordID && rec.AsBool(appdef.SystemField_IsActive) {
		// activate a deactivated combination
		uniqueViewKB.PutRecordID(field_ID, rec.ID())
	} else if uniqueViewRecord.AsRecordID(field_ID) != istructs.NullRecordID && !rec.AsBool(appdef.SystemField_IsActive) {
		// deactivate an active combination
		uniqueViewKB.PutRecordID(field_ID, istructs.NullRecordID)
	}
	return nil
}

func insert2(state istructs.IState, rec istructs.ICUDRow, intents istructs.IIntents, orderedUniqueFields orderedUniqueFields) error {
	uniqueViewRecord, uniqueViewKB, uniqueViewRecordExists, err := getUniqueViewRecord2(state, rec, orderedUniqueFields)
	if err != nil {
		return err
	}
	// no scenario whe we're inserting a deactivated record
	var uniqueViewRecordBuilder istructs.IStateValueBuilder
	if uniqueViewRecordExists {
		// the olny possible case here - we're inserting a new record, the view record exists for this combination and it is relates to an inactive record
		// case when it relates to an active record is already dropped by the validator
		// so just update the existing view record
		uniqueViewRecordBuilder, err = intents.UpdateValue(uniqueViewKB, uniqueViewRecord)
	} else {
		uniqueViewRecordBuilder, err = intents.NewValue(uniqueViewKB)
	}
	if err == nil {
		uniqueViewRecordBuilder.PutRecordID(field_ID, rec.ID())
	}
	return err
}

func getUniqueViewRecord2(st istructs.IState, rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields) (istructs.IStateValue, istructs.IStateKeyBuilder, bool, error) {
	uniqueViewRecordBuilder, err := st.KeyBuilder(state.View, qNameViewUniques)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	buildUniqueViewKey2(uniqueViewRecordBuilder, rec, orderedUniqueFields)
	sv, ok, err := st.CanExist(uniqueViewRecordBuilder)
	return sv, uniqueViewRecordBuilder, ok, err
}

// notest err
func buildUniqueViewKey2(kb istructs.IKeyBuilder, rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields) {
	uniqueKeyValues := getUniqueKeyValues2(rec, orderedUniqueFields)
	buildUniqueViewKeyByValues(kb, rec.AsQName(appdef.SystemField_QName), uniqueKeyValues)
}

func getUniqueKeyValues2(rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields) (res []byte) {
	buf := bytes.NewBuffer(nil)
	for _, uniqueField := range orderedUniqueFields {
		val := coreutils.ReadByKind(uniqueField.Name(), uniqueField.DataKind(), rec)
		switch uniqueField.DataKind() {
		case appdef.DataKind_string:
			buf.WriteByte(zeroByte)
			buf.WriteString(val.(string))
		case appdef.DataKind_bytes:
			buf.WriteByte(zeroByte)
			buf.Write(val.([]byte))
		default:
			binary.Write(buf, binary.BigEndian, val) // nolint
		}
	}
	return buf.Bytes()
}

func eventUniqueValidator2(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	//                      ???      unique key bytes
	uniquesState := map[appdef.QName]map[string]*uniqueViewRecord{}
	return iterate.ForEachError(rawEvent.CUDs, func(cudRec istructs.ICUDRow) (err error) {
		cudQName := cudRec.QName()
		cudUniques, ok := appStructs.AppDef().Type(cudQName).(appdef.IUniques)
		if !ok {
			return nil
		}
		err = iterate.ForEachError(cudUniques.Uniques, func(unique appdef.IUnique) error {
			var uniqueKeyValues []byte
			if cudRec.IsNew() {
				// currentRow is cudRec
				orderedUniqueFields := getOrderedUniqueFields(appStructs.AppDef(), cudRec, unique)
				uniqueKeyValues = getUniqueKeyValues2(cudRec, orderedUniqueFields)
			}
			// currentUniqueRecord - is for unique combination from current cudRec
			currentUniqueRecord, err := getCurrentUniqueViewRecord(uniquesState, cudQName, uniqueKeyValues, appStructs, wsid)
			if err != nil {
				return err
			}
			if cudRec.IsNew() {
				// !IsActive is impossible for new records anymore
				if currentUniqueRecord.refRecordID == istructs.NullRecordID {
					// inserting a new active record, unique is inactive -> allowed, update its ID in map
					currentUniqueRecord.refRecordID = cudRec.ID()
					// currentUniqueRecord.exists = true // avoid: 1st CUD insert a unique record, 2nd modify the unique value
				} else {
					// inserting a new active record, unique is active -> deny
					return conflict(cudQName, currentUniqueRecord.refRecordID)
				}
			} else {
				// update
				// unique view record exists because all unique fields are required.
				// cudRecIsActive := cudRec.AsBool(appdef.SystemField_IsActive)
				// let's check if we update unique key fields. Deny this.
				err := iterate.ForEachError2Values(cudRec.ModifiedFields, func(cudModifiedFieldName string, newValue interface{}) error {
					for _, uniqueField := range unique.Fields() {
						if uniqueField.Name() == cudModifiedFieldName {
							// update -> view record exists for sure because all unique fields are required. So just deny to modify unique fields here
							return fmt.Errorf("%v: unique field «%s» can not be changed: %w", cudQName, uniqueField.Name(), ErrUniqueFieldUpdateDeny)
						}
					}
					if cudModifiedFieldName != appdef.SystemField_IsActive {
						return nil
					}
					isActivating := newValue.(bool)
					if isActivating {
						if currentUniqueRecord.refRecordID == istructs.NullRecordID {
							// unique combination exists for any deactivated record (no matter for this cudRec or any other rec),
							// we're activating now -> set current unique combination ref to the cudRec
							currentUniqueRecord.refRecordID = cudRec.ID()
						} else if currentUniqueRecord.refRecordID != cudRec.ID() {
							// we're activating, this combination exists for another record already -> deny
							return conflict(cudQName, currentUniqueRecord.refRecordID)
						}
					} else {
						// deactivating
						currentUniqueRecord.refRecordID = istructs.NullRecordID
					}
					return nil
				})
				if err != nil {
					return err
				}
				// if currentUniqueRecord.refRecordID == cudRec.ID() || currentUniqueRecord.refRecordID == istructs.NullRecordID {
				// 	return fmt.Errorf("%v: unique field «%s» can not be changed: %w", cudQName, uniqueField.Name(), ErrUniqueFieldUpdateDeny)
				// }

			}
			return nil
		})
		return err
	})
}
