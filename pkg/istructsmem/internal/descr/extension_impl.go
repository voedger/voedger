/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func newExtensions() *Extensions {
	return &Extensions{
		Commands: make(map[appdef.QName]*CommandFunction),
		Queries:  make(map[appdef.QName]*QueryFunction),
	}
}

func (ff *Extensions) read(f appdef.IExtension) {
	if cmd, ok := f.(appdef.ICommand); ok {
		cf := &CommandFunction{}
		cf.read(cmd)
		ff.Commands[cf.QName] = cf
		return
	}
	if qry, ok := f.(appdef.IQuery); ok {
		qf := &QueryFunction{}
		qf.read(qry)
		ff.Queries[qf.QName] = qf
		return
	}
	if _, ok := f.(appdef.IProjector); ok {
		//TODO: implement projector
		return
	}

	//notest: This panic will only work when new appdef.IFunction interface descendants appear
	panic(fmt.Errorf("unknown func type %v", f))
}

func (e *Extension) read(ex appdef.IExtension) {
	e.Type.read(ex)
	e.Name = ex.Name()
	e.Engine = ex.Engine().TrimString()
}

func (f *Function) read(fn appdef.IFunction) {
	f.Extension.read(fn)
	if a := fn.Param(); a != nil {
		if n := a.QName(); n != appdef.NullQName {
			f.Arg = &n
		}
	}
	if r := fn.Result(); r != nil {
		if n := r.QName(); n != appdef.NullQName {
			f.Result = &n
		}
	}
}

func (f *CommandFunction) read(fn appdef.ICommand) {
	f.Function.read(fn)
	if a := fn.UnloggedParam(); a != nil {
		if n := a.QName(); n != appdef.NullQName {
			f.UnloggedArg = &n
		}
	}
}
