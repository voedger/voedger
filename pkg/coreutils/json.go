/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"bytes"
	"encoding/json"
)

func JSONUnmarshal(b []byte, ptrToPayload interface{}) error {
	reader := bytes.NewReader(b)
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode(ptrToPayload)
}
