/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
)

func Test_IsSysField(t *testing.T) {
	type args struct {
		name appdef.FieldName
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.QName",
			args: args{appdef.SystemField_QName},
			want: true,
		},
		{
			name: "true if sys.ID",
			args: args{appdef.SystemField_ID},
			want: true,
		},
		{
			name: "true if sys.ParentID",
			args: args{appdef.SystemField_ParentID},
			want: true,
		},
		{
			name: "true if sys.Container",
			args: args{appdef.SystemField_Container},
			want: true,
		},
		{
			name: "true if sys.IsActive",
			args: args{appdef.SystemField_IsActive},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if basic user",
			args: args{"userField"},
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
			if got := appdef.IsSysField(tt.args.name); got != tt.want {
				t.Errorf("sysField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerificationKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.VerificationKind
		want string
	}{
		{
			name: "0 —> `VerificationKind_EMail`",
			k:    appdef.VerificationKind_EMail,
			want: `VerificationKind_EMail`,
		},
		{
			name: "1 —> `VerificationKind_Phone`",
			k:    appdef.VerificationKind_Phone,
			want: `VerificationKind_Phone`,
		},
		{
			name: "3 —> `3`",
			k:    appdef.VerificationKind_FakeLast + 1,
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
		k    appdef.VerificationKind
		want string
	}{
		{
			name: `0 —> "VerificationKind_EMail"`,
			k:    appdef.VerificationKind_EMail,
			want: `"VerificationKind_EMail"`,
		},
		{
			name: `1 —> "VerificationKind_Phone"`,
			k:    appdef.VerificationKind_Phone,
			want: `"VerificationKind_Phone"`,
		},
		{
			name: "2 —> 2",
			k:    appdef.VerificationKind_FakeLast,
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
		want    appdef.VerificationKind
		wantErr bool
	}{
		{
			name:    `0 —> VerificationKind_Email`,
			data:    `0`,
			want:    appdef.VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `1 —> VerificationKind_Phone`,
			data:    `1`,
			want:    appdef.VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `2 —> VerificationKind(2)`,
			data:    `2`,
			want:    appdef.VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `3 —> VerificationKind(3)`,
			data:    `3`,
			want:    appdef.VerificationKind(3),
			wantErr: false,
		},
		{
			name:    `"VerificationKind_EMail" —> VerificationKind_EMail`,
			data:    `"VerificationKind_EMail"`,
			want:    appdef.VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"VerificationKind_Phone" —> VerificationKind_Phone`,
			data:    `"VerificationKind_Phone"`,
			want:    appdef.VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"0" —> VerificationKind_Email`,
			data:    `"0"`,
			want:    appdef.VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"1" —> VerificationKind_Phone`,
			data:    `"1"`,
			want:    appdef.VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"2" —> VerificationKind(2)`,
			data:    `"2"`,
			want:    appdef.VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `"3" —> VerificationKind(3)`,
			data:    `"3"`,
			want:    appdef.VerificationKind(3),
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
			var k appdef.VerificationKind
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
		k    appdef.VerificationKind
		want string
	}{
		{name: "basic test", k: appdef.VerificationKind_EMail, want: "EMail"},
		{name: "out of range", k: appdef.VerificationKind_FakeLast + 1, want: (appdef.VerificationKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(appdef.VerificationKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
