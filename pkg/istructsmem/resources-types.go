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

// reader
type IStatelessPkg interface {
	istructs.IResources
	PkgPath() string
	SyncProjectors(func(p istructs.Projector))
	AsyncProjectors(func(p istructs.Projector))
}

// passed to e.g. sys package
type IStatelessPkgBuilder interface {
	AddPackage(pkgPath string) IStatelessPkgResourcesBuilder
	Build() map[string]IStatelessPkg // pkgPath
}

// filled with e.g. sys resources
type IStatelessPkgResourcesBuilder interface {
	AddFunc(istructs.IResource)
	AddSyncProjectors(...istructs.Projector)
	AddAsyncProjectors(...istructs.Projector)
}

type statelessPkgs map[string]IStatelessPkg

func (sp statelessPkgs) AddPackage(pkgPath string) IStatelessPkgResourcesBuilder {
	if _, ok := sp[pkgPath]; ok {
		panic("package " + pkgPath + " is added already")
	}
	res := &statelessPkg{
		resources: NewResources(),
		pkgPath:   pkgPath,
	}
	sp[pkgPath] = res
	return res
}

func (sp statelessPkgs) Build() map[string]IStatelessPkg {
	return sp
}

func NewStatelessPkgBuilder() IStatelessPkgBuilder {
	return statelessPkgs{}
}

// IStatelessPkg, IStatelessPkgResourcesBuilder
type statelessPkg struct {
	resources       Resources
	pkgPath         string
	syncProjectors  []istructs.Projector
	asyncProjectors []istructs.Projector
}

func (sr *statelessPkg) PkgPath() string {
	return sr.pkgPath
}

func (sr *statelessPkg) SyncProjectors(cb func(p istructs.Projector)) {
	for _, sp := range sr.syncProjectors {
		cb(sp)
	}
}

func (sr *statelessPkg) AsyncProjectors(cb func(p istructs.Projector)) {
	for _, sp := range sr.asyncProjectors {
		cb(sp)
	}
}

func (sr *statelessPkg) QueryResource(name appdef.QName) istructs.IResource {
	return sr.resources.QueryResource(name)
}

func (sr *statelessPkg) Resources(enum func(appdef.QName)) {
	sr.resources.Resources(enum)
}

func (sr *statelessPkg) AddSyncProjectors(syncProjectors ...istructs.Projector) {
	sr.syncProjectors = append(sr.syncProjectors, syncProjectors...)
}

func (sr *statelessPkg) AddAsyncProjectors(asyncProjectors ...istructs.Projector) {
	sr.asyncProjectors = append(sr.asyncProjectors, asyncProjectors...)
}

func (sr statelessPkg) AddFunc(res istructs.IResource) {
	sr.resources.Add(res)
}

// type StatelessResources struct {
// 	resources       Resources
// 	pkgPath         string
// 	SyncProjectors  []istructs.Projector
// 	AsyncProjectors []istructs.Projector
// }

// Implements istructs.IResources
type Resources map[appdef.QName]istructs.IResource

func NewResources() Resources {
	return Resources{}
}

// Adds new resource to application resources
func (res Resources) Add(r istructs.IResource) {
	res[r.QName()] = r
}

// Finds application resource by QName
func (res Resources) QueryResource(name appdef.QName) istructs.IResource {
	r, ok := res[name]
	if !ok {
		return nullResource
	}
	return r
}

// Enumerates all application resources
func (res Resources) Resources(enum func(appdef.QName)) {
	for n := range res {
		enum(n)
	}
}

// Ancestor for command & query functions
type abstractFunction struct {
	name appdef.QName
	res  func(istructs.PrepareArgs) appdef.QName
}

// istructs.IResource
func (af *abstractFunction) QName() appdef.QName { return af.name }

// istructs.IFunction
func (af *abstractFunction) ResultType(args istructs.PrepareArgs) appdef.QName {
	if af.res == nil {
		panic("ResultType() must not be called if created by not NewQueryFunctionCustomResult()")
	}
	return af.res(args)
}

// For debug and logging purposes
func (af *abstractFunction) String() string {
	return fmt.Sprintf("%v", af.QName())
}

type (
	// Function type to call for query execute action
	ExecQueryClosure func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error)

	// Implements istructs.IQueryFunction
	queryFunction struct {
		abstractFunction
		exec ExecQueryClosure
	}
)

// Creates and returns new query function
func NewQueryFunction(name appdef.QName, exec ExecQueryClosure) istructs.IQueryFunction {
	return NewQueryFunctionCustomResult(name, nil, exec)
}

func NewQueryFunctionCustomResult(name appdef.QName, resultFunc func(istructs.PrepareArgs) appdef.QName, exec ExecQueryClosure) istructs.IQueryFunction {
	return &queryFunction{
		abstractFunction: abstractFunction{
			name: name,
			res:  resultFunc,
		},
		exec: exec,
	}
}

// Null execute action closure for query functions
func NullQueryExec(_ context.Context, _ istructs.ExecQueryArgs, _ istructs.ExecQueryCallback) error {
	return nil
}

// istructs.IQueryFunction
func (qf *queryFunction) Exec(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	return qf.exec(ctx, args, callback)
}

// istructs.IResource
func (qf *queryFunction) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_QueryFunction
}

// istructs.IQueryFunction
func (qf *queryFunction) ResultType(args istructs.PrepareArgs) appdef.QName {
	return qf.abstractFunction.ResultType(args)
}

// for debug and logging purposes
func (qf *queryFunction) String() string {
	return fmt.Sprintf("q:%v", qf.abstractFunction.String())
}

type (
	// Function type to call for command execute action
	ExecCommandClosure func(args istructs.ExecCommandArgs) (err error)

	// Implements istructs.ICommandFunction
	commandFunction struct {
		abstractFunction
		exec ExecCommandClosure
	}
)

// NewCommandFunction creates and returns new command function
func NewCommandFunction(name appdef.QName, exec ExecCommandClosure) istructs.ICommandFunction {
	return &commandFunction{
		abstractFunction: abstractFunction{
			name: name,
		},
		exec: exec,
	}
}

// NullCommandExec is null execute action closure for command functions
func NullCommandExec(_ istructs.ExecCommandArgs) error {
	return nil
}

// istructs.ICommandFunction
func (cf *commandFunction) Exec(args istructs.ExecCommandArgs) error {
	return cf.exec(args)
}

// istructs.IResource
func (cf *commandFunction) Kind() istructs.ResourceKindType {
	return istructs.ResourceKind_CommandFunction
}

// for debug and logging purposes
func (cf *commandFunction) String() string {
	return fmt.Sprintf("c:%v", cf.abstractFunction.String())
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
