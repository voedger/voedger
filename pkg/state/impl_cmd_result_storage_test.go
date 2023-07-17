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
	"github.com/voedger/voedger/pkg/istructsmem"
)

func TestCmdResultStorage_InsertInValue(t *testing.T) {
	cfg := istructsmem.AppConfigType{Name: istructs.NullAppQName}
	cmdResBuilder := istructsmem.NewIObjectBuilder(&cfg, appdef.NullQName)
	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID), nil, nil, nil, nil, 1, func() istructs.IObjectBuilder { return cmdResBuilder })

	kb, err := s.KeyBuilder(CmdResultStorage, testRecordQName1)
	require.NoError(t, err)

	vb, err := s.NewValue(kb)
	require.NoError(t, err)

	fieldName := "name"
	value := "value"

	vb.PutString(fieldName, value)
}

func TestCmdResultStorage_InsertInKey(t *testing.T) {
	defer func() {
		r := fmt.Sprint(recover())
		require.Equal(t, "assignment to entry in nil map", r)
	}()

	cfg := istructsmem.AppConfigType{Name: istructs.NullAppQName}
	cmdResBuilder := istructsmem.NewIObjectBuilder(&cfg, appdef.NullQName)
	s := ProvideCommandProcessorStateFactory()(context.Background(), nil, nil, SimpleWSIDFunc(istructs.NullWSID), nil, nil, nil, nil, 1, func() istructs.IObjectBuilder { return cmdResBuilder })

	kb, err := s.KeyBuilder(CmdResultStorage, testRecordQName1)
	require.NoError(t, err)

	fieldName := "name"
	value := "value"

	kb.PutString(fieldName, value)
}
