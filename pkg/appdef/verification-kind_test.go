/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"
)

func TestVerificationKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{
			name: "0 —> `VerificationKind_EMail`",
			k:    VerificationKind_EMail,
			want: `VerificationKind_EMail`,
		},
		{
			name: "1 —> `VerificationKind_Phone`",
			k:    VerificationKind_Phone,
			want: `VerificationKind_Phone`,
		},
		{
			name: "3 —> `3`",
			k:    VerificationKind_FakeLast + 1,
			want: `VerificationKind(3)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.String(); got != tt.want {
				t.Errorf("VerificationKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerificationKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{
			name: `0 —> "VerificationKind_EMail"`,
			k:    VerificationKind_EMail,
			want: `"VerificationKind_EMail"`,
		},
		{
			name: `1 —> "VerificationKind_Phone"`,
			k:    VerificationKind_Phone,
			want: `"VerificationKind_Phone"`,
		},
		{
			name: "2 —> 2",
			k:    VerificationKind_FakeLast,
			want: `2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalJSON()
			if err != nil {
				t.Errorf("VerificationKind.MarshalJSON() return unexpected error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("VerificationKind.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerificationKind_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    VerificationKind
		wantErr bool
	}{
		{
			name:    `0 —> VerificationKind_Email`,
			data:    `0`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `1 —> VerificationKind_Phone`,
			data:    `1`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `2 —> VerificationKind(2)`,
			data:    `2`,
			want:    VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `3 —> VerificationKind(3)`,
			data:    `3`,
			want:    VerificationKind(3),
			wantErr: false,
		},
		{
			name:    `"VerificationKind_EMail" —> VerificationKind_EMail`,
			data:    `"VerificationKind_EMail"`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"VerificationKind_Phone" —> VerificationKind_Phone`,
			data:    `"VerificationKind_Phone"`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"0" —> VerificationKind_Email`,
			data:    `"0"`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"1" —> VerificationKind_Phone`,
			data:    `"1"`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"2" —> VerificationKind(2)`,
			data:    `"2"`,
			want:    VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `"3" —> VerificationKind(3)`,
			data:    `"3"`,
			want:    VerificationKind(3),
			wantErr: false,
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
			var k VerificationKind
			err := k.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("VerificationKind.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if k != tt.want {
					t.Errorf("VerificationKind.UnmarshalJSON(%v) result = %v, want %v", tt.data, k, tt.want)
				}
			}
		})
	}
}

func TestVerificationKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{name: "basic test", k: VerificationKind_EMail, want: "EMail"},
		{name: "out of range", k: VerificationKind_FakeLast + 1, want: (VerificationKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("VerificationKind.ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
