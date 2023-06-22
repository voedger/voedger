/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package istoragecache

import (
	"bytes"
	"sync"
)

type StorageCacheConf struct {
	PreAllocatedBuffersCount int
	PreAllocatedBufferSize   int
	MaxBytes                 int
}

type keyPoolImpl struct {
	idle         []*bytes.Buffer
	active       []*bytes.Buffer
	capacity     int
	maxKeyLength int
	mu           *sync.Mutex
}

func newKeyPool(capacity, maxKeyLength int) keyPool {
	return &keyPoolImpl{
		idle:         make([]*bytes.Buffer, 0),
		active:       make([]*bytes.Buffer, 0),
		capacity:     capacity,
		maxKeyLength: maxKeyLength,
		mu:           &sync.Mutex{},
	}
}

func (p *keyPoolImpl) get(pKey []byte, cCols []byte) (bb *bytes.Buffer, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(pKey)+len(cCols) > p.maxKeyLength {
		return nil, errKeyLengthExceeded
	}

	if len(p.idle) == 0 && len(p.active) == p.capacity {
		return nil, errKeyPoolReachedMaximumCapacity
	} else if len(p.idle) == 0 {
		bb = bytes.NewBuffer(make([]byte, 0, p.maxKeyLength))
	} else {
		bb, p.idle = p.idle[0], p.idle[1:]
	}

	p.active = append(p.active, bb)

	_, err = bb.Write(pKey)
	_, err = bb.Write(cCols)

	return
}
func (p *keyPoolImpl) put(bb *bytes.Buffer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	bb.Reset()

	idx := p.findIndex(bb)

	p.active = append(p.active[:idx], p.active[idx+1:]...)
	p.idle = append(p.idle, bb)
}
func (p *keyPoolImpl) findIndex(bb *bytes.Buffer) int {
	for idx, item := range p.active {
		if bb == item {
			return idx
		}
	}
	panic("impossible")
}
