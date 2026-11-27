/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package istructsmem

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/dynobuffers"
	bytespool "github.com/valyala/bytebufferpool"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSingleLogReadFailureReleasesBuffers(t *testing.T) {
	require := require.New(t)
	storageErr := errors.New("storage read failed")
	const updatedValue = "updated document"

	storeTruncatedEvent := func(test *testEnvironment, pKey, cCols []byte, fieldValue string) {
		source := test.newTestEvent(test.plogOfs, test.wlogOfs)
		defer source.Free()
		origin := test.newTestCDoc(100500)
		defer origin.release()
		source.CUDBuilder().Update(origin).PutString("string", updatedValue)
		require.NoError(source.build())

		// Cutting inside a field leaves the decoder holding the buffer for that row.
		prefix, _, found := bytes.Cut(source.storeToBytes(), []byte(fieldValue))
		require.True(found, "missing encoded field %q", fieldValue)
		require.NoError(test.Storage.Put(pKey, cCols, prefix))
	}

	for _, log := range []string{"ReadPLog", "ReadWLog"} {
		for _, tc := range []struct {
			name        string
			truncateAt  string
			expectedErr error
			errContext  string
		}{
			{name: "storage error", expectedErr: storageErr},
			{name: "missing event"},
			{name: "truncated nested argument", truncateAt: defTestEnv.goodNames[1], expectedErr: io.ErrUnexpectedEOF, errContext: "child[1]"},
			{name: "truncated created row", truncateAt: defTestEnv.remarkValue, expectedErr: io.ErrUnexpectedEOF, errContext: "CUD new row"},
			{name: "truncated updated row", truncateAt: updatedValue, expectedErr: io.ErrUnexpectedEOF, errContext: "CUD updated row"},
		} {
			t.Run(log+"/"+tc.name, func(t *testing.T) {
				test := newTestWithPLogCacheSize(0)
				events := test.AppStructs.Events()
				onEvent := func(_ istructs.Offset, event istructs.IWLogEvent) error {
					t.Error("single-event read entered the callback")
					event.Release()
					return nil
				}
				pKey, cCols := WLogEventKey(test.workspace, test.wlogOfs)
				read := func() error { return events.ReadWLog(t.Context(), test.workspace, test.wlogOfs, 1, onEvent) }
				if log == "ReadPLog" {
					pKey, cCols = PLogEventKey(test.partition, test.plogOfs)
					read = func() error {
						return events.ReadPLog(t.Context(), test.partition, test.plogOfs, 1, func(ofs istructs.Offset, event istructs.IPLogEvent) error {
							return onEvent(ofs, event)
						})
					}
				}

				if tc.expectedErr == storageErr {
					test.Storage.ScheduleGetError(storageErr, pKey, cCols)
				}
				if tc.truncateAt != "" {
					storeTruncatedEvent(test, pKey, cCols, tc.truncateAt)
				}

				var borrowed bytespool.ByteBuffer
				var gets int
				var returned []*bytespool.ByteBuffer
				test.AppCfg.eventBuffers = eventBufferPool{
					get: func() *bytespool.ByteBuffer {
						gets++
						return &borrowed
					},
					put: func(b *bytespool.ByteBuffer) { returned = append(returned, b) },
				}
				inUse := dynobuffers.GetObjectsInUse()
				err := read()
				require.ErrorIs(err, tc.expectedErr)
				if tc.errContext != "" {
					require.ErrorContains(err, tc.errContext)
				}
				require.Equal(1, gets, "event buffer borrows")
				require.Len(returned, 1, "event buffer returns")
				require.Same(&borrowed, returned[0])
				require.Equal(inUse, dynobuffers.GetObjectsInUse(), "partially decoded row buffers")
			})
		}
	}
}

func TestReadPLogMarksEventsStored(t *testing.T) {
	for _, tc := range []struct {
		name      string
		buildErr  error
		corrupted bool
	}{
		{name: "valid event"},
		{name: "build-error event", buildErr: errors.New("command failed")},
		{name: "corrupted event", buildErr: ErrCorruptedData, corrupted: true},
	} {
		for _, read := range []struct {
			name  string
			count int
		}{
			{name: "single event", count: 1},
			{name: "scan", count: istructs.ReadToTheEnd},
		} {
			t.Run(tc.name+"/"+read.name, func(t *testing.T) {
				require := require.New(t)
				test := newTest()
				source := test.newTestEvent(test.plogOfs, test.wlogOfs)
				defer source.Free()
				if tc.corrupted {
					source.setName(istructs.QNameForCorruptedData)
				}
				source.setBuildError(tc.buildErr)
				require.PanicsWithValue("only events read from the storage can be re-applied", func() {
					test.AppStructs.GetEventReapplier(source)
				})

				// Bypass PutPlog so neither read path can return a cached event.
				pKey, cCols := PLogEventKey(test.partition, test.plogOfs)
				require.NoError(test.Storage.Put(pKey, cCols, source.storeToBytes()))

				readCount := 0
				err := test.AppStructs.Events().ReadPLog(t.Context(), test.partition, test.plogOfs, read.count,
					func(offset istructs.Offset, actual istructs.IPLogEvent) error {
						defer actual.Release()
						readCount++
						require.Equal(test.plogOfs, offset)
						require.Equal(test.wlogOfs, actual.WLogOffset())
						require.Equal(source.QName(), actual.QName())
						require.Equal(tc.buildErr == nil, actual.Error().ValidEvent())
						require.True(actual.(*eventType).isStored)
						require.NotNil(test.AppStructs.GetEventReapplier(actual))
						return nil
					})
				require.NoError(err)
				require.Equal(1, readCount)
			})
		}
	}
}

func TestLoadEventFailureDoesNotMarkStored(t *testing.T) {
	for _, tc := range []struct {
		name     string
		buildErr error
	}{
		{name: "truncated valid event"},
		{name: "truncated build-error event", buildErr: errors.New("command failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			test := newTest()
			source := test.newTestEvent(test.plogOfs, test.wlogOfs)
			defer source.Free()
			source.setBuildError(tc.buildErr)
			data := source.storeToBytes()
			for length := range data {
				actual := test.newEmptyTestEvent()
				t.Cleanup(actual.Free)
				require.Error(actual.loadFromBytes(data[:length]), "truncated at %d", length)
				require.False(actual.isStored, "truncated at %d", length)
			}
		})
	}

	t.Run("unknown codec", func(t *testing.T) {
		require := require.New(t)
		actual := newTest().newEmptyTestEvent()
		defer actual.Free()
		require.Error(actual.loadFromBytes([]byte{255}))
		require.False(actual.isStored)
	})
}
