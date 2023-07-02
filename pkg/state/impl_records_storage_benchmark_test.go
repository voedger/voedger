package state

import (
	"context"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

/*
Before:
BenchmarkRecordsGet-20    	   21326	     51747 ns/op	   30498 B/op	     329 allocs/op
*/

type mockBenchRecs struct {
	istructs.IRecords
}

type mockBenchRec struct {
	istructs.IRecord
}

func (r *mockBenchRec) QName() appdef.QName {
	return testRecordQName1
}

var mockRec istructs.IRecord

//	func (r *mockBenchRecs) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
//		return r.Called(workspace, highConsistency, ids).Error(0)
//	}
func (r *mockBenchRecs) Get(workspace istructs.WSID, highConsistency bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	return mockRec, nil
}
func BenchmarkRecordsGet(b *testing.B) {

	mockRec = &mockBenchRec{}

	records := &mockBenchRecs{}
	appDef := appdef.New()
	appDef.AddObject(testRecordQName1).
		AddField("number", appdef.DataKind_int64, false)
	appDef.AddObject(testRecordQName2).
		AddField("age", appdef.DataKind_int64, false)

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(appDef).
		On("Records").Return(records).
		On("ViewRecords").Return(&nilViewRecords{}).
		On("Events").Return(&nilEvents{})
	s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
	k1, err := s.KeyBuilder(RecordsStorage, appdef.NullQName)
	if err != nil {
		panic(err)
	}
	k1.PutRecordID(Field_ID, 2)
	k1.PutInt64(Field_WSID, 1)

	for i := 0; i < b.N; i++ {

		value, ok, err := s.CanExist(k1)
		if err != nil {
			panic(err)
		}
		if value == nil {
			panic("value is nil")
		}
		if !ok {
			panic("!ok")
		}
	}

}
