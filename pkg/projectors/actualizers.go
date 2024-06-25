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
		apps sync.Map //[appdef.AppQName]*appActs
	}

	appActs struct {
		parts sync.Map //[istructs.PartitionID]*partActs
	}

	partActs struct {
		cfg AsyncActualizerConf
		wg  sync.WaitGroup
		rt  sync.Map // [appdef.QName]*runtimeAct
	}

	runtimeAct struct {
		actualizer *asyncActualizer
		cancel     func()
	}
)

func newActualizers(cfg BasicAsyncActualizerConfig) *actualizers {
	return &actualizers{
		cfg:  cfg,
		apps: sync.Map{},
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

	var (
		app  *appActs
		part *partActs
	)

	if v, ok := a.apps.Load(n); ok {
		app = v.(*appActs)
	} else {
		app = &appActs{
			parts: sync.Map{},
		}
		a.apps.Store(n, app)
	}

	if v, ok := app.parts.Load(id); ok {
		part = v.(*partActs)
	} else {
		part = &partActs{
			cfg: AsyncActualizerConf{
				BasicAsyncActualizerConfig: a.cfg,
				AppQName:                   n,
				Partition:                  id,
			},
			rt: sync.Map{},
		}
		app.parts.Store(id, part)
	}

	// stop eliminated actualizers
	part.rt.Range(
		func(key, _ any) bool {
			name := key.(appdef.QName)
			if prj := def.Projector(name); (prj == nil) || prj.Sync() {
				part.stop(name)
			}
			return true
		})

	// start new async actualizers
	def.Projectors(
		func(proj appdef.IProjector) {
			if !proj.Sync() { // only async projectors should be started here,
				name := proj.QName()
				if !part.exists(name) {
					part.start(name)
				}
			}
		})

	return nil
}

func (a *actualizers) UndeployPartition(n appdef.AppQName, id istructs.PartitionID) {
	var (
		app  *appActs
		part *partActs
	)
	if v, ok := a.apps.Load(n); ok {
		app = v.(*appActs)
		if v, ok := app.parts.Load(id); ok {
			part = v.(*partActs)
		}
	}

	if (app == nil) || (part == nil) {
		return // or panic?
	}

	part.close()

	app.parts.Delete(id)
}

func (a *actualizers) SetAppPartitions(appParts appparts.IAppPartitions) {
	if (a.cfg.AppPartitions != nil) && (a.cfg.AppPartitions != appParts) {
		panic(fmt.Errorf("unable to reset application partitions: %w", errors.ErrUnsupported))
	}
	a.cfg.AppPartitions = appParts
}

func (a *actualizers) close() {
	wg := sync.WaitGroup{}

	a.apps.Range(
		func(k, v any) bool {
			app := v.(*appActs)
			app.parts.Range(
				func(k, v any) bool {
					part := v.(*partActs)
					wg.Add(1)
					go func(part *partActs) {
						part.close()
						app.parts.Delete(k)
						wg.Done()
					}(part)
					return true
				})
			a.apps.Delete(k)
			return true
		})

	wg.Wait()
}

func (p *partActs) close() {
	p.rt.Range(
		func(key, _ any) bool {
			name := key.(appdef.QName)
			p.stop(name)
			return true
		})
	p.wg.Wait()
}

func (p *partActs) exists(n appdef.QName) bool {
	_, ok := p.rt.Load(n)
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

	p.rt.Store(n, rt)

	p.wg.Add(1)
	go func() {
		rt.actualizer.Run(ctx)
		p.rt.Delete(n)
		p.wg.Done()
	}()

	return nil
}

func (p *partActs) stop(n appdef.QName) {
	if rt, ok := p.rt.Load(n); ok {
		rt.(*runtimeAct).cancel()
	}
}
