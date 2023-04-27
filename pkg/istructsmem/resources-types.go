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

// ResourcesType is type for application resources
//   - interfaces:
//     — istructs.IResources
type ResourcesType struct {
	cfg       *AppConfigType
	resources map[appdef.QName]istructs.IResource
}

func newResources(cfg *AppConfigType) ResourcesType {
	return ResourcesType{cfg, make(map[appdef.QName]istructs.IResource)}
}

// Add adds new resource to application resources
func (res *ResourcesType) Add(r istructs.IResource) *ResourcesType {
	res.resources[r.QName()] = r
	return res
}

// QueryResource finds application resources by QName
func (res *ResourcesType) QueryResource(resource appdef.QName) (r istructs.IResource) {
	r, ok := res.resources[resource]
	if !ok {
		return nullResource
	}
	return r
}

// QueryFunctionArgsBuilder returns argument object builder for query function
func (res *ResourcesType) QueryFunctionArgsBuilder(query istructs.IQueryFunction) istructs.IObjectBuilder {
	r := newObject(res.cfg, query.ParamsSchema())
	return &r
}

// CommandFunction returns command function from application resource by QName or nil if not founded
func (res *ResourcesType) CommandFunction(name appdef.QName) (cmd istructs.ICommandFunction) {
	r := res.QueryResource(name)
	if r.Kind() == istructs.ResourceKind_CommandFunction {
		cmd := r.(istructs.ICommandFunction)
		return cmd
	}
	return nil
}

// Resources enumerates all application resources
func (res *ResourcesType) Resources(enum func(appdef.QName)) {
	for n := range res.resources {
		enum(n)
	}
}

// abstractFunctionType is ancestor for CommandFunctionType and QueryFunctionType
type abstractFunctionType struct {
	name, paramsSchema appdef.QName
	resultSchemaFunc   func(istructs.PrepareArgs) appdef.QName
}

// istructs.IResource
func (af *abstractFunctionType) QName() appdef.QName { return af.name }

// istructs.IFunction
func (af *abstractFunctionType) ParamsSchema() appdef.QName { return af.paramsSchema }

// istructs.IFunction
func (af *abstractFunctionType) ResultSchema(args istructs.PrepareArgs) appdef.QName {
	return af.resultSchemaFunc(args)
}

// for debug and logging purposes
func (af *abstractFunctionType) String() string {
	return fmt.Sprintf("%v", af.QName())
}

type (
	// ExecQueryClosureType is function type to call for query execute action
	ExecQueryClosureType func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error)

	// queryFunctionType implements istructs.IQueryFunction
	queryFunctionType struct {
		abstractFunctionType
		exec ExecQueryClosureType
	}
)

// NewQueryFunction creates and returns new query function
func NewQueryFunction(name, paramsSchema, resultSchema appdef.QName, exec ExecQueryClosureType) istructs.IQueryFunction {
	return NewQueryFunctionCustomResult(name, paramsSchema, func(pa istructs.PrepareArgs) appdef.QName { return resultSchema }, exec)
}

func NewQueryFunctionCustomResult(name, paramsSchema appdef.QName, resultSchemaFunc func(istructs.PrepareArgs) appdef.QName, exec ExecQueryClosureType) istructs.IQueryFunction {
	return &queryFunctionType{
		abstractFunctionType: abstractFunctionType{
			name:             name,
			paramsSchema:     paramsSchema,
			resultSchemaFunc: resultSchemaFunc,
		},
		exec: exec,
	}
}

// NullQueryExec is null execute action closure for query functions
func NullQueryExec(_ context.Context, _ istructs.IQueryFunction, _ istructs.ExecQueryArgs, _ istructs.ExecQueryCallback) error {
	return nil
}

// istructs.IQueryFunction
func (qf *queryFunctionType) Exec(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	return qf.exec(ctx, qf, args, callback)
}

// istructs.IResource
func (qf *queryFunctionType) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_QueryFunction
}

// istructs.IQueryFunction
func (qf *queryFunctionType) ResultSchema(args istructs.PrepareArgs) appdef.QName {
	return qf.abstractFunctionType.ResultSchema(args)
}

// for debug and logging purposes
func (qf *queryFunctionType) String() string {
	return fmt.Sprintf("q:%v", qf.abstractFunctionType.String())
}

type (
	// ExecCommandClosureType is function type to call for command execute action
	ExecCommandClosureType func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error)

	// commandFunctionType implements istructs.ICommandFunction
	commandFunctionType struct {
		abstractFunctionType
		unloggedParamsSchema appdef.QName
		exec                 ExecCommandClosureType
	}
)

// NewCommandFunction creates and returns new command function
func NewCommandFunction(name, paramsSchema, unloggedParamsSchema, resultSchema appdef.QName, exec ExecCommandClosureType) istructs.ICommandFunction {
	return &commandFunctionType{
		abstractFunctionType: abstractFunctionType{
			name:             name,
			paramsSchema:     paramsSchema,
			resultSchemaFunc: func(pa istructs.PrepareArgs) appdef.QName { return resultSchema },
		},
		unloggedParamsSchema: unloggedParamsSchema,
		exec:                 exec,
	}
}

// NullCommandExec is null execute action closure for command functions
func NullCommandExec(_ istructs.ICommandFunction, _ istructs.ExecCommandArgs) error {
	return nil
}

// istructs.ICommandFunction
func (cf *commandFunctionType) Exec(args istructs.ExecCommandArgs) error {
	return cf.exec(cf, args)
}

// istructs.IResource
func (cf *commandFunctionType) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_CommandFunction
}

// istructs.ICommandFunction
func (cf *commandFunctionType) ResultSchema() appdef.QName {
	return cf.abstractFunctionType.ResultSchema(nullPrepareArgs)
}

// for debug and logging purposes
func (cf *commandFunctionType) String() string {
	return fmt.Sprintf("c:%v", cf.abstractFunctionType.String())
}

func (cf *commandFunctionType) UnloggedParamsSchema() appdef.QName {
	return cf.unloggedParamsSchema
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
