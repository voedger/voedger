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
)

type (
	// actualizers is a set of actualizers for application partitions.
	//
	// # Implements:
	//	- IActualizersService:
	//	   + appparts.IActualizers
	//	   + pipeline.IService
	actualizers struct {
		mx   sync.RWMutex
		cfg  BasicAsyncActualizerConfig
		apps map[appdef.AppQName]*appActs
	}

	appActs struct {
		mx    sync.RWMutex
		parts map[istructs.PartitionID]*partActs
	}

	partActs struct {
		cfg AsyncActualizerConf
		mx  sync.RWMutex
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
		mx:   sync.RWMutex{},
		cfg:  cfg,
		apps: make(map[appdef.AppQName]*appActs),
	}
}

func (*actualizers) Prepare(interface{}) error { return nil }

func (a *actualizers) Run(ctx context.Context) {
	// store vvm context for deploy new partitions (or redeploy existing)
	a.cfg.Ctx = ctx
}

func (a *actualizers) Stop() {
	// Cancellation has already been sent to the context by caller.
	// Here we are just waiting while all async actualizers are stopped

	wp := make([]*partActs, 0) // who works?
	a.mx.RLock()
	for _, app := range a.apps {
		app.mx.RLock()
		for _, part := range app.parts {
			part.mx.RLock()
			if len(part.rt) > 0 {
				wp = append(wp, part)
			}
			part.mx.RUnlock()
		}
		app.mx.RUnlock()
	}
	a.mx.RUnlock()
	if len(wp) == 0 {
		return // all done
	}

	// wait for worked partitions
	wg := sync.WaitGroup{}
	for _, part := range wp {
		wg.Add(1)
		go func(part *partActs) {
			part.wg.Wait()
			wg.Done()
		}(part)
	}
	wg.Wait()
}

func (a *actualizers) DeployPartition(n appdef.AppQName, id istructs.PartitionID) error {
	def, err := a.cfg.AppPartitions.AppDef(n)
	if err != nil {
		return err
	}

	a.mx.RLock()
	app, ok := a.apps[n]
	a.mx.RUnlock()

	if !ok {
		a.mx.Lock()
		if accuracy, ok := a.apps[n]; ok {
			app = accuracy // notest
		} else {
			app = &appActs{
				mx:    sync.RWMutex{},
				parts: make(map[istructs.PartitionID]*partActs),
			}
			a.apps[n] = app
		}
		a.mx.Unlock()
	}

	app.mx.RLock()
	part, ok := app.parts[id]
	app.mx.RUnlock()

	if !ok {
		app.mx.Lock()
		if accuracy, ok := app.parts[id]; ok {
			part = accuracy // notest
		} else {
			part = &partActs{
				cfg: AsyncActualizerConf{
					BasicAsyncActualizerConfig: a.cfg,
					AppQName:                   n,
					Partition:                  id,
				},
				mx: sync.RWMutex{},
				wg: sync.WaitGroup{},
				rt: make(map[appdef.QName]*runtimeAct),
			}
			app.parts[id] = part
		}
		app.mx.Unlock()
	}

	// stop async actualizers for removed projectors
	part.mx.RLock()
	for name := range part.rt {
		// TODO: Cover the tests after IAppPartitions will have the possibility to redeploy application with new AppDef
		// notest
		if prj := def.Projector(name); (prj == nil) || prj.Sync() {
			part.stop(name)
		}
	}
	part.mx.RUnlock()

	// start new async actualizers
	part.mx.Lock()
	def.Projectors(
		func(proj appdef.IProjector) {
			if !proj.Sync() { // only async projectors should be started here
				prj := proj.QName()
				if !part.exists(prj) {
					part.start(prj)
				}
			}
		})
	part.mx.Unlock()

	return nil
}

func (a *actualizers) UndeployPartition(n appdef.AppQName, id istructs.PartitionID) {
	a.mx.RLock()
	app, ok := a.apps[n]
	a.mx.RUnlock()
	if !ok {
		return
	}

	app.mx.RLock()
	part, ok := app.parts[id]
	app.mx.RUnlock()
	if !ok {
		return
	}

	part.mx.RLock()
	for prj := range part.rt {
		part.stop(prj)
	}
	part.mx.RUnlock()

	part.wg.Wait()

	app.mx.Lock()
	delete(app.parts, id)
	app.mx.Unlock()
}

func (a *actualizers) SetAppPartitions(appParts appparts.IAppPartitions) {
	if (a.cfg.AppPartitions != nil) && (a.cfg.AppPartitions != appParts) {
		panic(fmt.Errorf("unable to reset application partitions: %w", errors.ErrUnsupported))
	}
	a.cfg.AppPartitions = appParts
}

// internal metrics for testing
type actualizersMetrics struct {
	apps        []appdef.AppQName
	parts       map[appdef.AppQName][]istructs.PartitionID
	actualizers map[appdef.AppQName]map[istructs.PartitionID][]appdef.QName
}

func (a *actualizers) metrics() actualizersMetrics {
	a.mx.RLock()
	s := actualizersMetrics{
		apps:        make([]appdef.AppQName, 0, len(a.apps)),
		parts:       make(map[appdef.AppQName][]istructs.PartitionID, len(a.apps)),
		actualizers: make(map[appdef.AppQName]map[istructs.PartitionID][]appdef.QName, len(a.apps)),
	}
	for n, app := range a.apps {
		s.apps = append(s.apps, n)
		app.mx.RLock()
		s.parts[n] = make([]istructs.PartitionID, 0, len(app.parts))
		s.actualizers[n] = make(map[istructs.PartitionID][]appdef.QName, len(app.parts))
		for id, part := range app.parts {
			s.parts[n] = append(s.parts[n], id)
			part.mx.RLock()
			pp := make([]appdef.QName, 0, len(part.rt))
			for prj := range part.rt {
				pp = append(pp, prj)
			}
			part.mx.RUnlock()
			s.actualizers[n][id] = pp
		}
		app.mx.RUnlock()
	}
	a.mx.RUnlock()

	return s
}

// p.mx should be locked for read by caller
func (p *partActs) exists(n appdef.QName) bool {
	_, ok := p.rt[n]
	return ok
}

// p.mx should be locked for write by caller
func (p *partActs) start(n appdef.QName) {
	rt := &runtimeAct{
		actualizer: &asyncActualizer{
			projector: n,
			conf:      p.cfg,
		}}

	rt.actualizer.Prepare()

	ctx, cancel := context.WithCancel(p.cfg.Ctx)
	rt.cancel = cancel

	p.rt[n] = rt

	p.wg.Add(1)
	go func() {
		rt.actualizer.Run(ctx)

		p.mx.Lock()
		delete(p.rt, n)
		p.mx.Unlock()

		p.wg.Done()
	}()
}

// p.mx should be read locked by caller
func (p *partActs) stop(n appdef.QName) {
	if rt, ok := p.rt[n]; ok {
		rt.cancel()
	}
}
