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
		apps map[appdef.AppQName]*appActualizers
	}

	appActualizers struct {
		parts map[istructs.PartitionID]*partActualizers
	}

	partActualizers struct {
		cfg AsyncActualizerConf
		wg  sync.WaitGroup
		run map[appdef.QName]*struct {
			actualizer *asyncActualizer
			cancel     func()
		}
	}
)

func newActualizers(cfg BasicAsyncActualizerConfig) *actualizers {
	return &actualizers{
		cfg:  cfg,
		apps: make(map[appdef.AppQName]*appActualizers),
	}
}

func (a *actualizers) Close() {
	a.stop()
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
		app = &appActualizers{
			parts: make(map[istructs.PartitionID]*partActualizers),
		}
		a.apps[n] = app
	}

	part, ok := app.parts[id]
	if !ok {
		part = &partActualizers{
			cfg: AsyncActualizerConf{
				BasicAsyncActualizerConfig: a.cfg,
				AppQName:                   n,
				Partition:                  id,
			},
			run: make(map[appdef.QName]*struct {
				actualizer *asyncActualizer
				cancel     func()
			}),
		}
		app.parts[id] = part
	}

	// stop eliminated actualizers
	for name, run := range part.run {
		if prj := def.Projector(name); (prj == nil) || prj.Sync() {
			if run.cancel != nil {
				run.cancel()
			}
			delete(part.run, name)
		}
	}

	// start new actualizers
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

func (a *actualizers) UndeployPartition(appdef.AppQName, istructs.PartitionID) {}

func (a *actualizers) SetAppPartitions(appParts appparts.IAppPartitions) {
	if (a.cfg.AppPartitions != nil) && (a.cfg.AppPartitions != appParts) {
		panic(fmt.Errorf("unable to reset application partitions: %w", errors.ErrUnsupported))
	}
	a.cfg.AppPartitions = appParts
}

func (a *actualizers) stop() {
	wg := sync.WaitGroup{}
	for _, app := range a.apps {
		wg.Add(1)
		go func(app *appActualizers) {
			app.stop()
			wg.Done()
		}(app)
	}
	wg.Wait()
}

func (app *appActualizers) stop() {
	wg := sync.WaitGroup{}
	for _, p := range app.parts {
		wg.Add(1)
		go func(p *partActualizers) {
			p.stop()
			wg.Done()
		}(p)
	}
	wg.Wait()
}

func (p *partActualizers) exists(n appdef.QName) bool {
	_, ok := p.run[n]
	return ok
}

func (p *partActualizers) start(n appdef.QName) error {
	run := &struct {
		actualizer *asyncActualizer
		cancel     func()
	}{
		actualizer: &asyncActualizer{
			projector: n,
			conf:      p.cfg,
		},
	}

	// TODO: actualizer.Prepare never returns an error. Reduce complexity.
	if err := run.actualizer.Prepare(nil); err != nil {
		return err
	}

	p.run[n] = run

	ctx, cancel := context.WithCancel(p.cfg.Ctx)
	run.cancel = cancel

	p.wg.Add(1)
	go func() {
		run.actualizer.Run(ctx)
		p.wg.Done()
	}()

	return nil
}

func (p *partActualizers) stop() {
	for _, run := range p.run {
		if run.cancel != nil {
			run.cancel()
		}
	}
	p.wg.Wait()
}
