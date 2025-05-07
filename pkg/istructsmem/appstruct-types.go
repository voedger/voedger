/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/isequencer"
	istorage "github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/containers"
	"github.com/voedger/voedger/pkg/istructsmem/internal/dynobuf"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/singletons"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

// AppConfigsType: map of applications configurators
// does contain stateless resources
type AppConfigsType map[appdef.AppQName]*AppConfigType

// AddAppConfig: adds new config for specified application or replaces if exists
func (cfgs *AppConfigsType) AddAppConfig(name appdef.AppQName, id istructs.ClusterAppID, def appdef.IAppDef,
	wsCount istructs.NumAppWorkspaces) *AppConfigType {
	c := newAppConfig(name, id, def, wsCount)
	(*cfgs)[name] = c
	return c
}

// AddBuiltInAppConfig: adds new config for specified builtin application or replaces if exists
func (cfgs *AppConfigsType) AddBuiltInAppConfig(appName appdef.AppQName, appDef appdef.IAppDefBuilder) *AppConfigType {
	c := newBuiltInAppConfig(appName, appDef)

	(*cfgs)[appName] = c
	return c
}

// GetConfig: gets config for specified application
func (cfgs *AppConfigsType) GetConfig(appName appdef.AppQName) *AppConfigType {
	c, ok := (*cfgs)[appName]
	if !ok {
		panic(fmt.Errorf("unable return configuration for unknown application «%v»: %w", appName, istructs.ErrAppNotFound))
	}
	return c
}

// AppConfigType: configuration for application workflow
type AppConfigType struct {
	Name         appdef.AppQName
	ClusterAppID istructs.ClusterAppID

	appDefBuilder appdef.IAppDefBuilder
	AppDef        appdef.IAppDef
	Resources     Resources // does not contain stateless funcs

	// Application configuration parameters
	Params AppConfigParams

	dynoSchemes *dynobuf.DynoBufSchemes

	storage            istorage.IAppStorage // will be initialized on prepare()
	versions           *vers.Versions
	qNames             *qnames.QNames
	cNames             *containers.Containers
	singletons         *singletons.Singletons
	prepared           bool
	app                *appStructsType
	FunctionRateLimits functionRateLimits
	syncProjectors     istructs.Projectors
	asyncProjectors    istructs.Projectors
	cudValidators      []istructs.CUDValidator
	eventValidators    []istructs.EventValidator
	numAppWorkspaces   istructs.NumAppWorkspaces
	jobs               []BuiltinJob
	seqTypes           map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number
}

func newAppConfig(name appdef.AppQName, id istructs.ClusterAppID, def appdef.IAppDef, wsCount istructs.NumAppWorkspaces) *AppConfigType {
	cfg := AppConfigType{
		Name:             name,
		ClusterAppID:     id,
		Params:           makeAppConfigParams(),
		syncProjectors:   make(istructs.Projectors),
		asyncProjectors:  make(istructs.Projectors),
		numAppWorkspaces: wsCount,
		seqTypes:         map[isequencer.WSKind]map[isequencer.SeqID]isequencer.Number{},
	}

	cfg.AppDef = def
	cfg.Resources = NewResources()

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

func newBuiltInAppConfig(appName appdef.AppQName, appDef appdef.IAppDefBuilder) *AppConfigType {
	id, ok := istructs.ClusterApps[appName]
	if !ok {
		panic(fmt.Errorf("unable construct configuration for unknown application «%v»: %w", appName, istructs.ErrAppNotFound))
	}

	def, err := appDef.Build()
	if err != nil {
		panic(fmt.Errorf("%v: unable build application: %w", appName, err))
	}

	cfg := newAppConfig(appName, id, def, 0)
	cfg.appDefBuilder = appDef

	return cfg
}

// prepare: prepares application configuration to use. It creates config globals and must be called from thread-safe code
func (cfg *AppConfigType) prepare(buckets irates.IBuckets, appStorage istorage.IAppStorage) error {
	// if cfg.QNameID == istructs.NullClusterAppID {…} — unnecessary check. QNameID must be checked before prepare()

	if cfg.prepared {
		return nil
	}

	if cfg.appDefBuilder != nil {
		// BuiltIn application, appDefBuilder can be changed after add config
		appDef, err := cfg.appDefBuilder.Build()
		if err != nil {
			return fmt.Errorf("%v: unable rebuild changed application: %w", cfg.Name, err)
		}
		cfg.AppDef = appDef
	}

	cfg.dynoSchemes.Prepare(cfg.AppDef)

	// prepare IAppStorage
	cfg.storage = appStorage

	// prepare system views versions
	if err := cfg.versions.Prepare(cfg.storage); err != nil {
		return err
	}

	// prepare QNames
	if err := cfg.qNames.Prepare(cfg.storage, cfg.versions, cfg.AppDef); err != nil {
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

	if err := cfg.validateJobs(); err != nil {
		return err
	}

	if cfg.numAppWorkspaces <= 0 {
		return ErrNumAppWorkspacesNotSet(cfg.Name)
	}

	for _, iWorkspace := range cfg.AppDef.Workspaces() {
		if iWorkspace.Abstract() {
			continue
		}
		wsKindQNameID, err := cfg.QNameID(iWorkspace.Descriptor())
		if err != nil {
			// notest
			return err
		}
		cfg.AddSeqType(isequencer.WSKind(wsKindQNameID), isequencer.SeqID(istructs.QNameIDRecordIDSequence), isequencer.Number(istructs.FirstUserRecordID))
		cfg.AddSeqType(isequencer.WSKind(wsKindQNameID), isequencer.SeqID(istructs.QNameIDWLogOffsetSequence), isequencer.Number(istructs.FirstOffset))
	}

	cfg.prepared = true
	return nil
}

func (cfg *AppConfigType) validateJobs() error {
	for _, job := range cfg.jobs {
		if cfg.AppDef.Type(job.Name).Kind() == appdef.TypeKind_null {
			return fmt.Errorf("exec of job %s is defined but the job is not defined in SQL", job.Name)
		}
	}
	return nil
}

func (cfg *AppConfigType) validateResources() error {

	for qName := range cfg.Resources.Resources {
		if appdef.Extension(cfg.AppDef.Type, qName) == nil {
			return fmt.Errorf("exec of func %s is defined but the func is not defined in SQL", qName)
		}
	}

	for _, prj := range cfg.syncProjectors {
		if appdef.Projector(cfg.AppDef.Type, prj.Name) == nil {
			return fmt.Errorf("exec of sync projector %s is defined but the projector is not defined in SQL", prj.Name)
		}
	}
	for _, prj := range cfg.asyncProjectors {
		if appdef.Projector(cfg.AppDef.Type, prj.Name) == nil {
			return fmt.Errorf("exec of async projector %s is defined but the projector is not defined in SQL", prj.Name)
		}
	}
	return nil
}

func (cfg *AppConfigType) AddSyncProjectors(pp ...istructs.Projector) {
	for _, p := range pp {
		cfg.syncProjectors[p.Name] = p
	}
}

func (cfg *AppConfigType) AddAsyncProjectors(pp ...istructs.Projector) {
	for _, p := range pp {
		cfg.asyncProjectors[p.Name] = p
	}
}

func (cfg *AppConfigType) AddCUDValidators(cudValidators ...istructs.CUDValidator) {
	cfg.cudValidators = append(cfg.cudValidators, cudValidators...)
}

func (cfg *AppConfigType) AddEventValidators(eventValidators ...istructs.EventValidator) {
	cfg.eventValidators = append(cfg.eventValidators, eventValidators...)
}

func (cfg *AppConfigType) AddJobs(jobs ...BuiltinJob) {
	cfg.jobs = append(cfg.jobs, jobs...)
}

func (cfg *AppConfigType) AsyncProjectors() istructs.Projectors {
	return cfg.asyncProjectors
}

// Returns is application configuration prepared
func (cfg *AppConfigType) Prepared() bool {
	return cfg.prepared
}

func (cfg *AppConfigType) SyncProjectors() istructs.Projectors {
	return cfg.syncProjectors
}

func (cfg *AppConfigType) BuiltingJobs() []BuiltinJob {
	return cfg.jobs
}

func (cfg *AppConfigType) AddSeqType(wsKind isequencer.WSKind, seqID isequencer.SeqID, initialNumber isequencer.Number) {
	wsKindSeqTypes, ok := cfg.seqTypes[wsKind]
	if !ok {
		wsKindSeqTypes = map[isequencer.SeqID]isequencer.Number{}
		cfg.seqTypes[wsKind] = wsKindSeqTypes
	}
	if _, ok := wsKindSeqTypes[seqID]; ok {
		panic(fmt.Sprintf("initial number for SeqID %d in WSKind %d is already set", seqID, wsKind))
	}
	wsKindSeqTypes[seqID] = initialNumber
}

// need to build view.sys.NextBaseWSID and view.sys.projectionOffsets
// could be called on application build stage only
//
// Should be used for built-in applications only.
func (cfg *AppConfigType) AppDefBuilder() appdef.IAppDefBuilder {
	if cfg.prepared {
		panic("IAppStructsProvider.AppStructs() is called already for the app -> IAppDef is built already -> wrong to work with IAppDefBuilder")
	}
	return cfg.appDefBuilder
}

func (cfg *AppConfigType) NumAppWorkspaces() istructs.NumAppWorkspaces {
	return cfg.numAppWorkspaces
}

// must be called after creating the AppConfigType because app will provide the deployment descriptor with the actual NumAppWorkspaces after willing the AppConfigType
// so fisrt create AppConfigType, use it on app provide, then set the actual NumAppWorkspaces
func (cfg *AppConfigType) SetNumAppWorkspaces(naw istructs.NumAppWorkspaces) {
	if cfg.prepared {
		panic("must not set NumAppWorkspaces after first IAppStructsProvider.AppStructs() call because the app is considered working")
	}
	cfg.numAppWorkspaces = naw
}

func (cfg *AppConfigType) QNameID(qName appdef.QName) (istructs.QNameID, error) {
	return cfg.qNames.ID(qName)
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
