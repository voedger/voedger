/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"errors"
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
		{appdef.TypeKind_GRecord, appdef.RecordsOperations},
		{appdef.TypeKind_CDoc, appdef.RecordsOperations},
		{appdef.TypeKind_ViewRecord, appdef.RecordsOperations},
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

func Test_IsCompatibleOperations(t *testing.T) {

	test := []struct {
		o    set.Set[appdef.OperationKind]
		want bool
		err  error
	}{
		{set.Set[appdef.OperationKind]{}, false, appdef.ErrMissedError},
		{set.From(appdef.OperationKind_Insert, appdef.OperationKind_Update), true, nil},
		{set.From(appdef.OperationKind_Execute), true, nil},
		{set.From(appdef.OperationKind_Insert, appdef.OperationKind_Execute), false, appdef.ErrIncompatibleError},
	}

	for _, tt := range test {
		got, err := appdef.IsCompatibleOperations(tt.o)
		if got != tt.want {
			t.Errorf("IsCompatibleOperations(%v) = %v, want %v", tt.o, got, err)
		} else if tt.err != nil && !errors.Is(err, tt.err) {
			t.Errorf("IsCompatibleOperations(%v) returns error %v, want %v", tt.o, got, err)
		}
	}
}
