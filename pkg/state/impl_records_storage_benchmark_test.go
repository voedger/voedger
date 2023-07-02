package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

/*
Before:
BenchmarkRecordsGet-20    	   21326	     51747 ns/op	   30498 B/op	     329 allocs/op
*/
func BenchmarkRecordsGet(b *testing.B) {

	record := &mockRecord{}
	record.On("QName").Return(testRecordQName1)
	record.On("AsInt64", "number").Return(int64(10))

	require := require.New(b)
	records := &mockRecords{}
	records.
		On("Get", istructs.WSID(1), true, istructs.RecordID(2)).Return(record, nil).
		On("GetBatch", istructs.WSID(1), true, mock.AnythingOfType("[]istructs.RecordGetBatchItem")).
		Return(nil).
		Run(func(args mock.Arguments) {
			items := args.Get(2).([]istructs.RecordGetBatchItem)
			record := &mockRecord{}
			record.On("QName").Return(testRecordQName1)
			record.On("AsInt64", "number").Return(int64(10))
			items[0].Record = record
		})

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
	require.NoError(err)
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
