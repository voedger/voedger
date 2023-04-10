/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/untillpro/voedger/pkg/irates"
	istorage "github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
)

// AppConfigsType: map of applications configurators
type AppConfigsType map[istructs.AppQName]*AppConfigType

// AddConfig: adds new config for specified application
func (cfgs *AppConfigsType) AddConfig(appName istructs.AppQName) *AppConfigType {
	c := newAppConfig(appName)

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

// AppConfigType: configuration for application workflow (resources, schemas, etc.)
type AppConfigType struct {
	Name                    istructs.AppQName
	QNameID                 istructs.ClusterAppID
	Resources               ResourcesType
	Schemas                 SchemasCacheType
	Uniques                 *implIUniques
	storage                 istorage.IAppStorage // will be initialized on prepare()
	versions                verionsCacheType
	qNames                  qNameCacheType
	cNames                  containerNameCacheType
	singletons              singletonsCacheType
	prepared                bool
	app                     *appStructsType
	FunctionRateLimits      functionRateLimits
	syncProjectorFactories  []istructs.ProjectorFactory
	asyncProjectorFactories []istructs.ProjectorFactory
	cudValidators           []istructs.CUDValidator
	eventValidators         []istructs.EventValidator
}

func newAppConfig(appName istructs.AppQName) *AppConfigType {
	cfg := AppConfigType{Name: appName}

	qNameID, ok := istructs.ClusterApps[appName]
	if !ok {
		panic(fmt.Errorf("unable construct configuration for unknown application «%v»: %w", appName, istructs.ErrAppNotFound))
	}
	cfg.QNameID = qNameID

	cfg.Resources = newResources(&cfg)
	cfg.Schemas = newSchemaCache(&cfg)
	cfg.versions = newVerionsCache(&cfg)
	cfg.qNames = newQNameCache(&cfg)
	cfg.cNames = newContainerNameCache(&cfg)
	cfg.singletons = newSingletonsCache(&cfg)
	cfg.FunctionRateLimits = functionRateLimits{
		limits: map[istructs.QName]map[istructs.RateLimitKind]istructs.RateLimit{},
	}
	cfg.Uniques = newUniques()
	return &cfg
}

// prepare: prepares application configuration to use. It creates config globals and must be called from thread-safe code
func (cfg *AppConfigType) prepare(buckets irates.IBuckets, appStorage istorage.IAppStorage) error {
	// if cfg.QNameID == istructs.NullClusterAppID {…} — unnecessary check. QNameIDmust be checked before prepare()

	if cfg.prepared {
		return nil
	}

	// prepare IAppStorage
	cfg.storage = appStorage

	// build key schemas for all views
	if err := cfg.Schemas.buildViewKeySchemas(); err != nil {
		return err
	}

	// validate schemas
	if err := cfg.Schemas.ValidateSchemas(); err != nil {
		return err
	}

	// create dynoBufferSchemes
	cfg.Schemas.createDynoBufferSchemes()

	// prepare system views versions
	if err := cfg.versions.prepare(); err != nil {
		return err
	}

	// prepare QNames
	if err := cfg.qNames.prepare(); err != nil {
		return err
	}

	// prepare container names
	if err := cfg.cNames.prepare(); err != nil {
		return err
	}

	// prepare singleton CDOCs
	if err := cfg.singletons.prepare(); err != nil {
		return err
	}

	// prepare functions rate limiter
	cfg.FunctionRateLimits.prepare(buckets)

	// validate uniques
	if err := cfg.Uniques.validate(cfg); err != nil {
		return err
	}

	cfg.prepared = true
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
