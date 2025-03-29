/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	testPartitionID := istructs.PartitionID(1)
	
	New(testPartitionID, )
}
