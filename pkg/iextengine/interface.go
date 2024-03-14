/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextengine

import (
	"context"
	"net/url"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
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

	// Compile bool
}

type IExtensionIO interface {
	istructs.IState
	istructs.IIntents
	istructs.IPkgNameResolver
}

// 1 package = 1 ext engine instance
//
// Extension engine is not thread safe
type IExtensionEngine interface {
	SetLimits(limits ExtensionLimits)
	Invoke(ctx context.Context, extName ExtQName, io IExtensionIO) (err error)
	Close(ctx context.Context)
}

type ExtensionEngineFactories map[appdef.ExtensionEngineKind]IExtensionEngineFactory

type ExtQName struct {
	PackagePath string
	ExtName     string
}

func (n ExtQName) String() string {
	return n.PackagePath + "." + n.ExtName
}

type BuiltInExtFunc func(ctx context.Context, io IExtensionIO) error
type BuiltInExtFuncs map[ExtQName]BuiltInExtFunc // Provided to construct factory of engines

type ExtensionPackage struct {
	QualifiedName  string
	ModuleUrl      *url.URL
	ExtensionNames []string
}

type IExtensionEngineFactory interface {
	// LocalPath is a path package data can be got from
	// - packages is not used for ExtensionEngineKind_BuiltIn
	// - config is not used for ExtensionEngineKind_BuiltIn
	New(ctx context.Context, packages []ExtensionPackage, config *ExtEngineConfig, numEngines int) ([]IExtensionEngine, error)
}
