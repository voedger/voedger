/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"errors"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
)

// ttlStorageMock is a thread-safe in-memory mock of ITTLStorage that supports key expiration.
// [~server.design.orch/ttlStorageMock~impl]
type ttlStorageMock[K comparable, V comparable] struct {
	mu                     sync.Mutex
	data                   map[K]valueWithTTL[V]
	expirations            map[K]time.Time // expiration time for each key
	errorTrigger           map[K]bool
	tm                     timeu.ITime
	onBeforeCompareAndSwap func() // != nil -> called right before CompareAndSwap. Need to implement hook in tests
}

func newTTLStorageMock[K comparable, V comparable]() *ttlStorageMock[K, V] {
	return &ttlStorageMock[K, V]{
		data:         make(map[K]valueWithTTL[V]),
		expirations:  make(map[K]time.Time),
		errorTrigger: make(map[K]bool),
		tm:           testingu.MockTime,
	}
}

func (m *ttlStorageMock[K, V]) pruneExpired() {
	now := m.tm.Now()
	for k, v := range m.data {
		if now.After(v.expiresAt) {
			delete(m.data, k)
		}
	}
}

type valueWithTTL[V any] struct {
	value     V
	expiresAt time.Time
}

// causeErrorIfNeeded simulates a forced error for specific keys
func (m *ttlStorageMock[K, V]) causeErrorIfNeeded(key K) error {
	if m.errorTrigger[key] {
		return errors.New("forced storage error for test")
	}
	return nil
}

func (m *ttlStorageMock[K, V]) InsertIfNotExist(key K, val V, ttlSeconds int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	// Check if key exists
	if _, exists := m.data[key]; exists {
		return false, nil
	}

	// Insert new value with TTL
	m.data[key] = valueWithTTL[V]{
		value:     val,
		expiresAt: m.tm.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	return true, nil
}

func (m *ttlStorageMock[K, V]) CompareAndSwap(key K, oldVal V, newVal V, ttlSeconds int) (bool, error) {
	if m.onBeforeCompareAndSwap != nil {
		m.onBeforeCompareAndSwap()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	// Check if key exists and value matches
	if entry, exists := m.data[key]; !exists || entry.value != oldVal {
		return false, nil
	}

	// Update value and TTL
	m.data[key] = valueWithTTL[V]{
		value:     newVal,
		expiresAt: m.tm.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}

	return true, nil
}

// CompareAndDelete removes the key if current value matches val. Expiration is also removed.
func (m *ttlStorageMock[K, V]) CompareAndDelete(key K, val V) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, err
	}

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}
	if entry.value != val {
		return false, nil
	}

	delete(m.data, key)
	delete(m.expirations, key)
	return true, nil
}

func (m *ttlStorageMock[K, V]) Get(key K) (ok bool, val V, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpired()
	if err := m.causeErrorIfNeeded(key); err != nil {
		return false, val, err
	}

	entry, ok := m.data[key]
	return ok, entry.value, nil
}
