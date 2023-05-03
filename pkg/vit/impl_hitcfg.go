/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"encoding/json"
	"fmt"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	blobberapp "github.com/voedger/voedger/pkg/apps/sys/blobber"
	registryapp "github.com/voedger/voedger/pkg/apps/sys/registry"
	routerapp "github.com/voedger/voedger/pkg/apps/sys/router"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/vvm"
)

func NewOwnHITConfig(opts ...hitConfigOptFunc) HITConfig {
	// helper: implicitly append sys apps
	opts = append(opts,
		WithApp(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{})),
		WithApp(istructs.AppQName_sys_blobber, blobberapp.Provide(smtp.Cfg{})),
		WithApp(istructs.AppQName_sys_router, routerapp.Provide(smtp.Cfg{})),
	)
	return HITConfig{
		opts: opts,
	}
}

func NewSharedHITConfig(opts ...hitConfigOptFunc) HITConfig {
	cfg := NewOwnHITConfig(opts...)
	cfg.isShared = true
	return cfg
}

func WithBuilder(builder vvm.HVMAppBuilder) AppOptFunc {
	return func(app *app, cfg *vvm.HVMConfig) {
		cfg.HVMAppsBuilder.Add(app.name, builder)
	}
}

// at MainCluster
func WithUserLogin(name, pwd string, opts ...PostConstructFunc) AppOptFunc {
	return func(app *app, _ *vvm.HVMConfig) {
		login := NewLogin(name, pwd, app.name, istructs.SubjectKind_User, istructs.MainClusterID)
		for _, opt := range opts {
			opt(&login)
		}
		app.logins = append(app.logins, login)
	}
}

func WithWorkspaceTemplate(wsKind appdef.QName, templateName string, templateFS utils.EmbedFS) AppOptFunc {
	return func(app *app, cfg *vvm.HVMConfig) {
		app.wsTemplateFuncs = append(app.wsTemplateFuncs, func(sep vvm.IStandardExtensionPoints) {
			epWSKindTemplates := sep.EPWSTemplates().ExtensionPoint(wsKind)
			epWSKindTemplates.AddNamed(templateName, templateFS)
		})
	}
}

func WithChildWorkspace(wsKind appdef.QName, name, templateName string, templateParams string, ownerLoginName string, wsInitData map[string]interface{}, opts ...PostConstructFunc) AppOptFunc {
	return func(app *app, cfg *vvm.HVMConfig) {
		initData, err := json.Marshal(&wsInitData)
		if err != nil {
			panic(err)
		}
		wsParams := WSParams{
			Name:           name,
			TemplateName:   templateName,
			TemplateParams: templateParams,
			Kind:           wsKind,
			ownerLoginName: ownerLoginName,
			InitDataJSON:   string(initData),
			ClusterID:      istructs.MainClusterID,
			singletons:     map[appdef.QName]map[string]interface{}{},
		}
		for _, opt := range opts {
			opt(&wsParams)
		}
		app.ws[name] = wsParams
	}
}

func WithSingleton(name appdef.QName, data map[string]interface{}) PostConstructFunc {
	return func(intf interface{}) {
		switch t := intf.(type) {
		case *Login:
			t.singletons[name] = data
		case *WSParams:
			t.singletons[name] = data
		default:
			panic(fmt.Sprintln(t, name, data))
		}
	}
}

func WithHVMConfig(configurer func(cfg *vvm.HVMConfig)) hitConfigOptFunc {
	return func(hpc *hitPreConfig) {
		configurer(hpc.hvmCfg)
	}
}

func WithCleanup(cleanup func(hit *HIT)) hitConfigOptFunc {
	return func(hpc *hitPreConfig) {
		hpc.cleanups = append(hpc.cleanups, cleanup)
	}
}

func WithApp(appQName istructs.AppQName, updater vvm.HVMAppBuilder, appOpts ...AppOptFunc) hitConfigOptFunc {
	return func(hpc *hitPreConfig) {
		_, ok := hpc.hitApps[appQName]
		if ok {
			panic("app already added")
		}
		app := &app{
			name: appQName,
			ws:   map[string]WSParams{},
		}
		hpc.hitApps[appQName] = app
		hpc.hvmCfg.HVMAppsBuilder.Add(appQName, updater)
		for _, appOpt := range appOpts {
			appOpt(app, hpc.hvmCfg)
		}
		// to append tests templates to already declared templates
		for _, wsTempalateFunc := range app.wsTemplateFuncs {
			hpc.hvmCfg.HVMAppsBuilder.Add(appQName, func(_ *vvm.HVMConfig, hvmAPI vvm.HVMAPI, _ *istructsmem.AppConfigType, _ appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {
				wsTempalateFunc(sep)
			})
		}
	}
}
