/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"fmt"
	"maps"

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
	if cmd, ok := ext.(appdef.ICommand); ok {
		c := &CommandFunction{}
		c.read(cmd)
		ff.Commands[c.QName] = c
		return
	}
	if qry, ok := ext.(appdef.IQuery); ok {
		q := &QueryFunction{}
		q.read(qry)
		ff.Queries[q.QName] = q
		return
	}
	if prj, ok := ext.(appdef.IProjector); ok {
		p := &Projector{}
		p.read(prj)
		ff.Projectors[p.QName] = p
		return
	}
	if job, ok := ext.(appdef.IJob); ok {
		j := &Job{}
		j.read(job)
		ff.Jobs[j.QName] = j
		return
	}

	//notest: This panic will only work when new appdef.IFunction interface descendants appear
	panic(fmt.Errorf("unknown func type %v", ext))
}

func (e *Extension) read(ex appdef.IExtension) {
	e.Type.read(ex)
	e.Name = ex.Name()
	e.Engine = ex.Engine().TrimString()
	e.States = maps.Clone(ex.States().Map())
	e.Intents = maps.Clone(ex.Intents().Map())
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
	if prj.Events().Len() > 0 {
		p.Events = make(map[appdef.QName]ProjectorEvent)
		prj.Events().Enum(func(ev appdef.IProjectorEvent) {
			e := ProjectorEvent{}
			e.read(ev)
			p.Events[e.On] = e
		})
	}
	p.WantErrors = prj.WantErrors()
}

func (e *ProjectorEvent) read(ev appdef.IProjectorEvent) {
	e.Comment = ev.Comment()
	e.On = ev.On().QName()
	for _, k := range ev.Kind() {
		e.Kind = append(e.Kind, k.TrimString())
	}
}

func (j *Job) read(job appdef.IJob) {
	j.Extension.read(job)
	j.CronSchedule = job.CronSchedule()
}
