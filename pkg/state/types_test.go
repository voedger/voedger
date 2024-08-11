/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func appStructsFunc(app istructs.IAppStructs) AppStructsFunc {
	return func() istructs.IAppStructs {
		return app
	}
}

func TestKeyBuilder(t *testing.T) {
	require := require.New(t)

	k := newMapKeyBuilder(testStorage, appdef.NullQName)

	require.Equal(testStorage, k.storage)
	require.PanicsWithValue(ErrNotSupported, func() { k.PartitionKey() })
	require.PanicsWithValue(ErrNotSupported, func() { k.ClusteringColumns() })
}
