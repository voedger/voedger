/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newExtensions() *Extensions {
	return &Extensions{
		Commands:   make(map[appdef.QName]*CommandFunction),
		Queries:    make(map[appdef.QName]*QueryFunction),
		Projectors: make(map[appdef.QName]*Projector),
		Jobs:       make(map[appdef.QName]*Job),
	}
}

func (ff *Extensions) read(ext appdef.IExtension) {
	switch e := ext.(type) {
	case appdef.ICommand:
		c := &CommandFunction{}
		c.read(e)
		ff.Commands[c.QName] = c
	case appdef.IQuery:
		q := &QueryFunction{}
		q.read(e)
		ff.Queries[q.QName] = q
	case appdef.IProjector:
		p := &Projector{}
		p.read(e)
		ff.Projectors[p.QName] = p
	case appdef.IJob:
		j := &Job{}
		j.read(e)
		ff.Jobs[j.QName] = j
	}
}

func (e *Extension) read(ex appdef.IExtension) {
	e.Type.read(ex)
	e.Name = ex.Name()
	e.Engine = ex.Engine().TrimString()
	e.States = e.readStorages(ex.States())
	e.Intents = e.readStorages(ex.Intents())
}

func (e Extension) readStorages(storages appdef.IStorages) map[appdef.QName]appdef.QNames {
	s := make(map[appdef.QName]appdef.QNames)
	for _, n := range storages.Names() {
		st := storages.Storage(n)
		s[n] = appdef.QNamesFrom(st.Names()...)
	}
	return s
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

func (p *Projector) read(prj appdef.IProjector) {
	p.Extension.read(prj)
	for _, ev := range prj.Events() {
		e := ProjectorEvent{}
		e.read(ev)
		p.Events = append(p.Events, e)
	}
	p.WantErrors = prj.WantErrors()
}

func (e *ProjectorEvent) read(ev appdef.IProjectorEvent) {
	for _, o := range ev.Ops() {
		e.Ops = append(e.Ops, o.TrimString())
	}
	e.Filter.read(ev.Filter())
	e.Comment = ev.Comment()
}

func (j *Job) read(job appdef.IJob) {
	j.Extension.read(job)
	j.CronSchedule = job.CronSchedule()
}
