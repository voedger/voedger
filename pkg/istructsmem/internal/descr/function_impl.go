/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func newFunctions() *Functions {
	return &Functions{
		Commands: make(map[appdef.QName]*CommandFunction),
		Queries:  make(map[appdef.QName]*QueryFunction),
	}
}

func (ff *Functions) read(f appdef.IFunction) {
	if cmd, ok := f.(appdef.ICommand); ok {
		cf := &CommandFunction{}
		cf.read(cmd)
		ff.Commands[cf.Name] = cf
		return
	}
	if qry, ok := f.(appdef.IQuery); ok {
		qf := &QueryFunction{}
		qf.read(qry)
		ff.Queries[qf.Name] = qf
		return
	}

	//notest: This panic will only work when new appdef.IFunction interface descendants appear
	panic(fmt.Errorf("unknown func type %v", f))
}

func (f *Function) read(fn appdef.IFunction) {
	f.Comment = fn.Comment()
	f.Name = fn.QName()
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
	f.Extension = Extension{
		Name:   fn.Extension().Name(),
		Engine: fn.Extension().Engine(),
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
