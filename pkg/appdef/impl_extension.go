/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// # Implements:
//   - IExtension
type extension struct {
	typ
	name    string
	engine  ExtensionEngineKind
	states  *storages
	intents *storages
}

func makeExtension(app *appDef, ws *workspace, name QName, kind TypeKind) extension {
	e := extension{
		typ:     makeType(app, ws, name, kind),
		name:    name.Entity(),
		engine:  ExtensionEngineKind_BuiltIn,
		states:  newStorages(app),
		intents: newStorages(app),
	}

	return e
}

func (ex extension) Intents() IStorages { return ex.intents }

func (ex extension) Name() string { return ex.name }

func (ex extension) Engine() ExtensionEngineKind { return ex.engine }

func (ex extension) States() IStorages { return ex.states }

func (ex extension) String() string {
	// BuiltIn-function «test.func»
	return fmt.Sprintf("%s-%v", ex.Engine().TrimString(), ex.typ.String())
}

// Validates extension
//
// # Returns error:
//   - if storages (states or intents) contains unknown qname(s)
func (ex extension) Validate() error {
	return errors.Join(
		ex.states.validate(),
		ex.intents.validate(),
	)
}

func (ex *extension) setEngine(engine ExtensionEngineKind) {
	if (engine == ExtensionEngineKind_null) || (engine >= ExtensionEngineKind_count) {
		panic(ErrOutOfBounds("%v extension engine kind «%v»", ex, engine))
	}
	ex.engine = engine
}

func (ex *extension) setName(name string) {
	if name == "" {
		panic(ErrMissed("%v extension name", ex))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", ex, name, err))
	}
	ex.name = name
}

// # Implements:
//   - IExtensionBuilder
type extensionBuilder struct {
	typeBuilder
	*extension
	states  *storagesBuilder
	intents *storagesBuilder
}

func makeExtensionBuilder(extension *extension) extensionBuilder {
	return extensionBuilder{
		typeBuilder: makeTypeBuilder(&extension.typ),
		extension:   extension,
		states:      newStoragesBuilder(extension.states),
		intents:     newStoragesBuilder(extension.intents),
	}
}

func (exb *extensionBuilder) Intents() IStoragesBuilder { return exb.intents }

func (exb *extensionBuilder) SetEngine(engine ExtensionEngineKind) IExtensionBuilder {
	exb.extension.setEngine(engine)
	return exb
}

func (exb *extensionBuilder) SetName(name string) IExtensionBuilder {
	exb.extension.setName(name)
	return exb
}

func (exb *extensionBuilder) States() IStoragesBuilder { return exb.states }

func (exb extensionBuilder) String() string { return exb.extension.String() }

func (k ExtensionEngineKind) MarshalText() ([]byte, error) {
	var s string
	if k < ExtensionEngineKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an ExtensionEngineKind in human-readable form, without "ExtensionEngineKind_" prefix,
// suitable for debugging or error messages
func (k ExtensionEngineKind) TrimString() string {
	const pref = "ExtensionEngineKind_"
	return strings.TrimPrefix(k.String(), pref)
}
