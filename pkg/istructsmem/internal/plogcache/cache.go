/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package plogcache

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/objcache"
)

// plog events cache
//
// Get() returns plog event by handling partition and offset.
//
// Put() puts plog event into cache.
type Cache struct {
	cache objcache.ICache[plogEvKey, istructs.IPLogEvent]
}

func New(size int) *Cache {
	cache := &Cache{
		cache: objcache.New[plogEvKey, istructs.IPLogEvent](size),
	}
	return cache
}

// Gets PLOG event on the key from the specified partition and offset
func (c *Cache) Get(partition istructs.PartitionID, offset istructs.Offset) (e istructs.IPLogEvent, ok bool) {
	return c.cache.Get(plogEvKey{partition, offset})
}

// Puts the specified PLOG event on the key from the specified partition and offset
func (c *Cache) Put(partition istructs.PartitionID, offset istructs.Offset, event istructs.IPLogEvent) {
	c.cache.Put(plogEvKey{partition, offset}, event)
}

type plogEvKey struct {
	partition istructs.PartitionID
	offset    istructs.Offset
}
