/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/objcache"
)

type TckCache = objcache.ICache[IOffset, IEvent]

type IOffset interface {
	Partition() uint64
	Offset() uint64
}

type IEvent interface {
	Partition() uint64
	Offset() uint64
	Data() []byte
}

type IBomber interface {
	PutEvents(TckCache)
	GetEvents(TckCache)
}

func NewOffset(p, o uint64) IOffset {
	return &offset{p, o}
}

func NewEvent(p, o uint64) IEvent {
	const eventSize = 1024
	return &event{p, o, make([]byte, eventSize)}
}

func NewBomber(part uint64, maxOfs uint64) IBomber {
	return &bomber{
		part:   part,
		events: newEvents(part, maxOfs),
	}
}

type offset struct {
	p, o uint64
}

func (k offset) Partition() uint64 { return k.p }
func (k offset) Offset() uint64    { return k.o }

type event struct {
	p, o uint64
	data []byte
}

func (e event) Partition() uint64 { return e.p }
func (e event) Offset() uint64    { return e.o }
func (e event) Data() []byte      { return e.data }

type events map[IOffset]IEvent

func newEvents(part uint64, maxOfs uint64) events {
	events := make(map[IOffset]IEvent)
	for o := uint64(0); o < maxOfs; o++ {
		k := NewOffset(part, o)
		e := NewEvent(part, o)
		events[k] = e
	}
	return events
}

type bomber struct {
	part uint64
	events
}

func (b *bomber) PutEvents(cache objcache.ICache[IOffset, IEvent]) {
	for k, e := range b.events {
		cache.Put(k, e)
	}
}

func (b *bomber) GetEvents(cache objcache.ICache[IOffset, IEvent]) {
	for k := range b.events {
		_, ok := cache.Get(k)
		if !ok {
			panic(fmt.Errorf("missed event in cache, partition:%v, offset: %v", k.Partition(), k.Offset()))
		}
	}
}
