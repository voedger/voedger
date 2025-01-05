/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IExtension
type Extension struct {
	types.Typ
	name    string
	engine  appdef.ExtensionEngineKind
	states  *Storages
	intents *Storages
}

func MakeExtension(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Extension {
	return Extension{
		Typ:     types.MakeType(ws.App(), ws, name, kind),
		name:    name.Entity(),
		engine:  appdef.ExtensionEngineKind_BuiltIn,
		states:  NewStorages(ws.App()),
		intents: NewStorages(ws.App()),
	}
}

func (ex Extension) Intents() appdef.IStorages { return ex.intents }

func (ex Extension) Name() string { return ex.name }

func (ex Extension) Engine() appdef.ExtensionEngineKind { return ex.engine }

func (ex Extension) States() appdef.IStorages { return ex.states }

func (ex Extension) String() string {
	// BuiltIn-function «test.func»
	return fmt.Sprintf("%s-%v", ex.Engine().TrimString(), ex.Typ.String())
}

// Validates extension
//
// # Returns error:
//   - if storages (states or intents) contains unknown qname(s)
func (ex Extension) Validate() error {
	return errors.Join(
		ex.states.Validate(),
		ex.intents.Validate(),
	)
}

func (ex *Extension) setEngine(engine appdef.ExtensionEngineKind) {
	if (engine == appdef.ExtensionEngineKind_null) || (engine >= appdef.ExtensionEngineKind_count) {
		panic(appdef.ErrOutOfBounds("%v extension engine kind «%v»", ex, engine))
	}
	ex.engine = engine
}

func (ex *Extension) setName(name string) {
	if name == "" {
		panic(appdef.ErrMissed("%v extension name", ex))
	}
	if ok, err := appdef.ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", ex, name, err))
	}
	ex.name = name
}

// # Supports:
//   - appdef.IExtensionBuilder
type ExtensionBuilder struct {
	types.TypeBuilder
	*Extension
	states  *StoragesBuilder
	intents *StoragesBuilder
}

func MakeExtensionBuilder(extension *Extension) ExtensionBuilder {
	return ExtensionBuilder{
		TypeBuilder: types.MakeTypeBuilder(&extension.Typ),
		Extension:   extension,
		states:      NewStoragesBuilder(extension.states),
		intents:     NewStoragesBuilder(extension.intents),
	}
}

func (exb *ExtensionBuilder) Intents() appdef.IStoragesBuilder { return exb.intents }

func (exb *ExtensionBuilder) SetEngine(engine appdef.ExtensionEngineKind) appdef.IExtensionBuilder {
	exb.Extension.setEngine(engine)
	return exb
}

func (exb *ExtensionBuilder) SetName(name string) appdef.IExtensionBuilder {
	exb.Extension.setName(name)
	return exb
}

func (exb *ExtensionBuilder) States() appdef.IStoragesBuilder { return exb.states }

func (exb ExtensionBuilder) String() string { return exb.Extension.String() }
