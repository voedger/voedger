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
	case appdef.DataKind_int32:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to cast %s to int*: %w", value.String(), err)
		}
		if int64Val < math.MinInt32 || int64Val > math.MaxInt32 {
			return nil, fmt.Errorf("cast %s to int32: %w", value.String(), ErrNumberOverflow)
		}
		return int32(int64Val), nil
	case appdef.DataKind_int64:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to cast %s to int*: %w", value.String(), err)
		}
		return int64Val, nil
	case appdef.DataKind_float32:
		float64Val, err := value.Float64()
		if err != nil {
			return nil, fmt.Errorf("failed to cast %s to float*: %w", value.String(), err)
		}
		if float64Val < -math.MaxFloat32 || float64Val > math.MaxFloat32 {
			return nil, fmt.Errorf("cast %s to float32: %w", value.String(), ErrNumberOverflow)
		}
		return float32(float64Val), nil
	case appdef.DataKind_float64:
		float64Val, err := value.Float64()
		if err != nil {
			return nil, fmt.Errorf("failed to cast %s to float*: %w", value.String(), err)
		}
		return float64Val, nil
	case appdef.DataKind_RecordID:
		int64Val, err := value.Int64()
		if err != nil {
			return nil, fmt.Errorf("failed to cast %s to RecordID: %w", value.String(), err)
		}
		if int64Val < 0 {
			return nil, fmt.Errorf("wrong record ID: %d", int64Val)
		}
		return istructs.RecordID(int64Val), nil
	}
	panic(fmt.Sprintf("unsupported data kind %s for json.Number", kind.TrimString()))
}
