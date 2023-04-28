/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func newResource() *Resource {
	return &Resource{}
}

func (r *Resource) read(resource istructs.IResource) {
	r.Kind = resource.Kind()
	r.Name = resource.QName()
	r.Command = nil
	r.Query = nil
	switch r.Kind {
	case istructs.ResourceKind_CommandFunction:
		if cmd, ok := resource.(istructs.ICommandFunction); ok {
			r.readCmd(cmd)
		}
	case istructs.ResourceKind_QueryFunction:
		if q, ok := resource.(istructs.IQueryFunction); ok {
			r.readQuery(q)
		}
	}
}

func (r *Resource) readCmd(command istructs.ICommandFunction) {
	r.Command = new(Command)

	if n := command.ParamsDef(); n != appdef.NullQName {
		r.Command.Params = &n
	}

	if n := command.UnloggedParamsDef(); n != appdef.NullQName {
		r.Command.Unlogged = &n
	}
	if n := command.ResultDef(); n != appdef.NullQName {
		r.Command.Result = &n
	}
}

func (r *Resource) readQuery(query istructs.IQueryFunction) {
	r.Query = new(Query)

	if n := query.ParamsDef(); n != appdef.NullQName {
		r.Query.Params = &n
	}
	if n := query.ResultDef(istructs.PrepareArgs{}); n != appdef.NullQName {
		r.Query.Result = &n
	}
}
