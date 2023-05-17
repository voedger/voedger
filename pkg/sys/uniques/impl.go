/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"golang.org/x/exp/slices"
)

func provideUniquesProjectorFunc(uniques istructs.IUniques, appDef appdef.IAppDef) func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
		return event.CUDs(func(rec istructs.ICUDRow) error {
			uniques := uniques.GetAll(rec.QName())
			if len(uniques) == 0 {
				return nil
			}
			for _, unique := range uniques {
				if rec.IsNew() {
					err = insert(rec, st, unique, appDef, intents)
				} else {
					// came here -> we're updating fields that are not part of an unique key
					// e.g. updating sys.IsActive
					err = update(rec, st, unique, appDef, intents)
				}
				if err != nil {
					break
				}
			}
			return err
		})
	}
}

func insert(rec istructs.ICUDRow, state istructs.IState, unique istructs.IUnique, appDef appdef.IAppDef, intents istructs.IIntents) error {
	uniqueViewRecord, uniqueViewKB, ok, err := getUniqueViewRecord(unique, appDef, state, rec)
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

func update(rec istructs.ICUDRow, st istructs.IState, unique istructs.IUnique, appDef appdef.IAppDef, intents istructs.IIntents) error {
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
	uniqueViewRecord, uniqueViewKB, _, err := getUniqueViewRecord(unique, appDef, st, currentRecord)
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

func getUniqueViewRecord(unique istructs.IUnique, appDef appdef.IAppDef, st istructs.IState, rec istructs.IRowReader) (istructs.IStateValue, istructs.IStateKeyBuilder, bool, error) {
	kb, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewUniques)
	if err != nil {
		return nil, nil, false, err
	}
	if err := buildUniqueViewKey(kb, unique, appDef, rec); err != nil {
		// notest
		return nil, nil, false, err
	}
	sv, ok, err := st.CanExist(kb)
	return sv, kb, ok, err
}

func appendValue(fieldName string, buf *bytes.Buffer, rec istructs.IRowReader, kind appdef.DataKind) error {
	val := coreutils.ReadByKind(fieldName, kind, rec)
	switch kind {
	case appdef.DataKind_string:
		if _, err := buf.WriteString(val.(string)); err != nil {
			// notest
			return err
		}
	case appdef.DataKind_bytes:
		if _, err := buf.Write(val.([]byte)); err != nil {
			// notest
			return err
		}
	default:
		return binary.Write(buf, binary.BigEndian, val)
	}
	return nil
}

func getUniqueKeyValues(unique istructs.IUnique, appDef appdef.IAppDef, rec istructs.IRowReader) (res []byte, err error) {
	valuesBytes := bytes.NewBuffer(nil)
	varSizeFieldName := ""
	varSizeFieldKind := appdef.DataKind_null

	if def, ok := appDef.Def(unique.QName()).(appdef.IFields); ok {
		def.Fields(func(field appdef.IField) {
			if err != nil {
				// notest
				return
			}
			if !slices.Contains(unique.Fields(), field.Name()) {
				return
			}
			if field.DataKind().IsFixed() {
				err = appendValue(field.Name(), valuesBytes, rec, field.DataKind())
			} else {
				varSizeFieldName = field.Name()
				varSizeFieldKind = field.DataKind()
			}
		})
	}

	if err == nil && len(varSizeFieldName) > 0 {
		err = appendValue(varSizeFieldName, valuesBytes, rec, varSizeFieldKind)
	}
	return valuesBytes.Bytes(), err
}

// notest err
func buildUniqueViewKeyByValues(uniqueKeyValues []byte, kb istructs.IKeyBuilder, qName appdef.QName) error {
	h := fnv.New64()
	if _, err := h.Write(uniqueKeyValues); err != nil {
		// notest
		return err
	}
	hash := int64(h.Sum64())
	kb.PutQName(field_QName, qName)
	kb.PutInt64(field_ValuesHash, hash)
	kb.PutBytes(field_Values, uniqueKeyValues)
	return nil
}

// notest err
func buildUniqueViewKey(kb istructs.IKeyBuilder, unique istructs.IUnique, appDef appdef.IAppDef, rec istructs.IRowReader) error {
	uniqueKeyValues, err := getUniqueKeyValues(unique, appDef, rec)
	if err != nil {
		// notest
		return err
	}
	return buildUniqueViewKeyByValues(uniqueKeyValues, kb, unique.QName())
}

func provideCUDUniqueUpdateDenyValidator(uniques istructs.IUniques) func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) error {
	return func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
		if cudRow.IsNew() {
			return nil
		}
		uniques := uniques.GetAll(cudRow.QName()) // note: len(unique) > 0 gauranted by the validator matcher
		cudRow.ModifiedFields(func(fieldName string, newValue interface{}) {
			if err != nil {
				return
			}
			for _, unique := range uniques {
				for _, uniqueKeyField := range unique.Fields() {
					if uniqueKeyField == fieldName {
						err = ErrUniqueFieldUpdateDeny
					}
				}
			}
		})
		return err
	}
}

func provideEventUniqueValidator(uniques istructs.IUniques) func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
	return func(ctx context.Context, rawEvent istructs.IRawEvent, appStructs istructs.IAppStructs, wsid istructs.WSID) error {
		//                                      key         uvrID
		uniquesState := map[appdef.QName]map[string]istructs.RecordID{}
		err := rawEvent.CUDs(func(rec istructs.ICUDRow) (err error) {
			uniques := uniques.GetAll(rec.QName())
			if len(uniques) == 0 {
				return nil
			}
			var actualRow istructs.IRowReader
			if rec.IsNew() {
				actualRow = rec
			} else if actualRow, err = appStructs.Records().Get(wsid, true, rec.ID()); err != nil { // read current record
				return err
			}
			for _, unique := range uniques {
				cudUniqueKeyValues, err := getUniqueKeyValues(unique, appStructs.AppDef(), actualRow)
				if err != nil {
					// notest
					return err
				}
				// why to accumulate in a map?
				//              field1 field2 IsActive
				// stored: id1: 12     14     -
				// cud1:   id2: 12     14     +        -ok
				// cud2    id1:               +        -should be denied
				qNameEventUniques, ok := uniquesState[rec.QName()]
				if !ok {
					qNameEventUniques = map[string]istructs.RecordID{}
					uniquesState[rec.QName()] = qNameEventUniques
				}
				currentRecordID, ok := qNameEventUniques[string(cudUniqueKeyValues)]
				if !ok {
					if currentRecordID, err = getUniqueIDByValues(cudUniqueKeyValues, unique, appStructs, wsid); err != nil {
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
							return conflict(rec.QName(), currentRecordID)
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
							return conflict(rec.QName(), currentRecordID)
						}
					} else if currentRecordID != istructs.NullRecordID {
						// deactivating, unique is active -> allowed, reset its ID in map
						qNameEventUniques[string(cudUniqueKeyValues)] = istructs.NullRecordID
					}
					// deactivating, unique is deactivated -> allowed, nothing to do
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
