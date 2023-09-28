/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"testing"

	"github.com/untillpro/dynobuffers"
)

func TestFieldTypeToString(t *testing.T) {
	type args struct {
		ft dynobuffers.FieldType
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "FieldTypeInt64", args: args{ft: dynobuffers.FieldTypeInt64}, want: "int64"},
		{name: "FieldTypeByte", args: args{ft: dynobuffers.FieldTypeByte}, want: "[]byte"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FieldTypeToString(tt.args.ft); got != tt.want {
				t.Errorf("FieldTypeToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
