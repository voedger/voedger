/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTime_BasicUsage(t *testing.T) {
	require := require.New(t)
	tm := NewITime()

	t.Run("now", func(t *testing.T) {
		now := tm.Now()
		require.Less(now.UnixMilli()-time.Now().UnixMilli(), int64(10*time.Millisecond))
	})

	t.Run("timer", func(t *testing.T) {
		timer := tm.NewTimer(100 * time.Millisecond)
		start := tm.Now()
		firingInstant := <-timer
		require.Less(firingInstant.UnixMilli()-start.UnixMilli(), int64(130)) // 30 msecs lag for slow PCs
	})
}

func TestMockTime(t *testing.T) {
	require := require.New(t)
	t.Run("now", func(t *testing.T) {
		tm1 := MockTime.Now()
		time.Sleep(10 * time.Millisecond)
		tm2 := MockTime.Now()
		require.Equal(tm1, tm2)
	})

	t.Run("add", func(t *testing.T) {
		tm1 := MockTime.Now()
		MockTime.Add(1 * time.Minute)
		tm2 := MockTime.Now()
		require.Equal(time.Minute, tm2.Sub(tm1))
	})

	t.Run("timer", func(t *testing.T) {
		timer1 := MockTime.NewTimer(123 * time.Hour)
		timer2 := MockTime.NewTimer(125 * time.Hour)
		MockTime.Add(122 * time.Hour)
		select {
		case <-timer1:
			t.Fatal(1)
		case <-timer2:
			t.Fatal(2)
		default:
		}

		MockTime.Add(2 * time.Hour) // +124 hours

		select {
		case <-timer1:
		case <-timer2:
			t.Fatal(3)
		default:
			t.Fatal(4)
		}

		MockTime.Add(time.Hour) // +125 hours

		select {
		case <-timer1:
			t.Fatal(5)
		case <-timer2:
		default:
			t.Fatal(6)
		}
	})
}
