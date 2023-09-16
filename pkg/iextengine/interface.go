/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextengine

import (
	"context"
	"net/url"

	istructs "github.com/voedger/voedger/pkg/istructs"
)

type IExtensionsModule interface {
	// Returns URL to a resource
	//
	// Example: file:///home/user1/myextension.wasm
	GetURL() string
}

type ExtEngineConfig struct {
	// MemoryLimitPages limits the maximum memory pages available to the extension
	// 1 page = 2^16 bytes.
	//
	// Default value is 2^8 so the total available memory is 2^24 bytes
	MemoryLimitPages uint
}

type IExtentionIO interface {
	istructs.IState
	istructs.IIntents
}

// 1 package = 1 ext engine instance
//
// Extension engine is not thread safe
type IExtensionEngine interface {
	Invoke(ctx context.Context, extentionName string, io IExtentionIO) (err error)
	Close(ctx context.Context)
}

type ExtensionEngineFactory = func(context context.Context, moduleURL *url.URL, extensionNames []string, config ExtEngineConfig) (e IExtensionEngine, err error)
