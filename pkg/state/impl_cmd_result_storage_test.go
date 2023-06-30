/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
)

func TestCmdResultStorage_Insert(t *testing.T) {
	require := require.New(t)
	fieldName := "name"
	value := "Voedger" //???
	rw := &mockRowWriter{}
	rw.
		On("PutString", fieldName, value)
	cud := &mockCUD{}
	cud.On("Create").Return(rw)
	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID), nil, func() istructs.ICUD { return cud }, nil, nil, 1, nil)
	kb, err := s.KeyBuilder(CmdResultStorage, testRecordQName1)
	require.NoError(err)

	vb, err := s.NewValue(kb)
	require.NoError(err)

	vb.PutString(fieldName, value)
}
