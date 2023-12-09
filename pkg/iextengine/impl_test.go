/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/
package iextengine

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

func Test_BasicUsage(t *testing.T) {
	factories := ProvideExtensionEngineFactories()
	require.Nil(t, factories.QueryFactory(appdef.ExtensionEngineKind_BuiltIn))
	require.Nil(t, factories.QueryFactory(appdef.ExtensionEngineKind_WASM))
	require.Panics(t, func() {
		factories.QueryFactory(appdef.ExtensionEngineKind(100))
	}, "undefined extension engine kind")
}
