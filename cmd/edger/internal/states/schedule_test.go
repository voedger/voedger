/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package states

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_isScheduledTimeArrived(t *testing.T) {
	type args struct {
		scheduledTime time.Time
		now           time.Time
	}
	now := time.Now()
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "scheduled time is not arrived",
			args: args{scheduledTime: now.Add(time.Minute), now: now},
			want: false,
		},
		{
			name: "exactly scheduled time",
			args: args{scheduledTime: now, now: now},
			want: true,
		},
		{
			name: "scheduled time is arrived",
			args: args{scheduledTime: now, now: now.Add(time.Minute)},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isScheduledTimeArrived(tt.args.scheduledTime, tt.args.now); got != tt.want {
				t.Errorf("isScheduledTimeArrived() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("cover provided interface function IsScheduledTimeArrived", func(t *testing.T) {
		require.New(t).True(IsScheduledTimeArrived(now))
	})
}
