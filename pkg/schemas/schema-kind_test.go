/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"strconv"
	"testing"
)

func TestSchemaKindType_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    SchemaKind
		want string
	}{
		{name: `0 —> "SchemaKind_null"`,
			k:    SchemaKind_null,
			want: `SchemaKind_null`,
		},
		{name: `1 —> "SchemaKind_GDoc"`,
			k:    SchemaKind_GDoc,
			want: `SchemaKind_GDoc`,
		},
		{name: `SchemaKind_FakeLast —> 17`,
			k:    SchemaKind_FakeLast,
			want: strconv.FormatUint(uint64(SchemaKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("SchemaKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("SchemaKind.MarshalText() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover SchemaKind.String()", func(t *testing.T) {
		const tested = SchemaKind_FakeLast + 1
		want := "SchemaKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(SchemaKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}
