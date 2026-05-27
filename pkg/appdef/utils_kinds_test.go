/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/strconvu"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

type (
	baseEnum interface {
		~uint8
		String() string
	}

	enumKind interface {
		baseEnum
		MarshalText() ([]byte, error)
	}

	enumTrimmer interface {
		baseEnum
		TrimString() string
	}

	enumCase[T baseEnum] struct {
		name string
		k    T
		want string
	}
)

func testEnumMarshalText[T enumKind](t *testing.T, kindName string, count T, cases []enumCase[T]) {
	t.Helper()
	require := require.New(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.k.MarshalText()
			require.NoError(err)
			require.Equal(tc.want, string(got))
		})
	}
	t.Run("out-of-range String() coverage", func(t *testing.T) {
		oor := count + 1
		require.Equal(kindName+"("+strconvu.UintToString(oor)+")", oor.String())
	})
}

func testEnumTrimString[T enumTrimmer](t *testing.T, count T, cases []enumCase[T]) {
	t.Helper()
	require := require.New(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(tc.want, tc.k.TrimString())
		})
	}
	t.Run("out-of-range matches String()", func(t *testing.T) {
		oor := count + 1
		require.Equal(oor.String(), oor.TrimString())
	})
}

func TestEnumKinds_MarshalText(t *testing.T) {
	t.Run("ConstraintKind", func(t *testing.T) {
		testEnumMarshalText(t, "ConstraintKind", appdef.ConstraintKind_count, []enumCase[appdef.ConstraintKind]{
			{name: "null", k: appdef.ConstraintKind_null, want: "ConstraintKind_null"},
			{name: "MinLen", k: appdef.ConstraintKind_MinLen, want: "ConstraintKind_MinLen"},
			{name: "boundary", k: appdef.ConstraintKind_count, want: strconvu.UintToString(appdef.ConstraintKind_count)},
		})
	})
	t.Run("DataKind", func(t *testing.T) {
		testEnumMarshalText(t, "DataKind", appdef.DataKind_FakeLast, []enumCase[appdef.DataKind]{
			{name: "null", k: appdef.DataKind_null, want: "DataKind_null"},
			{name: "int8", k: appdef.DataKind_int8, want: "DataKind_int8"},
			{name: "int32", k: appdef.DataKind_int32, want: "DataKind_int32"},
			{name: "boundary", k: appdef.DataKind_FakeLast, want: strconvu.UintToString(appdef.DataKind_FakeLast)},
		})
	})
	t.Run("ExtensionEngineKind", func(t *testing.T) {
		testEnumMarshalText(t, "ExtensionEngineKind", appdef.ExtensionEngineKind_count, []enumCase[appdef.ExtensionEngineKind]{
			{name: "null", k: appdef.ExtensionEngineKind_null, want: "ExtensionEngineKind_null"},
			{name: "BuiltIn", k: appdef.ExtensionEngineKind_BuiltIn, want: "ExtensionEngineKind_BuiltIn"},
			{name: "boundary", k: appdef.ExtensionEngineKind_count, want: strconvu.UintToString(appdef.ExtensionEngineKind_count)},
		})
	})
	t.Run("FilterKind", func(t *testing.T) {
		testEnumMarshalText(t, "FilterKind", appdef.FilterKind_count, []enumCase[appdef.FilterKind]{
			{name: "null", k: appdef.FilterKind_null, want: "FilterKind_null"},
			{name: "QNames", k: appdef.FilterKind_QNames, want: "FilterKind_QNames"},
			{name: "boundary", k: appdef.FilterKind_count, want: strconvu.UintToString(appdef.FilterKind_count)},
		})
	})
	t.Run("TypeKind", func(t *testing.T) {
		testEnumMarshalText(t, "TypeKind", appdef.TypeKind_count, []enumCase[appdef.TypeKind]{
			{name: "null", k: appdef.TypeKind_null, want: "TypeKind_null"},
			{name: "Data", k: appdef.TypeKind_Data, want: "TypeKind_Data"},
			{name: "GDoc", k: appdef.TypeKind_GDoc, want: "TypeKind_GDoc"},
			{name: "boundary", k: appdef.TypeKind_count, want: strconvu.UintToString(appdef.TypeKind_count)},
		})
	})
	t.Run("LimitFilterOption", func(t *testing.T) {
		testEnumMarshalText(t, "LimitFilterOption", appdef.LimitFilterOption_count, []enumCase[appdef.LimitFilterOption]{
			{name: "ALL", k: appdef.LimitFilterOption_ALL, want: "LimitFilterOption_ALL"},
			{name: "EACH", k: appdef.LimitFilterOption_EACH, want: "LimitFilterOption_EACH"},
			{name: "boundary", k: appdef.LimitFilterOption_count, want: strconvu.UintToString(appdef.LimitFilterOption_count)},
		})
	})
}

func TestEnumKinds_TrimString(t *testing.T) {
	t.Run("ConstraintKind", func(t *testing.T) {
		testEnumTrimString(t, appdef.ConstraintKind_count, []enumCase[appdef.ConstraintKind]{
			{name: "basic", k: appdef.ConstraintKind_MinLen, want: "MinLen"},
		})
	})
	t.Run("DataKind", func(t *testing.T) {
		testEnumTrimString(t, appdef.DataKind_FakeLast, []enumCase[appdef.DataKind]{
			{name: "basic", k: appdef.DataKind_int32, want: "int32"},
		})
	})
	t.Run("ExtensionEngineKind", func(t *testing.T) {
		testEnumTrimString(t, appdef.ExtensionEngineKind_count, []enumCase[appdef.ExtensionEngineKind]{
			{name: "basic", k: appdef.ExtensionEngineKind_BuiltIn, want: "BuiltIn"},
		})
	})
	t.Run("FilterKind", func(t *testing.T) {
		testEnumTrimString(t, appdef.FilterKind_count, []enumCase[appdef.FilterKind]{
			{name: "basic", k: appdef.FilterKind_QNames, want: "QNames"},
		})
	})
	t.Run("TypeKind", func(t *testing.T) {
		testEnumTrimString(t, appdef.TypeKind_count, []enumCase[appdef.TypeKind]{
			{name: "null", k: appdef.TypeKind_null, want: "null"},
			{name: "basic", k: appdef.TypeKind_CDoc, want: "CDoc"},
		})
	})
	t.Run("LimitFilterOption", func(t *testing.T) {
		testEnumTrimString(t, appdef.LimitFilterOption_count, []enumCase[appdef.LimitFilterOption]{
			{name: "basic", k: appdef.LimitFilterOption_ALL, want: "ALL"},
		})
	})
	t.Run("RateScope", func(t *testing.T) {
		testEnumTrimString(t, appdef.RateScope_count, []enumCase[appdef.RateScope]{
			{name: "basic", k: appdef.RateScope_AppPartition, want: "AppPartition"},
		})
	})
}
