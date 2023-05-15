/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"
)

func Test_IsSysContainer(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.pkey",
			args: args{SystemContainer_ViewPartitionKey},
			want: true,
		},
		{
			name: "true if sys.ccols",
			args: args{SystemContainer_ViewClusteringCols},
			want: true,
		},
		{
			name: "true if sys.val",
			args: args{SystemContainer_ViewValue},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if basic user",
			args: args{"userContainer"},
			want: false,
		},
		{
			name: "false if curious user",
			args: args{"sys.user"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSysContainer(tt.args.name); got != tt.want {
				t.Errorf("IsSysContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}
