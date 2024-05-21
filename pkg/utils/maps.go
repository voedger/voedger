/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"errors"
	"fmt"
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
	for fieldName, vIntf := range data {
		switch v := vIntf.(type) {
		case nil:
		case float64:
			rw.PutNumber(fieldName, v)
		case float32:
			rw.PutNumber(fieldName, float64(v))
		case int32:
			rw.PutNumber(fieldName, float64(v))
		case int64:
			rw.PutNumber(fieldName, float64(v))
		case istructs.RecordID:
			rw.PutNumber(fieldName, float64(v))
		case string:
			rw.PutChars(fieldName, v)
		case bool:
			rw.PutBool(fieldName, v)
		default:
			return fmt.Errorf("field %s: unsupported value type %#v", fieldName, vIntf)
		}
	}
	return nil
}

func MergeMaps(target map[string]interface{}, toMergeMaps ...map[string]interface{}) {
	for _, toMergeMap := range toMergeMaps {
		for k, v := range toMergeMap {
			target[k] = v
		}
	}
}
