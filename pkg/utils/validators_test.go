/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func TestMatchQNames(t *testing.T) {
	myQName := appdef.NewQName("sys", "myQName")
	myQName2 := appdef.NewQName("sys", "myQName2")
	myQName3 := appdef.NewQName("sys", "myQName3")
	matcherFunc := MatchQName(myQName, myQName2)
	tests := []struct {
		cudQNames []appdef.QName
		expected  bool
	}{
		{
			cudQNames: []appdef.QName{myQName},
			expected:  true,
		},
		{
			cudQNames: []appdef.QName{myQName2},
			expected:  true,
		},
		{
			cudQNames: []appdef.QName{myQName3},
			expected:  false,
		},
	}
	for _, ts := range tests {
		require.Equal(t, ts.expected, TestMatchQNameFunc(matcherFunc, ts.cudQNames...))
	}

	require.False(t, TestMatchQNameFunc(matcherFunc))
}
