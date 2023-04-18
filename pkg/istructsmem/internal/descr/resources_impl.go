/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/istructs"

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

	if n := command.ParamsSchema(); n != istructs.NullQName {
		r.Command.ParamsSchema = &n
	}

	if n := command.UnloggedParamsSchema(); n != istructs.NullQName {
		r.Command.UnloggedParamsSchema = &n
	}
	if n := command.ResultSchema(); n != istructs.NullQName {
		r.Command.ResultSchema = &n
	}
}

func (r *Resource) readQuery(query istructs.IQueryFunction) {
	r.Query = new(Query)

	if n := query.ParamsSchema(); n != istructs.NullQName {
		r.Query.ParamsSchema = &n
	}
	if n := query.ResultSchema(istructs.PrepareArgs{}); n != istructs.NullQName {
		r.Query.ResultSchema = &n
	}
}
