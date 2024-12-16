/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/voedger/voedger/pkg/istructs"
)

// converts slice of "k=v" to map[k]v
func PairsToMap(pairs []string, m map[string]string) error {
	for _, pair := range pairs {
		vals := strings.Split(pair, "=")
		if len(vals) != 2 {
			return errors.New("wrong pair value: " + pair)
		}
		m[vals[0]] = vals[1]
	}
	return nil
}

func MapToObject(data map[string]interface{}, rw istructs.IRowWriter) (err error) {
	for fn, vIntf := range data {
		switch v := vIntf.(type) {
		case nil:
		case float64:
			rw.PutFloat64(fn, v)
		case float32:
			rw.PutFloat32(fn, v)
		case int32:
			rw.PutInt32(fn, v)
		case int64:
			rw.PutInt64(fn, v)
		case istructs.RecordID:
			rw.PutRecordID(fn, v)
		case json.Number:
			rw.PutNumber(fn, v)
		case string:
			rw.PutChars(fn, v)
		case bool:
			rw.PutBool(fn, v)
		default:
			return fmt.Errorf("field %s: unsupported value type %#v", fn, vIntf)
		}
	}
	return nil
}

func MergeMaps(toMergeMaps ...map[string]interface{}) (res map[string]interface{}) {
	res = map[string]interface{}{}
	for _, toMergeMap := range toMergeMaps {
		maps.Copy(res, toMergeMap)
	}
	return res
}
