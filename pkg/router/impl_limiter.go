/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

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

func (l *wsQueryLimiter) deferLogRejection(ctx context.Context, wsid istructs.WSID, extension string) {
	key := rejectionKey{wsid: wsid, extension: extension}
	val, _ := l.rejections.LoadOrStore(key, &rejectionCounter{})
	rc := val.(*rejectionCounter)
	rc.ctx.Store(ctx)
	rc.count.Add(1)
	if rc.timerActive.CompareAndSwap(false, true) {
		time.AfterFunc(rejectionLogInterval, func() {
			l.logPendingRejections(key, rc)
		})
	}
}

func (l *wsQueryLimiter) logPendingRejections(key rejectionKey, rc *rejectionCounter) {
	n := rc.count.Swap(0)
	if n > 0 {
		logCtx := rc.ctx.Load().(context.Context)
		logger.WarningCtx(logCtx, "routing.qp.limit",
			fmt.Sprintf("maxQPerWS=%d,rejectedInLastSecond=%d", l.maxQPerWS, n))
	}
	rc.timerActive.Store(false)
	if rc.count.Load() > 0 && rc.timerActive.CompareAndSwap(false, true) {
		time.AfterFunc(rejectionLogInterval, func() {
			l.logPendingRejections(key, rc)
		})
		return
	}
	l.rejections.Delete(key)
}

func (l *wsQueryLimiter) flushAll() {
	l.rejections.Range(func(k, v any) bool {
		rc := v.(*rejectionCounter)
		n := rc.count.Swap(0)
		if n > 0 {
			logCtx := rc.ctx.Load().(context.Context)
			logger.WarningCtx(logCtx, "routing.qp.limit",
				fmt.Sprintf("maxQPerWS=%d,rejectedInLastSecond=%d", l.maxQPerWS, n))
		}
		l.rejections.Delete(k)
		return true
	})
}

func isQPBoundAPIPath(apiPath processors.APIPath) bool {
	switch apiPath {
	case processors.APIPath_Queries, processors.APIPath_Views, processors.APIPath_Docs, processors.APIPath_CDocs:
		return true
	}
	return false
}
