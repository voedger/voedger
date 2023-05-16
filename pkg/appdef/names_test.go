/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"testing"
)

func Test_ValidIdent(t *testing.T) {
	type args struct {
		ident string
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr error
	}{
		// negative tests
		{
			name:    "error if empty ident",
			args:    args{ident: ""},
			wantOk:  false,
			wantErr: ErrNameMissed,
		},
		{
			name:    "error if wrong first char",
			args:    args{ident: "🐧26"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong digit starts",
			args:    args{ident: "2abc"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong last char",
			args:    args{ident: "lookAt🐧"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if wrong char anywhere",
			args:    args{ident: "This🐧isMy"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if starts from digit",
			args:    args{ident: "7zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces at begin",
			args:    args{ident: " zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces at end",
			args:    args{ident: "zip "},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name:    "error if spaces anywhere",
			args:    args{ident: "zip zip"},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		{
			name: "error if too long",
			args: args{ident: func() string {
				worm := "_"
				for i := 0; i < MaxIdentLen; i++ {
					worm += "_"
				}
				return worm
			}()},
			wantOk:  false,
			wantErr: ErrInvalidName,
		},
		// positive tests
		{
			name:   "one letter must pass",
			args:   args{ident: "i"},
			wantOk: true,
		},
		{
			name:   "single underscore must pass",
			args:   args{ident: "_"},
			wantOk: true,
		},
		{
			name:   "starts from underscore must pass",
			args:   args{ident: "_test"},
			wantOk: true,
		},
		{
			name:   "basic camel notation must pass",
			args:   args{ident: "thisIsIdent1"},
			wantOk: true,
		},
		{
			name:   "basic snake notation must pass",
			args:   args{ident: "this_is_ident_2"},
			wantOk: true,
		},
		{
			name:   "mixed notation must pass",
			args:   args{ident: "useMix_4_fun"},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := ValidIdent(tt.args.ident)
			if gotOk != tt.wantOk {
				t.Errorf("ValidIdent() = %v, want %v", gotOk, tt.wantOk)
				return
			}
			if err != nil {
				if tt.wantErr == nil {
					t.Errorf("ValidIdent() error = %v, wantErr is nil", err)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidIdent() error = %v not is %v", err, tt.wantErr)
					return
				}
			} else if tt.wantErr != nil {
				t.Errorf("ValidIdent() error = nil, wantErr - %v", tt.wantErr)
				return
			}
		})
	}
}

func Test_IsSysField(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.QName",
			args: args{SystemField_QName},
			want: true,
		},
		{
			name: "true if sys.ID",
			args: args{SystemField_ID},
			want: true,
		},
		{
			name: "true if sys.ParentID",
			args: args{SystemField_ParentID},
			want: true,
		},
		{
			name: "true if sys.Container",
			args: args{SystemField_Container},
			want: true,
		},
		{
			name: "true if sys.IsActive",
			args: args{SystemField_IsActive},
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
			if got := IsSysField(tt.args.name); got != tt.want {
				t.Errorf("sysField() = %v, want %v", got, tt.want)
			}
		})
	}
}
