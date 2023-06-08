/*
 *
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 *
 */

package in10nmem

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
)

// go: noinline
func reInsert100RandomValues(m map[int64]int64) {

	for key := range m {
		delete(m, key)
	}

	for i := 0; i < 100; i++ {
		key := rand.Int63()   // generate random int64
		value := rand.Int63() // generate random int64
		m[key] = value
	}
}

func Benchmark_Int64Map_Existing(b *testing.B) {

	m := make(map[int64]int64)

	for i := 0; i < b.N; i++ {
		reInsert100RandomValues(m)
	}
}

func Benchmark_Int64Map_New(b *testing.B) {

	for i := 0; i < b.N; i++ {
		m := make(map[int64]int64)
		reInsert100RandomValues(m)
	}
}

// go: noinline
func createChannel[T any]() chan T {
	return make(chan T)
}

// go: noinline
func closeChannel[T any](c chan T) {
	close(c)
}

func Benchmark_CreateAndCloseChannel(b *testing.B) {

	for i := 0; i < b.N; i++ {
		c := createChannel[int]()
		closeChannel(c)
	}
}

type fastProjection struct {
	pvalue atomic.Value
}

type fastValueType struct {
	sigchan chan struct{}
	offset  int64
}

func Test_UseChannelAsASignal(t *testing.T) {

	const lastOffset = 10

	wg := sync.WaitGroup{}

	watcher := func(p *fastProjection) {
		for {
			pvalue := p.pvalue.Load().(fastValueType)
			println(pvalue.offset)
			if pvalue.offset == lastOffset {
				break
			}
			<-pvalue.sigchan
		}
		wg.Done()
	}

	p := fastProjection{}

	vold := fastValueType{
		sigchan: make(chan struct{}),
		offset:  0,
	}

	p.pvalue.Store(vold)

	wg.Add(1)
	go watcher(&p)

	for i := 0; i <= lastOffset; i++ {

		vnew := fastValueType{
			sigchan: make(chan struct{}),
			offset:  int64(i),
		}
		p.pvalue.Store(vnew)
		close(vold.sigchan)
		vold = vnew
		println("stored ", i)
	}

	wg.Wait()
}

func Benchmark_FastValueCreation(b *testing.B) {

	p := fastProjection{}

	vold := fastValueType{
		sigchan: make(chan struct{}),
		offset:  0,
	}

	for i := 0; i < b.N; i++ {
		vnew := fastValueType{
			sigchan: make(chan struct{}),
			offset:  int64(i),
		}
		p.pvalue.Store(vnew)
		close(vold.sigchan)
		vold = vnew
	}
}
