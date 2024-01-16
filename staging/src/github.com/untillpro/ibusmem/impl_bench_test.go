/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package ibusmem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

// BenchmarkSectionedRequestResponse/#00-4    51944	     23527 ns/op	     42505 rps	    4229 B/op	      69 allocs/op
func BenchmarkSectionedRequestResponse(b *testing.B) {
	bus := Provide(func(requestCtx context.Context, bus ibus.IBus, sender interface{}, request ibus.Request) {
		rs := bus.SendParallelResponse2(sender)
		go func() {
			require.Nil(b, rs.ObjectSection("secObj", []string{"meta"}, "elem"))
			rs.StartMapSection("secMap", []string{"classifier", "2"})
			require.Nil(b, rs.SendElement("id1", "elem"))
			require.Nil(b, rs.SendElement("id2", "elem"))
			rs.StartArraySection("secArr", []string{"classifier", "4"})
			require.Nil(b, rs.SendElement("", "arrEl1"))
			require.Nil(b, rs.SendElement("", "arrEl2"))
			rs.StartMapSection("deps", []string{"classifier", "3"})
			require.Nil(b, rs.SendElement("id3", "elem"))
			require.Nil(b, rs.SendElement("id4", "elem"))
			rs.Close(errors.New("test error"))
		}()
	})

	b.Run("", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_, sections, _, _ := bus.SendRequest2(ctx, ibus.Request{}, ibus.DefaultTimeout)

			section := <-sections
			secObj := section.(ibus.IObjectSection)
			secObj.Value(ctx)

			section = <-sections
			secMap := section.(ibus.IMapSection)
			secMap.Next(ctx)
			secMap.Next(ctx)

			section = <-sections
			secArr := section.(ibus.IArraySection)
			secArr.Next(ctx)
			secArr.Next(ctx)

			section = <-sections
			secMap = section.(ibus.IMapSection)
			secMap.Next(ctx)
			secMap.Next(ctx)

			if _, ok := <-sections; ok {
				b.Fatal()
			}
		}
		elapsed := time.Since(start).Seconds()
		b.ReportMetric(float64(b.N)/elapsed, "rps")
	})
}

func BenchmarkOneSectionElement(b *testing.B) {
	bus := Provide(func(requestCtx context.Context, bus ibus.IBus, sender interface{}, request ibus.Request) {
		rs := bus.SendParallelResponse2(sender)
		go func() {
			require.Nil(b, rs.ObjectSection("secObj", nil, "hello"))
			rs.Close(nil)
		}()
	})

	b.Run("", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_, sections, _, _ := bus.SendRequest2(ctx, ibus.Request{}, ibus.DefaultTimeout)

			section := <-sections
			secObj := section.(ibus.IObjectSection)
			if len(secObj.Value(ctx)) != 7 { // "hello"
				b.Fatal()
			}
		}
		elapsed := time.Since(start).Seconds()
		b.ReportMetric(float64(b.N)/elapsed, "rps")
	})
}

func BenchmarkNonsectionedResponse(b *testing.B) {
	bus := Provide(func(requestCtx context.Context, bus ibus.IBus, sender interface{}, request ibus.Request) {
		bus.SendResponse(sender, ibus.Response{
			Data: []byte("hello"),
		})
	})

	b.Run("", func(b *testing.B) {
		start := time.Now()
		for i := 0; i < b.N; i++ {
			resp, _, _, _ := bus.SendRequest2(context.Background(), ibus.Request{}, ibus.DefaultTimeout)
			if len(resp.Data) != 5 {
				b.Fatal()
			}
		}
		elapsed := time.Since(start).Seconds()
		b.ReportMetric(float64(b.N)/elapsed, "rps")
	})
}
