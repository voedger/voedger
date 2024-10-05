/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/workspace"
	"github.com/voedger/voedger/pkg/vvm"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	"github.com/voedger/voedger/pkg/vvm/builtin/registryapp"
)

func NewOwnVITConfig(opts ...vitConfigOptFunc) VITConfig {
	// helper: implicitly append sys apps
	opts = append(opts,
		WithApp(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{}, vvm.DefaultNumCommandProcessors)),
		WithApp(istructs.AppQName_sys_cluster, clusterapp.Provide()),
	)
	return VITConfig{opts: opts}
}

func NewSharedVITConfig(opts ...vitConfigOptFunc) VITConfig {
	cfg := NewOwnVITConfig(opts...)
	cfg.isShared = true
	return cfg
}

func WithBuilder(builder builtinapps.Builder) AppOptFunc {
	return func(app *app, cfg *vvm.VVMConfig) {
		cfg.VVMAppsBuilder.Add(app.name, builder)
	}
}

// at MainCluster
func WithUserLogin(name, pwd string, opts ...PostConstructFunc) AppOptFunc {
	return func(app *app, _ *vvm.VVMConfig) {
		login := NewLogin(name, pwd, app.name, istructs.SubjectKind_User, istructs.CurrentClusterID())
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

func WithChild(wsKind appdef.QName, name, templateName string, templateParams string, ownerLoginName string, wsInitData map[string]interface{}, opts ...PostConstructFunc) PostConstructFunc {
	return func(intf interface{}) {
		wsParams := intf.(*WSParams)
		initData, err := json.Marshal(&wsInitData)
		if err != nil {
			panic(err)
		}
		newWSParams := WSParams{
			Name:           name,
			TemplateName:   templateName,
			TemplateParams: templateParams,
			Kind:           wsKind,
			ownerLoginName: ownerLoginName,
			InitDataJSON:   string(initData),
			ClusterID:      istructs.CurrentClusterID(),
			docs:           map[appdef.QName]func(verifiedValues map[string]string) map[string]interface{}{},
		}
		for _, opt := range opts {
			opt(&newWSParams)
		}
		wsParams.childs = append(wsParams.childs, newWSParams)
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
			ClusterID:      istructs.CurrentClusterID(),
			docs:           map[appdef.QName]func(verifiedValues map[string]string) map[string]interface{}{},
		}
		for _, opt := range opts {
			opt(&wsParams)
		}
		app.ws[name] = wsParams
	}
}

func WithDocWithVerifiedFields(name appdef.QName, dataFactory func(verifiedValues map[string]string) map[string]interface{}) PostConstructFunc {
	return func(intf interface{}) {
		switch t := intf.(type) {
		case *Login:
			t.docs[name] = dataFactory
		case *WSParams:
			t.docs[name] = dataFactory
		default:
			panic(fmt.Sprintln(t, name))
		}
	}
}

func WithDoc(name appdef.QName, data map[string]interface{}) PostConstructFunc {
	return WithDocWithVerifiedFields(name, func(verifiedValues map[string]string) map[string]interface{} {
		return data
	})
}

func WithSubject(login string, subjectKind istructs.SubjectKindType, roles []appdef.QName) PostConstructFunc {
	return func(intf interface{}) {
		wsParams := intf.(*WSParams)
		wsParams.subjects = append(wsParams.subjects, subject{
			login:       login,
			subjectKind: subjectKind,
			roles:       roles,
		})
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

func WithInit(initFunc func()) vitConfigOptFunc {
	return func(vpc *vitPreConfig) {
		vpc.initFuncs = append(vpc.initFuncs, initFunc)
	}
}

func WithPostInit(postInitFunc func(vit *VIT)) vitConfigOptFunc {
	return func(vpc *vitPreConfig) {
		vpc.postInitFunc = postInitFunc
	}
}

func WithApp(appQName appdef.AppQName, updater builtinapps.Builder, appOpts ...AppOptFunc) vitConfigOptFunc {
	return func(vpc *vitPreConfig) {
		_, ok := vpc.vitApps[appQName]
		if ok {
			panic("app already added")
		}
		app := &app{
			name:                  appQName,
			ws:                    map[string]WSParams{},
			verifiedValuesIntents: map[string]verifiedValueIntent{},
		}
		vpc.vitApps[appQName] = app
		vpc.vvmCfg.VVMAppsBuilder.Add(appQName, updater)
		for _, appOpt := range appOpts {
			appOpt(app, vpc.vvmCfg)
		}
	}
}

func WithVerifiedValue(docQName appdef.QName, fieldName string, desiredValue string) AppOptFunc {
	return func(app *app, cfg *vvm.VVMConfig) {
		app.verifiedValuesIntents[desiredValue] = verifiedValueIntent{
			docQName:     docQName,
			fieldName:    fieldName,
			desiredValue: desiredValue,
		}
	}
}

func WithSecret(name string, value []byte) vitConfigOptFunc {
	return func(vpc *vitPreConfig) {
		vpc.secrets[name] = value
	}
}
