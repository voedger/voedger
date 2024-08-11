/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestCmdResultStorage_InsertInValue(t *testing.T) {
	cmdResBuilder := istructs.NewNullObjectBuilder()
	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID),
		nil, nil, nil, nil, 1, func() istructs.IObjectBuilder { return cmdResBuilder }, nil, nil, nil, nil)

	kb, err := s.KeyBuilder(sys.Storage_Result, testRecordQName1)
	require.NoError(t, err)

	vb, err := s.NewValue(kb)
	require.NoError(t, err)

	fieldName := "name"
	value := "value"

	vb.PutString(fieldName, value)
}

func TestResultStorage_InsertInKey(t *testing.T) {
	defer func() {
		r := fmt.Sprint(recover())
		require.Equal(t, "undefined string field: name", r)
	}()

	cmdResBuilder := istructs.NewNullObjectBuilder()
	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID),
		nil, nil, nil, nil, 1, func() istructs.IObjectBuilder { return cmdResBuilder }, nil, nil, nil, nil)

	kb, err := s.KeyBuilder(sys.Storage_Result, testRecordQName1)
	require.NoError(t, err)

	fieldName := "name"
	value := "value"

	kb.PutString(fieldName, value)
}

func TestResultStorage_QueryProcessor(t *testing.T) {

	sentObjects := make([]istructs.IObject, 0)

	cmdResBuilder := istructs.NewNullObjectBuilder()

	execQueryCallback := func() istructs.ExecQueryCallback {
		return func(object istructs.IObject) error {
			sentObjects = append(sentObjects, object)
			return nil
		}
	}
	s := ProvideQueryProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID),
		nil, nil, nil, nil, nil, nil, func() istructs.IObjectBuilder { return cmdResBuilder }, nil, execQueryCallback)

	kb, err := s.KeyBuilder(sys.Storage_Result, appdef.NullQName)
	require.NoError(t, err)

	intent, err := s.NewValue(kb)
	require.NoError(t, err)
	require.NotNil(t, intent)

	intent, err = s.NewValue(kb)
	require.NoError(t, err)
	require.NotNil(t, intent)

	require.NoError(t, s.ApplyIntents())
	require.Len(t, sentObjects, 2)

}
