/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sqlquery

import (
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
)

func Test_parseQueryAppWs(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name      string
		args      args
		wantApp   istructs.AppQName
		wantWs    istructs.WSID
		wantClean string
		wantErr   bool
	}{
		{"fail: empty", args{""}, istructs.NullAppQName, 0, "", true},
		{"fail: select without table", args{"select"}, istructs.NullAppQName, 0, "", true},

		{"OK: select table", args{"select table"}, istructs.NullAppQName, 0, "select table", false},
		{"OK: select table args", args{"select table args"}, istructs.NullAppQName, 0, "select table args", false},
		{"OK: select owner.app.table", args{"select owner.app.table"}, istructs.MustParseAppQName("owner/app"), 0, "select table", false},
		{"OK: select owner.app.table args", args{"select owner.app.table args"}, istructs.MustParseAppQName("owner/app"), 0, "select table args", false},
		{"OK: select owner.app.123.table", args{"select owner.app.123.table"}, istructs.MustParseAppQName("owner/app"), 123, "select table", false},
		{"OK: select owner.app.123.table args", args{"select owner.app.123.table args"}, istructs.MustParseAppQName("owner/app"), 123, "select table args", false},
		{"OK: select 123.table", args{"select 123.table"}, istructs.NullAppQName, 123, "select table", false},
		{"OK: select 123.table args", args{"select 123.table args"}, istructs.NullAppQName, 123, "select table args", false},

		{"fail: select naked.🔫.table", args{"select naked.🔫.table"}, istructs.NullAppQName, 0, "", true},
		{"fail: select ups.table", args{"select ups.table"}, istructs.NullAppQName, 0, "", true},
		{"fail: select -123.table", args{"select -123.table"}, istructs.NullAppQName, 0, "", true},
		{"fail: select owner.app.ups.table", args{"select owner.app.ooo.table"}, istructs.NullAppQName, 0, "", true},
		{"fail: insert owner.app.100.table", args{"insert owner.app.100.table"}, istructs.NullAppQName, 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApp, gotWs, gotClean, err := parseQueryAppWs(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseQueryAppWs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotApp, tt.wantApp) {
				t.Errorf("parseQueryAppWs() gotApp = %v, want %v", gotApp, tt.wantApp)
			}
			if !reflect.DeepEqual(gotWs, tt.wantWs) {
				t.Errorf("parseQueryAppWs() gotWs = %v, want %v", gotWs, tt.wantWs)
			}
			if gotClean != tt.wantClean {
				t.Errorf("parseQueryAppWs() gotClean = %v, want %v", gotClean, tt.wantClean)
			}
		})
	}
}
