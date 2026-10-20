/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package coreutils

import "github.com/voedger/voedger/pkg/istructs"

func (m *MockPLogEvent) HandlingPartition() istructs.PartitionID {
	return m.Called().Get(0).(istructs.PartitionID)
}
