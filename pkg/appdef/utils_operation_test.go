/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/set"
)

func Test_AllOperationsForType(t *testing.T) {

	tests := []struct {
		appdef.TypeKind
		want set.Set[appdef.OperationKind]
	}{
		{appdef.TypeKind_null, set.Empty[appdef.OperationKind]()},
		{appdef.TypeKind_GRecord, set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select)},
		{appdef.TypeKind_CDoc, set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select)},
		{appdef.TypeKind_ViewRecord, set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select)},
		{appdef.TypeKind_Command, set.From(appdef.OperationKind_Execute)},
		{appdef.TypeKind_Role, set.From(appdef.OperationKind_Inherits)},
		{appdef.TypeKind_Projector, set.Empty[appdef.OperationKind]()},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.TypeKind.TrimString(), func(t *testing.T) {
			if got := appdef.ACLOperationsForType(tt.TypeKind); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AllOperationsForType(%v) = %v, want %v", tt.TypeKind, got, tt.want)
			}
		})
	}
}
