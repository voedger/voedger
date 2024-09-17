/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
)

func TestMockInterfacesCompliance(t *testing.T) {
	var (
		_ istructs.ICUDRow            = &MockCUDRow{}
		_ istructs.IPLogEvent         = &MockPLogEvent{}
		_ istructs.IObject            = &MockObject{}
		_ istructs.IState             = &MockState{}
		_ istructs.IStateKeyBuilder   = &MockStateKeyBuilder{}
		_ istructs.IStateValue        = &MockStateValue{}
		_ istructs.IStateValueBuilder = &MockStateValueBuilder{}
		_ istructs.IRawEvent          = &MockRawEvent{}
		_ istructs.IKey               = &MockKey{}
		_ istructs.IIntents           = &MockIntents{}
	)
}
