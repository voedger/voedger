/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Implements istructs.IResources
type Resources struct {
	cfg       *AppConfigType
	resources map[appdef.QName]istructs.IResource
}

func newResources(cfg *AppConfigType) Resources {
	return Resources{cfg, make(map[appdef.QName]istructs.IResource)}
}

// Adds new resource to application resources
func (res *Resources) Add(r istructs.IResource) *Resources {
	res.resources[r.QName()] = r
	return res
}

// Finds application resource by QName
func (res *Resources) QueryResource(resource appdef.QName) (r istructs.IResource) {
	r, ok := res.resources[resource]
	if !ok {
		return nullResource
	}
	return r
}

// Returns argument object builder for query function
func (res *Resources) QueryFunctionArgsBuilder(query istructs.IQueryFunction) istructs.IObjectBuilder {
	r := newObject(res.cfg, query.ParamsDef())
	return &r
}

// Returns command function from application resource by QName or nil if not founded
func (res *Resources) CommandFunction(name appdef.QName) (cmd istructs.ICommandFunction) {
	r := res.QueryResource(name)
	if r.Kind() == istructs.ResourceKind_CommandFunction {
		cmd := r.(istructs.ICommandFunction)
		return cmd
	}
	return nil
}

// Enumerates all application resources
func (res *Resources) Resources(enum func(appdef.QName)) {
	for n := range res.resources {
		enum(n)
	}
}

// Ancestor for command & query functions
type abstractFunction struct {
	name, parsDef appdef.QName
	resDef        func(istructs.PrepareArgs) appdef.QName
}

// istructs.IResource
func (af *abstractFunction) QName() appdef.QName { return af.name }

// istructs.IFunction
func (af *abstractFunction) ParamsDef() appdef.QName { return af.parsDef }

// istructs.IFunction
func (af *abstractFunction) ResultDef(args istructs.PrepareArgs) appdef.QName {
	return af.resDef(args)
}

// For debug and logging purposes
func (af *abstractFunction) String() string {
	return fmt.Sprintf("%v", af.QName())
}

type (
	// Function type to call for query execute action
	ExecQueryClosure func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error)

	// Implements istructs.IQueryFunction
	queryFunction struct {
		abstractFunction
		exec ExecQueryClosure
	}
)

// Creates and returns new query function
func NewQueryFunction(name, pars, result appdef.QName, exec ExecQueryClosure) istructs.IQueryFunction {
	return NewQueryFunctionCustomResult(name, pars, func(istructs.PrepareArgs) appdef.QName { return result }, exec)
}

func NewQueryFunctionCustomResult(name, pars appdef.QName, resultDef func(istructs.PrepareArgs) appdef.QName, exec ExecQueryClosure) istructs.IQueryFunction {
	return &queryFunction{
		abstractFunction: abstractFunction{
			name:    name,
			parsDef: pars,
			resDef:  resultDef,
		},
		exec: exec,
	}
}

// Null execute action closure for query functions
func NullQueryExec(_ context.Context, _ istructs.IQueryFunction, _ istructs.ExecQueryArgs, _ istructs.ExecQueryCallback) error {
	return nil
}

// istructs.IQueryFunction
func (qf *queryFunction) Exec(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	return qf.exec(ctx, qf, args, callback)
}

// istructs.IResource
func (qf *queryFunction) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_QueryFunction
}

// istructs.IQueryFunction
func (qf *queryFunction) ResultDef(args istructs.PrepareArgs) appdef.QName {
	return qf.abstractFunction.ResultDef(args)
}

// for debug and logging purposes
func (qf *queryFunction) String() string {
	return fmt.Sprintf("q:%v", qf.abstractFunction.String())
}

type (
	// Function type to call for command execute action
	ExecCommandClosure func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error)

	// Implements istructs.ICommandFunction
	commandFunction struct {
		abstractFunction
		unlParsDef appdef.QName
		exec       ExecCommandClosure
	}
)

// NewCommandFunction creates and returns new command function
func NewCommandFunction(name, params, unlogged, result appdef.QName, exec ExecCommandClosure) istructs.ICommandFunction {
	return &commandFunction{
		abstractFunction: abstractFunction{
			name:    name,
			parsDef: params,
			resDef:  func(pa istructs.PrepareArgs) appdef.QName { return result },
		},
		unlParsDef: unlogged,
		exec:       exec,
	}
}

// NullCommandExec is null execute action closure for command functions
func NullCommandExec(_ istructs.ICommandFunction, _ istructs.ExecCommandArgs) error {
	return nil
}

// istructs.ICommandFunction
func (cf *commandFunction) Exec(args istructs.ExecCommandArgs) error {
	return cf.exec(cf, args)
}

// istructs.IResource
func (cf *commandFunction) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_CommandFunction
}

// istructs.ICommandFunction
func (cf *commandFunction) ResultDef() appdef.QName {
	return cf.abstractFunction.ResultDef(nullPrepareArgs)
}

// for debug and logging purposes
func (cf *commandFunction) String() string {
	return fmt.Sprintf("c:%v", cf.abstractFunction.String())
}

func (cf *commandFunction) UnloggedParamsDef() appdef.QName {
	return cf.unlParsDef
}

// nullResourceType type to return then resource is not founded
//   - interfaces:
//     — IResource
type nullResourceType struct {
}

func newNullResource() *nullResourceType {
	return &nullResourceType{}
}

// IResource members
func (r *nullResourceType) Kind() istructs.ResourceKindType { return istructs.ResourceKind_null }
func (r *nullResourceType) QName() appdef.QName             { return appdef.NullQName }
