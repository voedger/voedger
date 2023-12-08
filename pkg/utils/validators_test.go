/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestMatchQNames(t *testing.T) {
	require := require.New(t)
	myQName := appdef.NewQName("sys", "myQName")
	myQName2 := appdef.NewQName("sys", "myQName2")
	myQName3 := appdef.NewQName("sys", "myQName3")
	matcherFunc := MatchQName(myQName, myQName2)
	tests := []struct {
		cudRow   istructs.ICUDRow
		expected bool
	}{
		{
			cudRow:   &TestObject{Name: myQName},
			expected: true,
		},
		{
			cudRow:   &TestObject{Name: myQName2},
			expected: true,
		},
		{
			cudRow:   &TestObject{Name: myQName3},
			expected: false,
		},
	}
	for _, ts := range tests {
		require.Equal(ts.expected, matcherFunc(ts.cudRow, istructs.NullWSID, appdef.NullQName))
	}
}
