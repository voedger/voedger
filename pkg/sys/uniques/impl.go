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
				schemaOrderedUniqueFields := unique.FieldsSchemaOrdered()
				if rec.IsNew() {
					err = insert(st, rec, intents, schemaOrderedUniqueFields, unique)
				} else {
					err = update(st, rec, intents, schemaOrderedUniqueFields, unique)
				}
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
}

type uniqueViewRecord struct {
	refRecordID istructs.RecordID
}

func update(st istructs.IState, rec istructs.ICUDRow, intents istructs.IIntents, schemaOrderedUniqueFields schemaOrderedUniqueFields, unique appdef.IUnique) error {
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

	// we're updating -> unique view record exists
	uniqueViewRecord, uniqueViewKB, _, err := getUniqueViewRecord(st, currentRecord, schemaOrderedUniqueFields, unique)
	if err != nil {
		return err
	}
	refIDToSet := istructs.NullRecordID
	uniqueViewRecordID := uniqueViewRecord.AsRecordID(field_ID)
	if rec.AsBool(appdef.SystemField_IsActive) {
		if uniqueViewRecordID == istructs.NullRecordID {
			// activating the record whereas previous combination was deactivated -> allow, update the view
			refIDToSet = rec.ID()
		} else {
			// activating the already activated record, unique combination exists for that record -> allow, nothing to do
			// note: case when uniqueViewRecordID != rec.ID() is handled already by the validator, so nothing to do here
			return nil
		}
	} else {
		if rec.ID() != uniqueViewRecordID {
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

func insert(state istructs.IState, rec istructs.ICUDRow, intents istructs.IIntents, schemaOrderedUniqueFields schemaOrderedUniqueFields, unique appdef.IUnique) error {
	uniqueViewRecord, uniqueViewKB, uniqueViewRecordExists, err := getUniqueViewRecord(state, rec, schemaOrderedUniqueFields, unique)
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

func getUniqueViewRecord(st istructs.IState, rec istructs.IRowReader, schemaOrderedUniqueFields schemaOrderedUniqueFields, unique appdef.IUnique) (istructs.IStateValue, istructs.IStateKeyBuilder, bool, error) {
	uniqueViewRecordBuilder, err := st.KeyBuilder(state.View, qNameViewUniques)
	if err != nil {
		// notest
		return nil, nil, false, err
	}
	uniqueKeyValues, err := getUniqueKeyValues(rec, schemaOrderedUniqueFields, unique.Name(), unique.ID())
	if err != nil {
		return nil, nil, false, err
	}
	buildUniqueViewKeyByValues(uniqueViewRecordBuilder, rec.AsQName(appdef.SystemField_QName), uniqueKeyValues, unique.ID())
	sv, ok, err := st.CanExist(uniqueViewRecordBuilder)
	return sv, uniqueViewRecordBuilder, ok, err
}

func buildUniqueViewKeyByValues(kb istructs.IKeyBuilder, docQName appdef.QName, uniqueKeyValues []byte, uniqueID appdef.UniqueID) {
	kb.PutQName(field_QName, docQName)
	kb.PutInt64(field_ValuesHash, coreutils.HashBytes(uniqueKeyValues))
	kb.PutBytes(field_Values, uniqueKeyValues)
}

func getUniqueKeyValues(rec istructs.IRowReader, schemaOrderedUniqueFields schemaOrderedUniqueFields, uniqueName string, uniqueID appdef.UniqueID) (res []byte, err error) {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, uniqueID)
	for _, uniqueField := range schemaOrderedUniqueFields {
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
			ErrUniqueValueTooLong, rec.AsQName(appdef.SystemField_QName), uniqueName, buf.Len(), appdef.MaxFieldLength)
	}
	return buf.Bytes(), nil
}

func getCurrentUniqueViewRecord(uniquesState map[appdef.QName]map[appdef.UniqueID]map[string]*uniqueViewRecord,
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
		currentUniqueRecordID, _, err := getUniqueIDByValues(appStructs, wsid, cudQName, uniqueKeyValues, uniqueID)
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

func getUniqueIDByValues(appStructs istructs.IAppStructs, wsid istructs.WSID, docQName appdef.QName, uniqueKeyValues []byte, uniqueID appdef.UniqueID) (istructs.RecordID, bool, error) {
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

func eventUniqueValidator(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
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
			schemaOrderedUniqueFields := unique.FieldsSchemaOrdered()
			uniqueKeyValues, err = getUniqueKeyValues(rowSource, schemaOrderedUniqueFields, unique.Name(), unique.ID())
			if err != nil {
				return err
			}
			// uniqueViewRecord - is for unique combination from current cudRec
			uniqueViewRecord, err := getCurrentUniqueViewRecord(uniquesState, cudQName, uniqueKeyValues, appStructs, wsid, unique.ID())
			if err != nil {
				return err
			}
			if cudRec.IsNew() {
				// !IsActive is impossible for new records anymore
				if uniqueViewRecord.refRecordID == istructs.NullRecordID {
					// inserting a new active record, the doc record according to this combination is inactive or does not exist -> allow, update its ID in map
					uniqueViewRecord.refRecordID = cudRec.ID()
				} else {
					// inserting a new active record, the doc record according to this combination is active -> deny
					return conflict(cudQName, uniqueViewRecord.refRecordID, unique.Name())
				}
			} else {
				// update
				// unique view record exists because all unique fields are required.
				// let's deny to update unique fields and handle IsActive state
				err := iterate.ForEachError2Values(cudRec.ModifiedFields, func(cudModifiedFieldName string, newValue interface{}) error {
					for _, uniqueField := range unique.Fields() {
						if uniqueField.Name() == cudModifiedFieldName {
							return fmt.Errorf("%v: unique field «%s» can not be changed: %w", cudQName, uniqueField.Name(), ErrUniqueFieldUpdateDeny)
						}
					}
					if cudModifiedFieldName != appdef.SystemField_IsActive {
						return nil
					}
					// we're updating IsActive field here.
					isActivating := newValue.(bool)
					if isActivating {
						if uniqueViewRecord.refRecordID == istructs.NullRecordID {
							// doc rec for this combination does not exist or is inactive (no matter for this cudRec or any other rec),
							// we're activating now -> set current unique combination ref to the cudRec
							uniqueViewRecord.refRecordID = cudRec.ID()
						} else if uniqueViewRecord.refRecordID != cudRec.ID() {
							// we're activating, doc rec for this combination exists, it is active and it is the another rec (not the one we're updating by the current CUD) -> deny
							return conflict(cudQName, uniqueViewRecord.refRecordID, unique.Name())
						}
					} else {
						// deactivating
						uniqueViewRecord.refRecordID = istructs.NullRecordID
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func conflict(qName appdef.QName, conflictingWithID istructs.RecordID, uniqueName string) error {
	return coreutils.NewHTTPError(http.StatusConflict, fmt.Errorf(`%s: "%s" %w with ID %d`, qName, uniqueName, ErrUniqueConstraintViolation, conflictingWithID))
}
