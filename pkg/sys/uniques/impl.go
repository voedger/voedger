/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideUniquesProjectorFunc(appDef appdef.IAppDef) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		return event.CUDs(func(rec istructs.ICUDRow) (err error) {
			if unique, ok := appDef.Def(rec.QName()).(appdef.IUniques); ok {
				if field := unique.UniqueField(); field != nil {
					if rec.IsNew() {
						err = insert(rec, field, st, intents)
					} else {
						// came here -> we're updating fields that are not part of an unique key
						// e.g. updating sys.IsActive
						err = update(rec, field, st, intents)
					}
				}
			}
			return err
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

func provideCUDUniqueUpdateDenyValidator() func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) error {
	return func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
		if cudRow.IsNew() {
			return nil
		}
		qName := cudRow.QName()
		if uniques, ok := appStructs.AppDef().Def(qName).(appdef.IUniques); ok {
			if field := uniques.UniqueField(); field != nil {
				cudRow.ModifiedFields(func(fieldName string, newValue interface{}) {
					if fieldName == field.Name() {
						err = errors.Join(err,
							fmt.Errorf("%v: unique field «%s» can not to be changed: %w", qName, fieldName, ErrUniqueFieldUpdateDeny))
					}
				})
			}
		}
		return err
	}
}

func provideEventUniqueValidator() func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	return func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
		//                                      key         uvrID
		uniquesState := map[appdef.QName]map[string]istructs.RecordID{}
		err := rawEvent.CUDs(
			func(rec istructs.ICUDRow) (err error) {
				var actualRow istructs.IRowReader
				if rec.IsNew() {
					actualRow = rec
				} else if actualRow, err = appStructs.Records().Get(wsid, true, rec.ID()); err != nil { // read current record
					return err
				}

				qName := rec.QName()
				if uniques, ok := appStructs.AppDef().Def(qName).(appdef.IUniques); ok {
					if field := uniques.UniqueField(); field != nil {
						cudUniqueKeyValues, err := getUniqueKeyValues(actualRow, field)
						if err != nil {
							// notest
							return err
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
							qNameEventUniques = map[string]istructs.RecordID{}
							uniquesState[qName] = qNameEventUniques
						}
						currentRecordID, ok := qNameEventUniques[string(cudUniqueKeyValues)]
						if !ok {
							if currentRecordID, err = getUniqueIDByValues(appStructs, wsid, qName, cudUniqueKeyValues); err != nil {
								return err
							}
							qNameEventUniques[string(cudUniqueKeyValues)] = currentRecordID
						}

						// inserting a new inactive record, unique is inactive or active -> allowed, nothing to do
						if rec.IsNew() {
							if rec.AsBool(appdef.SystemField_IsActive) {
								if currentRecordID == istructs.NullRecordID {
									// inserting a new active record, unique is inactive -> allowed, update its ID in map
									qNameEventUniques[string(cudUniqueKeyValues)] = rec.ID()
								} else {
									// inserting a new active record, unique is active -> deny
									return conflict(qName, currentRecordID)
								}
							}
						} else {
							// came here -> we're updating fields that are not unique key fields
							// let's check sys.IsActive only
							recIsActive := rec.AsBool(appdef.SystemField_IsActive)
							existingRecordIsActive := actualRow.AsBool(appdef.SystemField_IsActive)
							isActivating := false
							if recIsActive && !existingRecordIsActive {
								isActivating = true
							}
							if (recIsActive && existingRecordIsActive) || (!recIsActive && !existingRecordIsActive) {
								// no changes
								return nil
							}

							if isActivating {
								if currentRecordID == istructs.NullRecordID {
									// activating, unique is active -> allowed, update its ID in map
									qNameEventUniques[string(cudUniqueKeyValues)] = rec.ID()
								} else {
									// activating, unique is active -> deny
									return conflict(qName, currentRecordID)
								}
							} else if currentRecordID != istructs.NullRecordID {
								// deactivating, unique is active -> allowed, reset its ID in map
								qNameEventUniques[string(cudUniqueKeyValues)] = istructs.NullRecordID
							}
							// deactivating, unique is deactivated -> allowed, nothing to do
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
