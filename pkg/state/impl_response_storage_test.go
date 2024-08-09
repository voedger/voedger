/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestResponseStorage(t *testing.T) {
	var s IHostState

	test := func(t *testing.T) {
		kb, err := s.KeyBuilder(sys.Storage_Response, appdef.NullQName)
		require.NoError(t, err)

		vb, err := s.NewValue(kb)
		require.NoError(t, err)

		vb.PutInt32(sys.Storage_Response_Field_StatusCode, 404)
		vb.PutString(sys.Storage_Response_Field_ErrorMessage, "Not found")

		kb2, err := s.KeyBuilder(sys.Storage_Response, appdef.NullQName)
		require.NoError(t, err)
		intent := s.FindIntent(kb2)
		require.NotNil(t, intent)

		value := intent.BuildValue()
		require.NotNil(t, value)
		require.Equal(t, int32(404), value.AsInt32(sys.Storage_Response_Field_StatusCode))
		require.Equal(t, "Not found", value.AsString(sys.Storage_Response_Field_ErrorMessage))
		require.PanicsWithError(t, "undefined string field: unknown", func() {
			value.AsString("unknown")
		})
	}

	s = ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID),
		nil, nil, nil, nil, 1, nil, nil, nil, nil, nil)
	t.Run("CommandProcessor", test)
	s = ProvideQueryProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID),
		nil, nil, nil, nil, nil, nil, nil, nil, nil)
	t.Run("QueryProcessor", test)

}
