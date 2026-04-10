/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package router

import (
	"sync/atomic"

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

func (l *wsQueryLimiter) size() int {
	n := 0
	l.counters.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

func isQPBoundAPIPath(apiPath processors.APIPath) bool {
	switch apiPath {
	case processors.APIPath_Queries, processors.APIPath_Views, processors.APIPath_Docs, processors.APIPath_CDocs:
		return true
	}
	return false
}
