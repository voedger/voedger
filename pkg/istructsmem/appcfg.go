/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

/**
import (
	"fmt"

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

type AppConfigs struct {
	builtIn map[appdef.AppQName]*BuiltInAppConfig
	user    map[appdef.AppQName]*AppConfig
}

func (c *AppConfigs) AddBuiltInConfig(name appdef.AppQName, def appdef.IAppDefBuilder) *BuiltInAppConfig {
	cfg := BuiltInAppConfig{
		AppConfig: AppConfig{
			name: name,
		},
	}

	if err := cfg.prepare(def); err != nil {
		panic(fmt.Errorf("unable prepare built-in application «%v»: %w", name, err))
	}

	c.builtIn[name] = &cfg
	return &cfg
}

type AppConfig struct {
	name         appdef.AppQName
	clusterAppID istructs.ClusterAppID

	appDef           appdef.IAppDef
	params           AppConfigParams
	numAppWorkspaces istructs.NumAppWorkspaces

	dynoSchemes *dynobuf.DynoBufSchemes
	rateLimits  functionRateLimits

	storage    istorage.IAppStorage
	versions   *vers.Versions
	qNames     *qnames.QNames
	cNames     *containers.Containers
	singletons *singletons.Singletons

	prepared bool
}

// AppDef returns application definition
func (cfg AppConfig) AppDef() appdef.IAppDef { return cfg.appDef }

// ClusterAppID returns application cluster ID
func (cfg AppConfig) ClusterAppID() istructs.ClusterAppID { return cfg.clusterAppID }

// Name returns application name
func (cfg AppConfig) Name() appdef.AppQName { return cfg.name }

// NumAppWorkspaces returns number of application workspaces
func (cfg AppConfig) NumAppWorkspaces() istructs.NumAppWorkspaces { return cfg.numAppWorkspaces }

// Params returns application configuration parameters
func (cfg AppConfig) Params() AppConfigParams { return cfg.params }

// Prepare prepares application configuration to use.
// It creates config globals and should be called from thread-safe code
func (cfg AppConfig) Prepare(buckets irates.IBuckets, appStorage istorage.IAppStorage) error {
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

	if cfg.numAppWorkspaces <= 0 {
		return fmt.Errorf("%s: %w", cfg.Name, ErrNumAppWorkspacesNotSet)
	}

	cfg.prepared = true
	return nil
}

type BuiltInAppConfig struct {
	AppConfig

	Resources Resources

	appDefBuilder   appdef.IAppDefBuilder
	syncProjectors  istructs.Projectors
	asyncProjectors istructs.Projectors
	cudValidators   []istructs.CUDValidator
	eventValidators []istructs.EventValidator
}
**/
