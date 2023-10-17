/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	blobberapp "github.com/voedger/voedger/pkg/apps/sys/blobber"
	registryapp "github.com/voedger/voedger/pkg/apps/sys/registry"
	routerapp "github.com/voedger/voedger/pkg/apps/sys/router"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/authnz/workspace"
	"github.com/voedger/voedger/pkg/sys/smtp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

func NewOwnVITConfig(opts ...vitConfigOptFunc) VITConfig {
	// helper: implicitly append sys apps
	opts = append(opts,
		WithApp(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{}, false)),
		WithApp(istructs.AppQName_sys_blobber, blobberapp.Provide(smtp.Cfg{}, false)),
		WithApp(istructs.AppQName_sys_router, routerapp.Provide(smtp.Cfg{}, false)),
	)
	return VITConfig{
		opts: opts,
	}
}

func NewSharedVITConfig(opts ...vitConfigOptFunc) VITConfig {
	cfg := NewOwnVITConfig(opts...)
	cfg.isShared = true
	return cfg
}

func WithBuilder(builder apps.AppBuilder) AppOptFunc {
	return func(app *app, cfg *vvm.VVMConfig) {
		cfg.VVMAppsBuilder.Add(app.name, builder)
	}
}

// at MainCluster
func WithUserLogin(name, pwd string, opts ...PostConstructFunc) AppOptFunc {
	return func(app *app, _ *vvm.VVMConfig) {
		login := NewLogin(name, pwd, app.name, istructs.SubjectKind_User, istructs.MainClusterID)
		for _, opt := range opts {
			opt(&login)
		}
		app.logins = append(app.logins, login)
	}
}

func WithWorkspaceTemplate(wsKind appdef.QName, templateName string, templateFS coreutils.EmbedFS) AppOptFunc {
	return func(app *app, cfg *vvm.VVMConfig) {
		app.wsTemplateFuncs = append(app.wsTemplateFuncs, func(ep extensionpoints.IExtensionPoint) {
			epWSKindTemplates := ep.ExtensionPoint(workspace.EPWSTemplates).ExtensionPoint(wsKind)
			epWSKindTemplates.AddNamed(templateName, templateFS)
		})
	}
}

func WithChildWorkspace(wsKind appdef.QName, name, templateName string, templateParams string, ownerLoginName string, wsInitData map[string]interface{}, opts ...PostConstructFunc) AppOptFunc {
	return func(app *app, cfg *vvm.VVMConfig) {
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

func WithVVMConfig(configurer func(cfg *vvm.VVMConfig)) vitConfigOptFunc {
	return func(hpc *vitPreConfig) {
		configurer(hpc.vvmCfg)
	}
}

func WithCleanup(cleanup func(*VIT)) vitConfigOptFunc {
	return func(hpc *vitPreConfig) {
		hpc.cleanups = append(hpc.cleanups, cleanup)
	}
}

func WithApp(appQName istructs.AppQName, updater apps.AppBuilder, appOpts ...AppOptFunc) vitConfigOptFunc {
	return func(vpc *vitPreConfig) {
		_, ok := vpc.vitApps[appQName]
		if ok {
			panic("app already added")
		}
		app := &app{
			name: appQName,
			ws:   map[string]WSParams{},
		}
		vpc.vitApps[appQName] = app
		vpc.vvmCfg.VVMAppsBuilder.Add(appQName, updater)
		for _, appOpt := range appOpts {
			appOpt(app, vpc.vvmCfg)
		}
		// to append tests templates to already declared templates
		for _, wsTempalateFunc := range app.wsTemplateFuncs {
			vpc.vvmCfg.VVMAppsBuilder.Add(appQName, func(appAPI apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
				wsTempalateFunc(ep)
			})
		}
	}
}
