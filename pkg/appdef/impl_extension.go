/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strconv"
	"strings"
)

// # Implements:
//   - IExtension
type extension struct {
	typ
	name   string
	engine ExtensionEngineKind
}

func makeExtension(app *appDef, name QName, kind TypeKind) extension {
	e := extension{
		typ:    makeType(app, name, kind),
		name:   name.Entity(),
		engine: ExtensionEngineKind_BuiltIn,
	}

	return e
}

func (ex extension) Name() string {
	return ex.name
}

func (ex extension) Engine() ExtensionEngineKind {
	return ex.engine
}

func (ex extension) String() string {
	// BuiltIn-function «test.func»
	return fmt.Sprintf("%s-%v", ex.Engine().TrimString(), ex.typ.String())
}

func (ex *extension) setEngine(engine ExtensionEngineKind) {
	if (engine == ExtensionEngineKind_null) || (engine >= ExtensionEngineKind_Count) {
		panic(fmt.Errorf("%v: extension engine kind «%v» is invalid: %w", ex, engine, ErrInvalidExtensionEngineKind))
	}
	ex.engine = engine
}

func (ex *extension) setName(name string) {
	if name == "" {
		panic(fmt.Errorf("%v: extension name is empty: %w", ex, ErrNameMissed))
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
}

func makeExtensionBuilder(extension *extension) extensionBuilder {
	return extensionBuilder{
		typeBuilder: makeTypeBuilder(&extension.typ),
		extension:   extension,
	}
}

func (exb *extensionBuilder) SetEngine(engine ExtensionEngineKind) IExtensionBuilder {
	exb.extension.setEngine(engine)
	return exb
}

func (exb *extensionBuilder) SetName(name string) IExtensionBuilder {
	exb.extension.setName(name)
	return exb
}

func (k ExtensionEngineKind) MarshalText() ([]byte, error) {
	var s string
	if k < ExtensionEngineKind_Count {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an ExtensionEngineKind in human-readable form, without "ExtensionEngineKind_" prefix,
// suitable for debugging or error messages
func (k ExtensionEngineKind) TrimString() string {
	const pref = "ExtensionEngineKind_"
	return strings.TrimPrefix(k.String(), pref)
}
