/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func SimpleWSIDFunc(wsid istructs.WSID) WSIDFunc {
	return func() istructs.WSID { return wsid }
}
func SimplePartitionIDFunc(partitionID istructs.PartitionID) PartitionIDFunc {
	return func() istructs.PartitionID { return partitionID }
}
func WithExcludeFields(fieldNames ...string) ToJSONOption {
	return func(opts *ToJSONOptions) {
		for _, name := range fieldNames {
			opts.excludedFields[name] = true
		}
	}
}
func put(fieldName string, kind appdef.DataKind, rr istructs.IRowReader, rw istructs.IRowWriter) {
	switch kind {
	case appdef.DataKind_int32:
		rw.PutInt32(fieldName, rr.AsInt32(fieldName))
	case appdef.DataKind_int64:
		rw.PutInt64(fieldName, rr.AsInt64(fieldName))
	case appdef.DataKind_float32:
		rw.PutFloat32(fieldName, rr.AsFloat32(fieldName))
	case appdef.DataKind_float64:
		rw.PutFloat64(fieldName, rr.AsFloat64(fieldName))
	case appdef.DataKind_bytes:
		rw.PutBytes(fieldName, rr.AsBytes(fieldName))
	case appdef.DataKind_string:
		rw.PutString(fieldName, rr.AsString(fieldName))
	case appdef.DataKind_QName:
		rw.PutQName(fieldName, rr.AsQName(fieldName))
	case appdef.DataKind_bool:
		rw.PutBool(fieldName, rr.AsBool(fieldName))
	case appdef.DataKind_RecordID:
		rw.PutRecordID(fieldName, rr.AsRecordID(fieldName))
	default:
		panic(fmt.Errorf("illegal state: field - '%s', kind - '%d': %w", fieldName, kind, ErrNotSupported))
	}
}

func getStorageID(key istructs.IKeyBuilder) appdef.QName {
	switch k := key.(type) {
	case *pLogKeyBuilder:
		return PLogStorage
	case *wLogKeyBuilder:
		return WLogStorage
	case *recordsKeyBuilder:
		return RecordsStorage
	case *keyBuilder:
		return k.storage
	case *sendMailStorageKeyBuilder:
		return SendMailStorage
	case *httpStorageKeyBuilder:
		return HTTPStorage
	case *viewRecordsKeyBuilder:
		return ViewRecordsStorage
	case *cmdResultKeyBuilder:
		return CmdResultStorage
	default:
		panic(fmt.Errorf("key %+v: %w", key, ErrUnknownStorage))
	}
}

func cudRowToMap(rec istructs.ICUDRow, cache appDefFunc) (res map[string]interface{}) {
	res = coreutils.FieldsToMap(rec, cache())
	res["IsNew"] = rec.IsNew()
	return res
}
