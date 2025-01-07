/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
)

func TestOccurs_String(t *testing.T) {
	tests := []struct {
		name string
		o    appdef.Occurs
		want string
	}{
		{
			name: "0 —> `0`",
			o:    0,
			want: `0`,
		},
		{
			name: "1 —> `1`",
			o:    1,
			want: `1`,
		},
		{
			name: "∞ —> `unbounded`",
			o:    appdef.Occurs_Unbounded,
			want: `unbounded`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.String(); got != tt.want {
				t.Errorf("Occurs.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOccurs_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		o       appdef.Occurs
		want    string
		wantErr bool
	}{
		{
			name:    "0 —> `0`",
			o:       0,
			want:    `0`,
			wantErr: false,
		},
		{
			name:    "1 —> `1`",
			o:       1,
			want:    `1`,
			wantErr: false,
		},
		{
			name:    "∞ —> `unbounded`",
			o:       appdef.Occurs_Unbounded,
			want:    `"unbounded"`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Occurs.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Occurs.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOccurs_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    appdef.Occurs
		wantErr bool
	}{
		{
			name:    "0 —> 0",
			data:    `0`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "1 —> 1",
			data:    `1`,
			want:    1,
			wantErr: false,
		},
		{
			name:    `"unbounded" —> ∞`,
			data:    `"unbounded"`,
			want:    appdef.Occurs_Unbounded,
			wantErr: false,
		},
		{
			name:    `"3" —> error`,
			data:    `"3"`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `65536 —> error`,
			data:    `65536`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `-1 —> error`,
			data:    `-1`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `"abc" —> error`,
			data:    `"abc"`,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o appdef.Occurs
			err := o.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Occurs.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if o != tt.want {
					t.Errorf("o.UnmarshalJSON() result = %v, want %v", o, tt.want)
				}
			}
		})
	}
}
