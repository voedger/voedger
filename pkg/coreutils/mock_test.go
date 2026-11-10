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
		_ istructs.ICUDRow             = &MockCUDRow{}
		_ istructs.IPLogEvent          = &MockPLogEvent{}
		_ istructs.IObject             = &MockObject{}
		_ istructs.IState              = &MockState{}
		_ istructs.IStateKeyBuilder    = &MockStateKeyBuilder{}
		_ istructs.IStateValue         = &MockStateValue{}
		_ istructs.IStateValueBuilder  = &MockStateValueBuilder{}
		_ istructs.IRawEvent           = &MockRawEvent{}
		_ istructs.IKey                = &MockKey{}
		_ istructs.IIntents            = &MockIntents{}
		_ istructs.IEvents             = &MockEvents{}
		_ istructs.IAppStructsProvider = &MockAppStructsProvider{}
		_ istructs.IAppStructs         = &MockAppStructs{}
		_ istructs.IAppStructs        = &MockAppStructs{}
		_ istructs.IRecords           = &MockRecords{}
		_ istructs.IViewRecords       = &MockViewRecords{}
		_ istructs.IRecord            = &MockRecord{}
		_ istructs.ICUD               = &MockCUD{}
		_ istructs.IStateValue        = &MockStateRecordValue{}
		_ istructs.IStateRecordValue  = &MockStateRecordValue{}
		_ istructs.IValue             = &MockStateValue{}
		_ istructs.IValueBuilder      = &MockStateValueBuilder{}
		_ istructs.IRowReader         = &MockKey{}
		_ istructs.IRowWriter         = &MockStateValueBuilder{}
		_ istructs.IKeyBuilder        = &MockStateKeyBuilder{}
		_ istructs.IWLogEvent         = &MockPLogEvent{}
	)
}
