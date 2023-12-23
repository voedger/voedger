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
	"net/http"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
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
			for _, unique := range iUniques.Uniques() {
				orderedUniqueFields := getOrderedUniqueFields(appDef, rec, unique)
				if rec.IsNew() {
					err = insert2(st, rec, intents, orderedUniqueFields, unique)
				} else {
					err = update2(st, rec, intents, orderedUniqueFields, unique)
				}
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
}

func getOrderedUniqueFields(appDef appdef.IAppDef, rec istructs.IRowReader, unique appdef.IUnique) (orderedUniqueFields orderedUniqueFields) {
	recType := appDef.Type(rec.AsQName(appdef.SystemField_QName))
	recSchemaFields := recType.(appdef.IFields)
	for _, schemaField := range recSchemaFields.Fields() {
		for _, uniqueFieldDesc := range unique.Fields() {
			if uniqueFieldDesc.Name() == schemaField.Name() {
				orderedUniqueFields = append(orderedUniqueFields, schemaField)
			}
		}
	}
	return orderedUniqueFields
}

type uniqueViewRecord struct {
	// refRecordID is not enought because NullRecordID could mean record exists but deactivated
	// need to avoid unique key fields modification in this case
	// no needed because uniqueViewRecord type is used on update only. But all unique fields are required -> view record exists on any update
	// exists      bool
	refRecordID istructs.RecordID
}

func update2(st istructs.IState, rec istructs.ICUDRow, intents istructs.IIntents, orderedUniqueFields orderedUniqueFields, unique appdef.IUnique) error {
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
	uniqueViewRecord, uniqueViewKB, _, err := getUniqueViewRecord2(st, currentRecord, orderedUniqueFields, unique)
	if err != nil {
		return err
	}
	refIDToSet := istructs.NullRecordID
	uvrID := uniqueViewRecord.AsRecordID(field_ID)
	if rec.AsBool(appdef.SystemField_IsActive) {
		if uvrID == istructs.NullRecordID {
			// activating the record whereas previous combination was deactivated -> allow, update the view
			refIDToSet = rec.ID()
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
	uniqueViewUpdater, err := intents.UpdateValue(uniqueViewKB, uniqueViewRecord)
	if err != nil {
		return err
	}
	uniqueViewUpdater.PutRecordID(field_ID, refIDToSet)
	return nil
}

func insert2(state istructs.IState, rec istructs.ICUDRow, intents istructs.IIntents, orderedUniqueFields orderedUniqueFields, unique appdef.IUnique) error {
	uniqueViewRecord, uniqueViewKB, uniqueViewRecordExists, err := getUniqueViewRecord2(state, rec, orderedUniqueFields, unique)
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

func getUniqueViewRecord2(st istructs.IState, rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields, unique appdef.IUnique) (istructs.IStateValue, istructs.IStateKeyBuilder, bool, error) {
	uniqueViewRecordBuilder, err := st.KeyBuilder(state.View, qNameViewUniques)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	buildUniqueViewKey2(uniqueViewRecordBuilder, rec, orderedUniqueFields, unique)
	sv, ok, err := st.CanExist(uniqueViewRecordBuilder)
	return sv, uniqueViewRecordBuilder, ok, err
}

func buildUniqueViewKeyByValues(kb istructs.IKeyBuilder, docQName appdef.QName, uniqueKeyValues []byte, uniqueID appdef.UniqueID) {
	kb.PutQName(field_QName, docQName)
	kb.PutInt64(field_ValuesHash, coreutils.HashBytes(uniqueKeyValues))
	kb.PutBytes(field_Values, uniqueKeyValues)
}

// notest err
func buildUniqueViewKey2(kb istructs.IKeyBuilder, rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields, unique appdef.IUnique) error {
	uniqueKeyValues, err := getUniqueKeyValues2(rec, orderedUniqueFields, unique.Name(), unique.ID())
	if err != nil {
		return err
	}
	buildUniqueViewKeyByValues(kb, rec.AsQName(appdef.SystemField_QName), uniqueKeyValues, unique.ID())
	return nil
}

func getUniqueKeyValues2(rec istructs.IRowReader, orderedUniqueFields orderedUniqueFields, uniqueName string, uniqueID appdef.UniqueID) (res []byte, err error) {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, uniqueID)
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
	if buf.Len() > int(appdef.MaxFieldLength) {
		return nil, fmt.Errorf(`%w: resulting len of the unique combination "%s.%s" is %d, max %d is allowed. Decrease len of values of unique fields`,
			errUniqueValueTooLong, rec.AsQName(appdef.SystemField_QName), uniqueName, buf.Len(), appdef.MaxFieldLength)
	}
	return buf.Bytes(), nil
}

func getCurrentUniqueViewRecord2(uniquesState map[appdef.QName]map[appdef.UniqueID]map[string]*uniqueViewRecord,
	cudQName appdef.QName, uniqueKeyValues []byte, appStructs istructs.IAppStructs, wsid istructs.WSID, uniqueID appdef.UniqueID) (*uniqueViewRecord, error) {
	// why to accumulate in a map?
	//         id:  field: IsActive: Result:
	// stored: 111: xxx    -
	// …
	// cud(I): 222: xxx    +         - should be ok to insert new record
	// …
	// cud(J): 111:        +         - should be denied to restore old record
	cudQNameUniques, ok := uniquesState[cudQName]
	if !ok {
		cudQNameUniques = map[appdef.UniqueID]map[string]*uniqueViewRecord{}
		uniquesState[cudQName] = cudQNameUniques
	}
	uniqueViewRecords, ok := cudQNameUniques[uniqueID]
	if !ok {
		uniqueViewRecords = map[string]*uniqueViewRecord{}
		cudQNameUniques[uniqueID] = uniqueViewRecords
	}
	currentUniqueViewRecord, ok := uniqueViewRecords[string(uniqueKeyValues)]
	if !ok {
		currentUniqueRecordID, _, err := getUniqueIDByValues2(appStructs, wsid, cudQName, uniqueKeyValues, uniqueID)
		if err != nil {
			return nil, err
		}
		currentUniqueViewRecord = &uniqueViewRecord{
			refRecordID: currentUniqueRecordID,
		}
		uniqueViewRecords[string(uniqueKeyValues)] = currentUniqueViewRecord
	}
	return currentUniqueViewRecord, nil
}

func getUniqueIDByValues2(appStructs istructs.IAppStructs, wsid istructs.WSID, docQName appdef.QName, uniqueKeyValues []byte, uniqueID appdef.UniqueID) (istructs.RecordID, bool, error) {
	kb := appStructs.ViewRecords().KeyBuilder(qNameViewUniques)
	buildUniqueViewKeyByValues(kb, docQName, uniqueKeyValues, uniqueID)
	val, err := appStructs.ViewRecords().Get(wsid, kb)
	if err == nil {
		return val.AsRecordID(field_ID), true, nil
	}
	if err == istructsmem.ErrRecordNotFound {
		err = nil
	}
	return istructs.NullRecordID, false, err
}

func eventUniqueValidator2(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	//                    cudQName                      unique-key-bytes
	uniquesState := map[appdef.QName]map[appdef.UniqueID]map[string]*uniqueViewRecord{}
	return iterate.ForEachError(rawEvent.CUDs, func(cudRec istructs.ICUDRow) (err error) {
		cudQName := cudRec.QName()
		cudUniques, ok := appStructs.AppDef().Type(cudQName).(appdef.IUniques)
		if !ok {
			return nil
		}
		for _, unique := range cudUniques.Uniques() {
			var uniqueKeyValues []byte
			var rowSource istructs.IRowReader
			if cudRec.IsNew() {
				// insert -> will get existing values from the current CUD
				rowSource = cudRec
			} else {
				// update -> will get existing values from the stored record
				rowSource, err = appStructs.Records().Get(wsid, true, cudRec.ID())
				if err != nil {
					// notest
					return err
				}
			}
			orderedUniqueFields := getOrderedUniqueFields(appStructs.AppDef(), rowSource, unique)
			uniqueKeyValues, err = getUniqueKeyValues2(rowSource, orderedUniqueFields, unique.Name(), unique.ID())
			if err != nil {
				return err
			}
			// currentUniqueRecord - is for unique combination from current cudRec
			currentUniqueRecord, err := getCurrentUniqueViewRecord2(uniquesState, cudQName, uniqueKeyValues, appStructs, wsid, unique.ID())
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
					return conflict(cudQName, currentUniqueRecord.refRecordID, unique.Name())
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
							return conflict(cudQName, currentUniqueRecord.refRecordID, unique.Name())
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
		}
		return nil
	})
}

func conflict(qName appdef.QName, conflictingWithID istructs.RecordID, uniqueName string) error {
	return coreutils.NewHTTPError(http.StatusConflict, fmt.Errorf(`%s: "%s" %w with ID %d`, qName, uniqueName, ErrUniqueConstraintViolation, conflictingWithID))
}
