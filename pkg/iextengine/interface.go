/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextengine

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type IExtensionsModule interface {
	// Returns URL to a resource
	//
	// Example: file:///home/user1/myextension.wasm
	GetURL() string
}

type ExtensionLimits struct {

	// Default is 0 (execution interval not specified)
	ExecutionInterval time.Duration
}

type ExtEngineConfig struct {
	// MemoryLimitPages limits the maximum memory pages available to the extension
	// 1 page = 2^16 bytes.
	//
	// Default value is 2^8 so the total available memory is 2^24 bytes
	MemoryLimitPages uint
}

type IExtensionIO interface {
	istructs.IState
	istructs.IIntents
}

// 1 package = 1 ext engine instance
//
// Extension engine is not thread safe
type IExtensionEngine interface {
	SetLimits(limits ExtensionLimits)
	Invoke(ctx context.Context, extentionName string, io IExtensionIO) (err error)
	Close()
}

type IExtensionEngineFactories interface {
	QueryFactory(appdef.ExtensionEngineKind) IExtensionEngineFactory
}

type IExtensionEngineFactory interface {
	// LocalPath is a path package data can be got from
	// - packageNameToLocalPath is not used for ExtensionEngineKind_BuiltIn
	// - config is not used for ExtensionEngineKind_BuiltIn
	New(packageNameToLocalPath map[string]string, config ExtEngineConfig, numEngines int) []IExtensionEngine
}
