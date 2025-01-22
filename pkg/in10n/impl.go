/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package in10n

import (
	"bytes"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func (pk ProjectionKey) ToJSON() string {
	buf := bytes.NewBufferString(`{"App":"`)
	buf.WriteString(pk.App.String())
	buf.WriteString(`","Projection":"`)
	buf.WriteString(pk.Projection.String())
	buf.WriteString(`","WS":`)
	buf.WriteString(utils.UintToString(pk.WS))
	buf.WriteString("}")
	return buf.String()
}
