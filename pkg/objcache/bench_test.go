/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	IEvent = interface {
		Ofs() uint64
	}
	event struct {
		ofs  uint64
		data [1024]byte
	}
)

func (e *event) Ofs() uint64 {
	return e.ofs
}

func Benchmark_CachePLogEvents(b *testing.B) {

	testEvents := func(count int) []*event {
		e := make([]*event, count)
		for i := 0; i < count; i++ {
			e[i] = &event{ofs: uint64(i)}
		}
		return e
	}

	sequenceTest := func(evCount int) {
		fmt.Println(fmt.Sprintf("### sequence test: %d events:", evCount))
		require := require.New(b)
		events := testEvents(evCount)

		c := New[uint64, IEvent](evCount, func(ofs uint64, e IEvent) {
			require.Fail("unexpected event eviction", "offset: %v", ofs)
		})

		var put, get int64
		b.Run("Put", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, e := range events {
					c.Put(e.Ofs(), e)
				}
			}
			put = b.Elapsed().Nanoseconds() / int64(b.N) / int64(evCount)
		})

		b.Run("Get", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, e := range events {
					ce, ok := c.Get(e.Ofs())
					require.True(ok)
					require.Equal(e, ce)
				}
			}
			get = b.Elapsed().Nanoseconds() / int64(b.N) / int64(evCount)
		})
		fmt.Printf("\t— Put:\t%10d ns/op; Get:\t%10d ns/op\n", put, get)
	}

	sequenceTest(100)
	sequenceTest(1000)
	sequenceTest(10000)
	sequenceTest(100000)
}
