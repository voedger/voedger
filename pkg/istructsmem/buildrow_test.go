/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_BuildRow(t *testing.T) {

	require := require.New(t)

	test := test()

	t.Run("Should be ok to BuildRow implemented by local rowType", func(t *testing.T) {
		w := newTestRow()

		r, err := istructs.BuildRow(w)
		require.NoError(err)

		testTestRow(t, r)
	})

	t.Run("Should be ok to BuildRow implemented by local rowType descendants", func(t *testing.T) {

		t.Run("recordType", func(t *testing.T) {
			w := newTestCRecord(100500)

			r, err := istructs.BuildRow(w)
			require.NoError(err)

			rec, ok := r.(istructs.IRecord)
			require.True(ok)

			testTestCRec(t, rec, 100500)
		})

		t.Run("objectType", func(t *testing.T) {
			w := newObject(test.AppCfg, test.saleCmdDocName, nil)
			fillTestObject(w)

			r, err := istructs.BuildRow(w)
			require.NoError(err)

			o, ok := r.(istructs.IObject)
			require.True(ok)

			testTestObject(t, o)
		})

		t.Run("keyType", func(t *testing.T) {
			kb := test.AppStructs.ViewRecords().KeyBuilder(test.testViewRecord.name)
			kb.PutInt32(test.testViewRecord.partFields.partition, 1)
			kb.PutInt64(test.testViewRecord.partFields.workspace, 2)
			kb.PutInt32(test.testViewRecord.ccolsFields.device, 3)
			kb.PutString(test.testViewRecord.ccolsFields.sorter, "a")

			r, err := istructs.BuildRow(kb)
			require.NoError(err)

			key, ok := r.(istructs.IKey)
			require.True(ok)

			require.EqualValues(1, key.AsInt32(test.testViewRecord.partFields.partition))
			require.EqualValues(2, key.AsInt64(test.testViewRecord.partFields.workspace))
			require.EqualValues(3, key.AsInt32(test.testViewRecord.ccolsFields.device))
			require.EqualValues("a", key.AsString(test.testViewRecord.ccolsFields.sorter))
		})

		t.Run("valueType", func(t *testing.T) {
			w := newTestViewValue()

			r, err := istructs.BuildRow(w)
			require.NoError(err)

			v, ok := r.(istructs.IValue)
			require.True(ok)

			testTestViewValue(t, v)
		})
	})

	t.Run("Should be error to BuildRow with errors", func(t *testing.T) {
		w := newEmptyTestRow()
		w.PutBool("unknownField", true)

		r, err := istructs.BuildRow(w)
		require.ErrorWith(err,
			require.Is(ErrNameNotFoundError),
			require.Has("unknownField"),
		)
		require.Nil(r)
	})

	t.Run("Should be error to BuildRow implemented by unknown", func(t *testing.T) {
		type unknown struct {
			istructs.IRowWriter
		}

		w := &unknown{}

		r, err := istructs.BuildRow(w)
		require.ErrorWith(err,
			require.Is(errors.ErrUnsupported),
			require.Has("istructsmem.unknown"),
		)
		require.Nil(r)
	})
}
