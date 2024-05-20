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
)

func TestQueryContextStorage(t *testing.T) {
	wsid := istructs.NullWSID
	arg := istructs.NewNullObject()

	wsidFunc := SimpleWSIDFunc(wsid)
	argFunc := func() istructs.IObject { return arg }

	s := ProvideQueryProcessorStateFactory()(context.Background(), nil, nil, wsidFunc, nil, nil, nil, nil, argFunc, nil, nil, nil)

	kb, err := s.KeyBuilder(QueryContext, appdef.NullQName)
	require.NoError(t, err)

	v, err := s.MustExist(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(Field_Workspace))
	require.NotNil(t, v.AsValue(Field_ArgumentObject))
}
