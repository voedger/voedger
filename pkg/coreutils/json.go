/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"

	"github.com/voedger/voedger/pkg/istructs"
)

func JSONUnmarshal(b []byte, ptrToPayload interface{}) error {
	reader := bytes.NewReader(b)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode(ptrToPayload)
}

func JSONUnmarshalDisallowUnknownFields(b []byte, ptrToPayload interface{}) error {
	reader := bytes.NewReader(b)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	return decoder.Decode(ptrToPayload)
}

func ClarifyJSONNumber[T int8 | int16 | int32 | int64 | float32 | float64 | istructs.RecordID](value json.Number) (val T, err error) {
	kind := reflect.TypeFor[T]().Name()
	minInt, maxInt := int64(math.MinInt64), int64(math.MaxInt64)
	switch any(val).(type) {
	case int8:
		minInt, maxInt = math.MinInt8, math.MaxInt8
	case int16:
		minInt, maxInt = math.MinInt16, math.MaxInt16
	case int32:
		minInt, maxInt = math.MinInt32, math.MaxInt32
	case istructs.RecordID:
		minInt = 0
	case float32, float64:
		_, isFloat32 := any(val).(float32)
		floatVal, err := value.Float64()
		if err != nil {
			return 0, errFailedToCast(value, kind, err)
		}
		if isFloat32 && (floatVal < -math.MaxFloat32 || floatVal > math.MaxFloat32) {
			return 0, errNumberOverflow(value, kind)
		}
		return T(floatVal), nil
	}
	intVal, err := value.Int64()
	if err != nil {
		return 0, errFailedToCast(value, kind, err)
	}
	if intVal < minInt || intVal > maxInt {
		return 0, errNumberOverflow(value, kind)
	}
	return T(intVal), nil
}

func ClarifyJSONWSID(wsidNumber json.Number) (wsid istructs.WSID, err error) {
	int64Val, err := wsidNumber.Int64()
	if err != nil {
		return 0, errFailedToCast(wsidNumber, "WSID", err)
	}
	if int64Val < 0 || int64Val > istructs.MaxAllowedWSID {
		return 0, errNumberOverflow(wsidNumber, "WSID")
	}
	return istructs.WSID(int64Val), nil
}
