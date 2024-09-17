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
	myQName := appdef.NewQName("sys", "myQName")
	myQName2 := appdef.NewQName("sys", "myQName2")
	myQName3 := appdef.NewQName("sys", "myQName3")
	matcherFunc := MatchQName(myQName, myQName2)
	tests := []struct {
		qName    appdef.QName
		expected bool
	}{
		{
			qName:    myQName,
			expected: true,
		},
		{
			qName:    myQName2,
			expected: true,
		},
		{
			qName:    myQName3,
			expected: false,
		},
	}
	for _, ts := range tests {
		cudRow := &TestObject{Name: ts.qName}
		require.Equal(t, ts.expected, matcherFunc(cudRow, istructs.NullWSID, appdef.NullQName))
	}
}
