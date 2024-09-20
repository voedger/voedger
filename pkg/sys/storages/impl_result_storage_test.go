/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package storages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

func TestCmdResultStorage_InsertInValue(t *testing.T) {
	cmdResBuilder := istructs.NewNullObjectBuilder()
	storage := NewResultStorage(func() istructs.IObjectBuilder { return cmdResBuilder })

	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	vb, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
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
	storage := NewResultStorage(func() istructs.IObjectBuilder { return cmdResBuilder })

	kb := storage.NewKeyBuilder(appdef.NullQName, nil)

	fieldName := "name"
	value := "value"

	kb.PutString(fieldName, value)
}
