/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package timeu

import (
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestTime_BasicUsage(t *testing.T) {
	require := require.New(t)
	tm := NewITime()

	t.Run("now", func(t *testing.T) {
		now := tm.Now()
		require.Less(now.UnixMilli()-time.Now().UnixMilli(), int64(10*time.Millisecond))
	})

	t.Run("timer", func(t *testing.T) {
		timer := tm.NewTimerChan(100 * time.Millisecond)
		start := tm.Now()
		firingInstant := <-timer
		require.Less(firingInstant.UnixMilli()-start.UnixMilli(), int64(100+30)) // 30 msecs lag for slow PCs
	})
}

func BenchmarkRealTimers(b *testing.B) {
	b.Run("create only", func(b *testing.B) {
		t := NewITime()
		for i := 0; i < b.N; i++ {
			_ = t.NewTimerChan(time.Hour)
		}
	})

	b.Run("create and fire", func(b *testing.B) {
		t := NewITime()
		for i := 0; i < b.N; i++ {
			_ = t.NewTimerChan(time.Millisecond)
			time.Sleep(2 * time.Millisecond)
		}
	})

	b.Run("create, fire and read", func(b *testing.B) {
		t := NewITime()
		for i := 0; i < b.N; i++ {
			timerChan := t.NewTimerChan(time.Millisecond)
			time.Sleep(2 * time.Millisecond)
			<-timerChan
		}
	})
}
