/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestResponseStorage(t *testing.T) {
	storage := NewResponseStorage()
	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	vb, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
	require.NoError(t, err)
	vb.PutInt32(sys.Storage_Response_Field_StatusCode, 404)
	vb.PutString(sys.Storage_Response_Field_ErrorMessage, "Not found")
	value := vb.BuildValue()
	require.NotNil(t, value)
	require.Equal(t, int32(404), value.AsInt32(sys.Storage_Response_Field_StatusCode))
	require.Equal(t, "Not found", value.AsString(sys.Storage_Response_Field_ErrorMessage))
	require.PanicsWithError(t, "undefined string field: unknown", func() {
		value.AsString("unknown")
	})
}
