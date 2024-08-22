/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

/*
Before:
BenchmarkRecordsGet-20    	 3476623	       332.4 ns/op	     104 B/op	       5 allocs/op
After:
BenchmarkRecordsGet-20    	23072106	        55.72 ns/op	      16 B/op	       1 allocs/op
*/
func BenchmarkRecordsGet(b *testing.B) {
	mockRec = &mockBenchRec{}
	appStructs := &mockAppStr{}
	appStructsFunc := func() istructs.IAppStructs {
		return appStructs
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
	k1 := storage.NewKeyBuilder(appdef.NullQName, nil)
	k1.PutRecordID(sys.Storage_Record_Field_ID, 2)
	k1.PutInt64(sys.Storage_Record_Field_WSID, 1)
	withGet := storage.(state.IWithGet)

	for i := 0; i < b.N; i++ {
		value, err := withGet.Get(k1)
		if err != nil {
			panic(err)
		}
		if value == nil {
			panic("value is nil")
		}
	}
}

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
