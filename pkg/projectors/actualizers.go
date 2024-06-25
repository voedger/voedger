/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package projectors

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type (
	// actualizers is a set of actualizers for application partitions.
	//
	// # Implements:
	//   - IActualizers
	//   - appparts.IActualizers
	actualizers struct {
		cfg  BasicAsyncActualizerConfig
		apps map[appdef.AppQName]*appActs
	}

	appActs struct {
		parts map[istructs.PartitionID]*partActs
	}

	partActs struct {
		cfg AsyncActualizerConf
		wg  sync.WaitGroup
		rt  map[appdef.QName]*runtimeAct
	}

	runtimeAct struct {
		actualizer *asyncActualizer
		cancel     func()
	}
)

func newActualizers(cfg BasicAsyncActualizerConfig) *actualizers {
	return &actualizers{
		cfg:  cfg,
		apps: make(map[appdef.AppQName]*appActs),
	}
}

func (a *actualizers) Close() {
	a.close()
}

func (a *actualizers) AsServiceOperator() pipeline.ISyncOperator {
	return a
}

func (a *actualizers) DoSync(ctx context.Context, _ interface{}) error {
	if a.cfg.Ctx == nil {
		// store vvm context for deploy new partitions (or redeploy existing)
		a.cfg.Ctx = ctx
	}
	return nil
}

func (a *actualizers) DeployPartition(n appdef.AppQName, id istructs.PartitionID) error {
	def, err := a.cfg.AppPartitions.AppDef(n)
	if err != nil {
		return err
	}

	app, ok := a.apps[n]
	if !ok {
		app = &appActs{
			parts: make(map[istructs.PartitionID]*partActs),
		}
		a.apps[n] = app
	}

	part, ok := app.parts[id]
	if !ok {
		part = &partActs{
			cfg: AsyncActualizerConf{
				BasicAsyncActualizerConfig: a.cfg,
				AppQName:                   n,
				Partition:                  id,
			},
			rt: make(map[appdef.QName]*runtimeAct),
		}
		app.parts[id] = part
	}

	// stop eliminated actualizers
	for name, rt := range part.rt {
		if prj := def.Projector(name); (prj == nil) || prj.Sync() {
			if rt.cancel != nil {
				rt.cancel()
			}
			delete(part.rt, name)
		}
	}

	// start new async actualizers
	def.Projectors(
		func(proj appdef.IProjector) {
			if proj.Sync() {
				// only async projectors should be started here,
				// sync projectors are started by command processor sync pipeline
				return
			}

			name := proj.QName()
			if part.exists(name) {
				return
			}
			part.start(name)
		})

	return nil
}

func (a *actualizers) UndeployPartition(app appdef.AppQName, id istructs.PartitionID) {
	part, ok := a.apps[app].parts[id]
	if !ok {
		return // or panics?
	}
	part.close()

	delete(a.apps[app].parts, id)
	if len(a.apps[app].parts) == 0 {
		delete(a.apps, app)
	}
}

func (a *actualizers) SetAppPartitions(appParts appparts.IAppPartitions) {
	if (a.cfg.AppPartitions != nil) && (a.cfg.AppPartitions != appParts) {
		panic(fmt.Errorf("unable to reset application partitions: %w", errors.ErrUnsupported))
	}
	a.cfg.AppPartitions = appParts
}

func (a *actualizers) close() {
	wg := sync.WaitGroup{}
	for _, app := range a.apps {
		for _, part := range app.parts {
			wg.Add(1)
			go func(part *partActs) {
				part.close()
				wg.Done()
			}(part)
		}
	}
	wg.Wait()

	clear(a.apps)
}

func (p *partActs) close() {
	for n := range p.rt {
		p.stop(n)
	}
	p.wg.Wait()
}

func (p *partActs) exists(n appdef.QName) bool {
	_, ok := p.rt[n]
	return ok
}

func (p *partActs) start(n appdef.QName) error {
	rt := &runtimeAct{
		actualizer: &asyncActualizer{
			projector: n,
			conf:      p.cfg,
		}}

	// TODO: actualizer.Prepare never returns an error. Reduce complexity.
	if err := rt.actualizer.Prepare(nil); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(p.cfg.Ctx)
	rt.cancel = cancel

	p.rt[n] = rt

	p.wg.Add(1)
	go func() {
		rt.actualizer.Run(ctx)
		p.wg.Done()
	}()

	return nil
}

func (p *partActs) stop(n appdef.QName) {
	if rt, ok := p.rt[n]; ok {
		if rt.cancel != nil {
			rt.cancel()
		}
		delete(p.rt, n)
	}
}
