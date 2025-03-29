/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage_SeqStorage(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqStorage := NewSeqStorage(sysVvmAppStorage)

	data := []byte{}
	cCols := []byte{}
	ok, err := seqStorage.Get(cCols, &data)
	require.NoError(err)
	require.False(ok)

	cCols = []byte{4, 5, 6}
	err = seqStorage.Put(cCols, []byte{7, 8, 9})
	require.NoError(err)

	cCols = []byte{}
	ok, err = seqStorage.Get(cCols, &data)
	require.NoError(err)
	require.False(ok)

	cCols = []byte{1, 2, 3}
	ok, err = seqStorage.Get(cCols, &data)
	require.NoError(err)
	require.False(ok)

	cCols = []byte{4, 5, 6}
	ok, err = seqStorage.Get(cCols, &data)
	require.NoError(err)
	require.True(ok)
	require.Equal([]byte{7, 8, 9}, data)

	ok, err = sysVvmAppStorage.CompareAndDelete(seqStorage.(*implISeqSysVVMStorage).getPKey(), []byte{4, 5, 6}, []byte{7, 8, 9})
	require.NoError(err)
	require.True(ok)
	ok, err = seqStorage.Get(cCols, &data)
	require.NoError(err)
	require.False(ok)
}
