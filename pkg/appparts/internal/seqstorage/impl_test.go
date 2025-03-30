/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	testWSQName := appdef.NewQName("test", "ws")
	testCDocQName := appdef.NewQName("test", "cdoc")
	mockEvents := &coreutils.MockEvents{}
	appDefBuilder := builder.New()
	ws := appDefBuilder.AddWorkspace(testWSQName)
	ws.AddCDoc(testCDocQName).AddField("IntFld", appdef.DataKind_int32, false)
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	seqSysVVMStorage := storage.NewSeqStorage()
	New(istructs.PartitionID(1), mockEvents, appDef, )
}