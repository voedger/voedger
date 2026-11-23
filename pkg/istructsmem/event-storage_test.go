/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

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
