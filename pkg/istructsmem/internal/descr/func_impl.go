/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func newFuncs() *Funcs {
	return &Funcs{
		Commands: make(map[appdef.QName]*CommandFunc),
		Queries:  make(map[appdef.QName]*QueryFunc),
	}
}

func (ff *Funcs) read(fn appdef.IFunc) {
	if cmd, ok := fn.(appdef.ICommand); ok {
		c := &CommandFunc{}
		c.read(cmd)
		ff.Commands[c.Name] = c
		return
	}
	if qry, ok := fn.(appdef.IQuery); ok {
		q := &QueryFunc{}
		q.read(qry)
		ff.Queries[q.Name] = q
		return
	}

	//notest: This panic will only work when new appdef.IFunc interface descendants appear
	panic(fmt.Errorf("unknown func type %v", fn))
}

func (f *Func) read(fn appdef.IFunc) {
	f.Comment = fn.Comment()
	f.Name = fn.QName()
	if a := fn.Arg(); a != nil {
		if n := a.QName(); n != appdef.NullQName {
			f.Arg = &n
		}
	}
	if r := fn.Arg(); r != nil {
		if n := r.QName(); n != appdef.NullQName {
			f.Result = &n
		}
	}
	f.Extension = Extension{
		Name:   fn.Extension().Name(),
		Engine: fn.Extension().Engine(),
	}
}

func (f *CommandFunc) read(fn appdef.ICommand) {
	f.Func.read(fn)
	if a := fn.UnloggedArg(); a != nil {
		if n := a.QName(); n != appdef.NullQName {
			f.UnloggedArg = &n
		}
	}
}
