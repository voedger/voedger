/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// Implements IAppDef and IAppDefBuilder interfaces
type appDef struct {
	changes int
	schemas map[QName]*schema
}

func newAppDef() *appDef {
	app := appDef{
		schemas: make(map[QName]*schema),
	}
	return &app
}

func (app *appDef) Add(name QName, kind DefKind) SchemaBuilder {
	if name == NullQName {
		panic(fmt.Errorf("schema name cannot be empty: %w", ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("invalid schema name «%v»: %w", name, err))
	}
	if app.SchemaByName(name) != nil {
		panic(fmt.Errorf("schema name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	schema := newSchema(app, name, kind)
	app.schemas[name] = schema
	app.changed()
	return schema
}

func (app *appDef) AddView(name QName) ViewBuilder {
	v := newViewBuilder(app, name)
	app.changed()
	return &v
}

func (app *appDef) Build() (result IAppDef, err error) {
	app.prepare()

	validator := newValidator()
	app.Schemas(func(schema Schema) {
		err = errors.Join(err, validator.validate(schema))
	})
	if err != nil {
		return nil, err
	}

	app.changes = 0
	return app, nil
}

func (app *appDef) HasChanges() bool {
	return app.changes > 0
}

func (app *appDef) Schema(name QName) Schema {
	if schema := app.SchemaByName(name); schema != nil {
		return schema
	}
	return NullSchema
}

func (app *appDef) SchemaByName(name QName) Schema {
	if schema, ok := app.schemas[name]; ok {
		return schema
	}
	return nil
}

func (app *appDef) SchemaCount() int {
	return len(app.schemas)
}

func (app *appDef) Schemas(enum func(Schema)) {
	for _, schema := range app.schemas {
		enum(schema)
	}
}

func (app *appDef) changed() {
	app.changes++
}

func (app *appDef) prepare() {
	app.Schemas(func(s Schema) {
		if s.Kind() == DefKind_ViewRecord {
			app.prepareViewFullKeySchema(s)
		}
	})
}
