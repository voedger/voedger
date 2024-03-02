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

func TestCommandContextStorage(t *testing.T) {
	cmdResBuilder := istructs.NewNullObjectBuilder()

	wsid := istructs.NullWSID
	arg := istructs.NewNullObject()
	unloggedArg := istructs.NewNullObject()

	wsidFunc := SimpleWSIDFunc(wsid)
	argFunc := func() istructs.IObject { return arg }
	unloggedArgFunc := func() istructs.IObject { return unloggedArg }

	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, wsidFunc, nil, nil, nil, nil, 1, func() istructs.IObjectBuilder { return cmdResBuilder }, argFunc, unloggedArgFunc)

	kb, err := s.KeyBuilder(CommandContext, appdef.NullQName)
	require.NoError(t, err)

	v, err := s.MustExist(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(Field_Workspace))
	require.NotNil(t, v.AsValue(Field_ArgumentObject))
	require.NotNil(t, v.AsValue(Field_ArgumentUnloggedObject))

}
