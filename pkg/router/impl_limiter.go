/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors"
)

func (l *wsQueryLimiter) acquire(wsid istructs.WSID) bool {
	if l.maxQPerWS <= 0 {
		return true
	}
	val, ok := l.counters.Load(wsid)
	if !ok {
		val, _ = l.counters.LoadOrStore(wsid, &atomic.Int32{})
	}
	counter := val.(*atomic.Int32)
	for {
		current := counter.Load()
		if int(current) >= l.maxQPerWS {
			return false
		}
		if counter.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func (l *wsQueryLimiter) release(wsid istructs.WSID) {
	val, ok := l.counters.Load(wsid)
	if !ok {
		return
	}
	val.(*atomic.Int32).Add(-1)
}

func (l *wsQueryLimiter) onQueryDrop(requestCtx context.Context, wsid istructs.WSID, extension string) {
	l.mu.Lock()
	key := rejectionKey{wsid: wsid, extension: extension}
	rc := l.rejections[key]
	if rc == nil {
		rc = &rejectionCounter{}
		l.rejections[key] = rc
	}
	rc.count++
	rc.logCtxFromLastQuery = requestCtx
	if atomic.LoadInt64(&l.lastLoggedAt) == 0 {
		atomic.StoreInt64(&l.lastLoggedAt, l.iTime.Now().UnixNano())
	}
	l.mu.Unlock()
	l.tryFlush()
}

func (l *wsQueryLimiter) tryFlush() {
	lastLoggedAt := atomic.LoadInt64(&l.lastLoggedAt)
	if lastLoggedAt == 0 || l.iTime.Now().UnixNano()-lastLoggedAt < int64(rejectionLogInterval) {
		return
	}
	l.mu.Lock()
	if l.lastLoggedAt == 0 || l.iTime.Now().UnixNano()-l.lastLoggedAt < int64(rejectionLogInterval) {
		l.mu.Unlock()
		return
	}
	entries := l.rejections
	if len(entries) == 0 {
		atomic.StoreInt64(&l.lastLoggedAt, 0)
		l.mu.Unlock()
		return
	}
	l.rejections = make(map[rejectionKey]*rejectionCounter)
	atomic.StoreInt64(&l.lastLoggedAt, l.iTime.Now().UnixNano())
	l.mu.Unlock()
	logRejections(entries)
}

func (l *wsQueryLimiter) flushAll() {
	l.mu.Lock()
	entries := l.rejections
	l.rejections = make(map[rejectionKey]*rejectionCounter)
	atomic.StoreInt64(&l.lastLoggedAt, 0)
	l.mu.Unlock()
	logRejections(entries)
}

func logRejections(entries map[rejectionKey]*rejectionCounter) {
	for _, rc := range entries {
		if rc.count > 0 {
			logger.WarningCtx(rc.logCtxFromLastQuery, "routing.qp.limit",
				fmt.Sprintf("droppedInLast10Seconds=%d", rc.count))
		}
	}
}

func isQPBoundAPIPath(apiPath processors.APIPath) bool {
	switch apiPath {
	case processors.APIPath_Queries, processors.APIPath_Views, processors.APIPath_Docs, processors.APIPath_CDocs:
		return true
	}
	return false
}
