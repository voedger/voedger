/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// # Implements:
//   - IDef
type def struct {
	app  *appDef
	name QName
	kind DefKind
}

func makeDef(app *appDef, name QName, kind DefKind) def {
	if name == NullQName {
		panic(fmt.Errorf("definition name cannot be empty: %w", ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("invalid definition name «%v»: %w", name, err))
	}
	if app.DefByName(name) != nil {
		panic(fmt.Errorf("definition name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	return def{app, name, kind}
}

func (d *def) App() IAppDef {
	return d.app
}

func (d *def) Kind() DefKind {
	return d.kind
}

func (d *def) QName() QName {
	return d.name
}

// NullDef is used for return then definition	is not founded
var NullDef = new(nullDef)

type nullDef struct{}

func (d *nullDef) App() IAppDef  { return nil }
func (d *nullDef) Kind() DefKind { return DefKind_null }
func (d *nullDef) QName() QName  { return NullQName }
