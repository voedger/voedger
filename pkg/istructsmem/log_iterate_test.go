/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_readLogParts(t *testing.T) {
	const logSize = uint64(16 * 1000)
	type (
		readRangeType  struct{ min, max istructs.Offset }
		readResultType []readRangeType
		resultType     struct {
			readed     readResultType
			totalReads uint64
			err        error
		}
	)
	tests := []struct {
		name        string
		startOffset istructs.Offset
		toReadCount int
		result      resultType
	}{
		{
			name:        "0, 8: read 8 first recs from first partition",
			startOffset: 0,
			toReadCount: 8,
			result: resultType{
				readed:     readResultType{{0, 7}},
				totalReads: 8,
				err:        nil,
			},
		},
		{
			name:        "0, 4096: read all first partition",
			startOffset: 0,
			toReadCount: 4096,
			result: resultType{
				readed:     readResultType{{0, 4095}},
				totalReads: 4096,
				err:        nil,
			},
		},
		{
			name:        "4090, 6: read last 6 recs from first partition",
			startOffset: 4090,
			toReadCount: 6,
			result: resultType{
				readed:     readResultType{{4090, 4095}},
				totalReads: 6,
				err:        nil,
			},
		},
		{
			name:        "4090, 10: read 10 recs from tail of first partition and head of second",
			startOffset: 4090,
			toReadCount: 10,
			result: resultType{
				readed:     readResultType{{4090, 4095}, {1*4096 + 0, 1*4096 + 3}},
				totalReads: 10,
				err:        nil,
			},
		},
		{
			name:        "4000, 10000: read 10’000 recs from tail of first partition, across second and third, and from head of fourth",
			startOffset: 4000,
			toReadCount: 10000,
			result: resultType{
				readed:     readResultType{{4000, 4095}, {1*4096 + 0, 1*4096 + 4095}, {2*4096 + 0, 2*4096 + 4095}, {3*4096 + 0, 3*4096 + 1711}},
				totalReads: 10000,
				err:        nil,
			},
		},
		{
			name:        "15999, ∞: read one last rec from log",
			startOffset: 15999,
			toReadCount: istructs.ReadToTheEnd,
			result: resultType{
				readed:     readResultType{{3*4096 + 3711, 3*4096 + 3711}},
				totalReads: 1,
				err:        io.EOF,
			},
		},
		{
			name:        "15000, ∞: read all recs from 15000 to end of log",
			startOffset: 15000,
			toReadCount: istructs.ReadToTheEnd,
			result: resultType{
				readed:     readResultType{{3*4096 + 2712, 3*4096 + 3711}},
				totalReads: 1000,
				err:        io.EOF,
			},
		},
		{
			name:        "100500, ∞: read all recs beyond the end of log",
			startOffset: 100500,
			toReadCount: istructs.ReadToTheEnd,
			result: resultType{
				readed:     readResultType{},
				totalReads: 0,
				err:        io.EOF,
			},
		},
		{
			name:        "0, ∞: read all recs from log",
			startOffset: 0,
			toReadCount: istructs.ReadToTheEnd,
			result: resultType{
				readed:     readResultType{{0, 4095}, {1*4096 + 0, 1*4096 + 4095}, {2*4096 + 0, 2*4096 + 4095}, {3*4096 + 0, 3*4096 + 3711}},
				totalReads: logSize,
				err:        io.EOF,
			},
		},
		{
			name:        "10, 0: read zero recs from log",
			startOffset: 10,
			toReadCount: 0,
			result: resultType{
				readed:     readResultType{},
				totalReads: 0,
				err:        nil,
			},
		},
		{
			name:        "10, -1: read negative recs from log",
			startOffset: 10,
			toReadCount: -1,
			result: resultType{
				readed:     readResultType{},
				totalReads: 0,
				err:        nil,
			},
		},
	}

	calcOffsets := func(pk []byte, clustFrom, clustTo []byte) (startOffs, finishOffs istructs.Offset) {
		if clustFrom == nil {
			startOffs = calcLogOffset(pk, []byte{0, 0})
		} else {
			startOffs = calcLogOffset(pk, clustFrom)
		}
		if clustTo == nil {
			finishOffs = calcLogOffset(pk, []byte{0, 0})
			finishOffs |= istructs.Offset(lowMask)
		} else {
			finishOffs = calcLogOffset(pk, clustTo) - 1
		}
		return startOffs, finishOffs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			readed := make(readResultType, 0)
			totalReads := uint64(0)

			readPart := func(pk []byte, clustFrom, clustTo []byte) (bool, error) {
				o1, o2 := calcOffsets(pk, clustFrom, clustTo)
				if uint64(o1) >= logSize {
					return false, io.EOF
				}

				r := readRangeType{
					min: o1,
					max: o2,
				}
				if uint64(r.max) >= logSize {
					r.max = istructs.Offset(logSize - 1)
				}
				readed = append(readed, r)

				totalReads = totalReads + uint64(r.max) - uint64(r.min) + 1

				if uint64(r.max) >= logSize {
					return false, io.EOF
				}

				return true, nil
			}

			err := readLogParts(tt.startOffset, tt.toReadCount, readPart)

			require.Equal(tt.result.totalReads, totalReads, "logIterateType.iterate() reads = %v, want %v", totalReads, tt.result.totalReads)
			require.Equal(tt.result.readed, readed, "logIterateType.iterate() readed = %v, want %v", readed, tt.result.readed)
			if err != tt.result.err {
				t.Errorf("logIterateType.iterate() error = %v, want %v", err, tt.result.err)
			}

		})
	}

	t.Run("check readLogParts is breakable by cb() result", func(t *testing.T) {
		require := require.New(t)

		readed := 0

		readPart := func(pk []byte, clustFrom, clustTo []byte) (bool, error) {
			o1, o2 := calcOffsets(pk, clustFrom, clustTo)

			readed += int(o2-o1) + 1

			return readed < 4096*2, nil
		}

		err := readLogParts(0, 100500, readPart)
		require.NoError(err)
		require.Equal(readed, 4096*2)
	})

	t.Run("check readLogParts is breakable by cb() error", func(t *testing.T) {
		require := require.New(t)

		readed := 0
		toRead := 3 * 4096

		testError := fmt.Errorf("test error")

		readPart := func(pk []byte, clustFrom, clustTo []byte) (bool, error) {
			o1, o2 := calcOffsets(pk, clustFrom, clustTo)

			readed += int(o2-o1) + 1

			if readed >= toRead {
				return true, testError
			}

			return true, nil
		}

		err := readLogParts(0, 100500, readPart)
		require.ErrorIs(err, testError)
		require.Equal(toRead, readed)
	})
}
