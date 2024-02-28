/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/istructsmem/internal/dynobuf"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/singletons"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

// AppConfigsType: map of applications configurators
type AppConfigsType map[istructs.AppQName]*AppConfigType

// AddConfig: adds new config for specified application or replaces if exists
func (cfgs *AppConfigsType) AddConfig(appName istructs.AppQName, appDef appdef.IAppDefBuilder) *AppConfigType {
	c := newAppConfig(appName, appDef)

	(*cfgs)[appName] = c
	return c
}

// GetConfig: gets config for specified application
func (cfgs *AppConfigsType) GetConfig(appName istructs.AppQName) *AppConfigType {
	c, ok := (*cfgs)[appName]
	if !ok {
		panic(fmt.Errorf("unable return configuration for unknown application «%v»: %w", appName, istructs.ErrAppNotFound))
	}
	return c
}

// AppConfigType: configuration for application workflow
type AppConfigType struct {
	Name         istructs.AppQName
	ClusterAppID istructs.ClusterAppID

	appDefBuilder appdef.IAppDefBuilder
	AppDef        appdef.IAppDef
	Resources     Resources

	// Application configuration parameters
	Params AppConfigParams

	dynoSchemes *dynobuf.DynoBufSchemes

	storage                 istorage.IAppStorage // will be initialized on prepare()
	versions                *vers.Versions
	qNames                  *qnames.QNames
	cNames                  *containers.Containers
	singletons              *singletons.Singletons
	prepared                bool
	app                     *appStructsType
	FunctionRateLimits      functionRateLimits
	syncProjectorFactories  []istructs.ProjectorFactory
	asyncProjectorFactories []istructs.ProjectorFactory
	cudValidators           []istructs.CUDValidator
	eventValidators         []istructs.EventValidator
}

func newAppConfig(appName istructs.AppQName, appDef appdef.IAppDefBuilder) *AppConfigType {
	cfg := AppConfigType{
		Name:   appName,
		Params: makeAppConfigParams(),
	}

	qNameID, ok := istructs.ClusterApps[appName]
	if !ok {
		panic(fmt.Errorf("unable construct configuration for unknown application «%v»: %w", appName, istructs.ErrAppNotFound))
	}
	cfg.ClusterAppID = qNameID

	cfg.appDefBuilder = appDef
	app, err := appDef.Build()
	if err != nil {
		panic(fmt.Errorf("%v: unable build application: %w", appName, err))
	}
	cfg.AppDef = app
	cfg.Resources = newResources(&cfg)

	cfg.dynoSchemes = dynobuf.New()

	cfg.versions = vers.New()
	cfg.qNames = qnames.New()
	cfg.cNames = containers.New()
	cfg.singletons = singletons.New()

	cfg.FunctionRateLimits = functionRateLimits{
		limits: map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit{},
	}
	return &cfg
}

// prepare: prepares application configuration to use. It creates config globals and must be called from thread-safe code
func (cfg *AppConfigType) prepare(buckets irates.IBuckets, appStorage istorage.IAppStorage) error {
	// if cfg.QNameID == istructs.NullClusterAppID {…} — unnecessary check. QNameIDmust be checked before prepare()

	if cfg.prepared {
		return nil
	}

	app, err := cfg.appDefBuilder.Build()
	if err != nil {
		return fmt.Errorf("%v: unable rebuild changed application: %w", cfg.Name, err)
	}
	cfg.AppDef = app

	cfg.dynoSchemes.Prepare(cfg.AppDef)

	// prepare IAppStorage
	cfg.storage = appStorage

	// prepare system views versions
	if err := cfg.versions.Prepare(cfg.storage); err != nil {
		return err
	}

	// prepare QNames
	if err := cfg.qNames.Prepare(cfg.storage, cfg.versions, cfg.AppDef, &cfg.Resources); err != nil {
		return err
	}

	// prepare container names
	if err := cfg.cNames.Prepare(cfg.storage, cfg.versions, cfg.AppDef); err != nil {
		return err
	}

	// prepare singleton CDocs
	if err := cfg.singletons.Prepare(cfg.storage, cfg.versions, cfg.AppDef); err != nil {
		return err
	}

	// prepare functions rate limiter
	cfg.FunctionRateLimits.prepare(buckets)

	if err := cfg.validateResources(); err != nil {
		return err
	}

	cfg.prepared = true
	return nil
}

func (cfg *AppConfigType) validateResources() error {
	err := iterate.ForEachError(cfg.AppDef.Types, func(tp appdef.IType) error {
		switch tp.Kind() {
		case appdef.TypeKind_Query, appdef.TypeKind_Command:
			r := cfg.Resources.QueryResource(tp.QName())
			if r.QName() == appdef.NullQName {
				return fmt.Errorf("exec of func %s is not defined", tp.QName())
			}
		case appdef.TypeKind_Projector:
			syncFound := false
			asyncFound := false
			for _, pFunc := range cfg.syncProjectorFactories {
				p := pFunc(0)
				if p.Name == tp.QName() {
					syncFound = true
					break
				}
			}
			for _, pFunc := range cfg.asyncProjectorFactories {
				p := pFunc(0)
				if p.Name == tp.QName() {
					asyncFound = true
					break
				}
			}
			if !syncFound && !asyncFound {
				return fmt.Errorf("exec of projector %s is not defined", tp.QName())
			}
			if syncFound && asyncFound {
				return fmt.Errorf("exec of projector %s is defined twice: sync and async", tp.QName())
			}
			if tp.(appdef.IProjector).Sync() && asyncFound {
				return fmt.Errorf("exec of sync projector %s is defined as async", tp.QName())
			}
			if !tp.(appdef.IProjector).Sync() && syncFound {
				return fmt.Errorf("exec of async projector %s is defined as sync", tp.QName())
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = iterate.ForEachError(cfg.Resources.Resources, func(qName appdef.QName) error {
		if cfg.AppDef.Type(qName).Kind() == appdef.TypeKind_null {
			return fmt.Errorf("exec of func %s is defined but the func is not defined in SQL", qName)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, pFunc := range cfg.asyncProjectorFactories {
		p := pFunc(0)
		if cfg.AppDef.Type(p.Name).Kind() == appdef.TypeKind_null {
			return fmt.Errorf("exec of projector %s is defined but the projector is not defined in SQL", p.Name)
		}
	}
	for _, pFunc := range cfg.syncProjectorFactories {
		p := pFunc(0)
		if cfg.AppDef.Type(p.Name).Kind() == appdef.TypeKind_null {
			return fmt.Errorf("exec of projector %s is defined but the projector is not defined in SQL", p.Name)
		}
	}
	return nil
}

func (cfg *AppConfigType) AddSyncProjectors(sp ...istructs.ProjectorFactory) {
	cfg.syncProjectorFactories = append(cfg.syncProjectorFactories, sp...)
}

func (cfg *AppConfigType) AddAsyncProjectors(ap ...istructs.ProjectorFactory) {
	cfg.asyncProjectorFactories = append(cfg.asyncProjectorFactories, ap...)
}

func (cfg *AppConfigType) AddCUDValidators(cudValidators ...istructs.CUDValidator) {
	cfg.cudValidators = append(cfg.cudValidators, cudValidators...)
}

func (cfg *AppConfigType) AddEventValidators(eventValidators ...istructs.EventValidator) {
	cfg.eventValidators = append(cfg.eventValidators, eventValidators...)
}

// Returns is application configuration prepared
func (cfg *AppConfigType) Prepared() bool {
	return cfg.prepared
}

// Application configuration parameters
type AppConfigParams struct {
	// PLog events cache size.
	//
	// Default value is DefaultPLogEventCacheSize (10’000 events).
	// Zero (0) means that cache will not be used
	PLogEventCacheSize int
}

func makeAppConfigParams() AppConfigParams {
	return AppConfigParams{
		PLogEventCacheSize: DefaultPLogEventCacheSize, // 10’000
	}
}
