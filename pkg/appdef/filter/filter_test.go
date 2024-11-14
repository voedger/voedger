/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
)

func TestFilter_And(t *testing.T) {
	f := filter{}
	if got := f.And(); got != nil {
		t.Errorf("filter.And() = %v, want nil", got)
	}
}

func TestFilter_Kind(t *testing.T) {
	f := filter{}
	if got := f.Kind(); got != appdef.FilterKind_null {
		t.Errorf("filter.Kind() = %v, want %v", got, appdef.FilterKind_null)
	}
}

func TestFilter_Not(t *testing.T) {
	f := filter{}
	if got := f.Not(); got != nil {
		t.Errorf("filter.Not() = %v, want nil", got)
	}
}

func TestFilter_Or(t *testing.T) {
	f := filter{}
	if got := f.Or(); got != nil {
		t.Errorf("filter.Or() = %v, want nil", got)
	}
}

func TestFilter_QNames(t *testing.T) {
	f := filter{}
	if got := f.QNames(); got != nil {
		t.Errorf("filter.QNames() = %v, want nil", got)
	}
}

func TestFilter_Match(t *testing.T) {
	f := filter{}
	if got := f.Match(appdef.NullType); got != false {
		t.Errorf("filter.Match() = %v, want false", got)
	}
}

func TestFilter_Matches(t *testing.T) {
	f := filter{}
	if got := f.Matches(nil); got != nil {
		t.Errorf("filter.Matches() = %v, want nil", got)
	}
}

func TestFilter_Tags(t *testing.T) {
	f := filter{}
	if got := f.Tags(); got != nil {
		t.Errorf("filter.Tags() = %v, want nil", got)
	}
}

func TestFilter_Types(t *testing.T) {
	f := filter{}
	if got := f.Types(); got != (appdef.TypeKindSet{}) {
		t.Errorf("filter.Types() = %v, want %v", got, appdef.TypeKindSet{})
	}
}
