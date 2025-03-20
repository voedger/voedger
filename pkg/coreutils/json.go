/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func JSONUnmarshal(b []byte, ptrToPayload interface{}) error {
	reader := bytes.NewReader(b)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode(ptrToPayload)
}

func ClarifyJSONNumber(value json.Number, kind appdef.DataKind) (val interface{}, err error) {
	switch kind {
	case appdef.DataKind_int8: // #3434 [small integers]
		int64Val, err := value.Int64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		if int64Val < math.MinInt8 || int64Val > math.MaxInt8 {
			return nil, errNumberOverflow(value, kind.TrimString())
		}
		return int8(int64Val), nil
	case appdef.DataKind_int16: // #3434 [small integers]
		int64Val, err := value.Int64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		if int64Val < math.MinInt16 || int64Val > math.MaxInt16 {
			return nil, errNumberOverflow(value, kind.TrimString())
		}
		return int16(int64Val), nil
	case appdef.DataKind_int32:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		if int64Val < math.MinInt32 || int64Val > math.MaxInt32 {
			return nil, errNumberOverflow(value, kind.TrimString())
		}
		return int32(int64Val), nil
	case appdef.DataKind_int64:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		return int64Val, nil
	case appdef.DataKind_float32:
		float64Val, err := value.Float64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		if float64Val < -math.MaxFloat32 || float64Val > math.MaxFloat32 {
			return nil, errNumberOverflow(value, kind.TrimString())
		}
		return float32(float64Val), nil
	case appdef.DataKind_float64:
		float64Val, err := value.Float64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		return float64Val, nil
	case appdef.DataKind_RecordID:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, errFailedToCast(value, kind.TrimString(), err)
		}
		if int64Val < 0 {
			return nil, errNumberOverflow(value, kind.TrimString())
		}
		return istructs.RecordID(int64Val), nil
	}
	panic(fmt.Sprintf("unsupported data kind %s for json.Number", kind.TrimString()))
}
