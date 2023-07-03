package state

import (
	"context"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

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

type mockAppStr struct {
	istructs.IAppStructs
	recs mockBenchRecs
	vr   nilViewRecords
}

func (s *mockAppStr) Records() istructs.IRecords         { return &s.recs }
func (s *mockAppStr) ViewRecords() istructs.IViewRecords { return &s.vr }

//	func (r *mockBenchRecs) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
//		return r.Called(workspace, highConsistency, ids).Error(0)
//	}
func (r *mockBenchRecs) Get(workspace istructs.WSID, highConsistency bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	return mockRec, nil
}
func (r *mockBenchRecs) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
	for i := range ids {
		ids[i].Record = mockRec
	}
	return nil
}

/*
Before:
BenchmarkRecordsGet-20    	  105993	      9932 ns/op	    7424 B/op	      75 allocs/op
After:
BenchmarkRecordsGet-20    	  110175	     11081 ns/op	    7893 B/op	      79 allocs/op
*/
func BenchmarkRecordsGet(b *testing.B) {

	mockRec = &mockBenchRec{}

	// records := &mockBenchRecs{}
	// appDef := appdef.New()
	// appDef.AddObject(testRecordQName1).
	// 	AddField("number", appdef.DataKind_int64, false)
	// appDef.AddObject(testRecordQName2).
	// 	AddField("age", appdef.DataKind_int64, false)

	appStructs := &mockAppStr{}
	// appStructs.
	// 	On("AppDef").Return(appDef).
	// 	On("Records").Return(records).
	// 	On("ViewRecords").Return(&nilViewRecords{}).
	// 	On("Events").Return(&nilEvents{})
	s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
	k1, err := s.KeyBuilder(RecordsStorage, appdef.NullQName)
	if err != nil {
		panic(err)
	}
	k1.PutRecordID(Field_ID, 2)
	k1.PutInt64(Field_WSID, 1)

	for i := 0; i < b.N; i++ {
		s.CanExist(k1)

		// value, ok, err := s.CanExist(k1)
		// if err != nil {
		// 	panic(err)
		// }
		// if value == nil {
		// 	panic("value is nil")
		// }
		// if !ok {
		// 	panic("!ok")
		// }
	}

}
