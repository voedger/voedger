/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/sys/smtp"

	bp3def "github.com/untillpro/airs-scheme/bp3"
	"github.com/voedger/voedger/pkg/appdef"
	registryapp "github.com/voedger/voedger/pkg/apps/sys/registry"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz/wskinds"
	test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

const (
	TestEmail       = "123@123.com"
	TestEmail2      = "124@124.com"
	TestEmail3      = "125@125.com"
	TestServicePort = 10000
)

var (
	QNameTestWSKind               = appdef.NewQName("my", "WSKind")
	QNameTestTable                = appdef.NewQName("untill", "air_table_plan")
	QNameTestView                 = appdef.NewQName("my", "View")
	QNameTestEmailVerificationDoc = appdef.NewQName("test", "Doc")
	QNameCDocTestConstraints      = appdef.NewQName("test", "DocConstraints")
	QNameTestSingleton            = appdef.NewQName("test", "Config")
	QNameCmdRated                 = appdef.NewQName(appdef.SysPackage, "RatedCmd")
	QNameQryRated                 = appdef.NewQName(appdef.SysPackage, "RatedQry")

	// BLOBMaxSize 5
	SharedConfig_Simple = NewSharedHITConfig(
		WithApp(istructs.AppQName_test1_app1, ProvideSimpleApp,
			WithWorkspaceTemplate(QNameTestWSKind, "test_template", test_template.TestTemplateFS),
			WithUserLogin("login", "pwd"),
			WithUserLogin(TestEmail, "1"),
			WithUserLogin(TestEmail2, "1"),
			WithUserLogin(TestEmail3, "1"),
			WithChildWorkspace(QNameTestWSKind, "test_ws", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
		),
		WithApp(istructs.AppQName_test1_app2, ProvideSimpleApp, WithUserLogin("login", "1")),
		WithHVMConfig(func(cfg *vvm.HVMConfig) {
			// for impl_reverseproxy_test
			cfg.Routes["/grafana"] = fmt.Sprintf("http://127.0.0.1:%d", TestServicePort)
			cfg.RoutesRewrite["/grafana-rewrite"] = fmt.Sprintf("http://127.0.0.1:%d/rewritten", TestServicePort)
			cfg.RouteDefault = fmt.Sprintf("http://127.0.0.1:%d/not-found", TestServicePort)
			cfg.RouteDomains["localhost"] = "http://127.0.0.1"
		}),
		WithCleanup(func(hit *HIT) {
			MockCmdExec = func(input string) error { panic("") }
			MockQryExec = func(input string, callback istructs.ExecQueryCallback) error { panic("") }
		}),
	)
	MockQryExec func(input string, callback istructs.ExecQueryCallback) error
	MockCmdExec func(input string) error
)

func EmptyApp(hvmCfg *vvm.HVMConfig, hvmAPI vvm.HVMAPI, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {
	registryapp.Provide(smtp.Cfg{})(hvmCfg, hvmAPI, cfg, adf, sep)
	adf.AddStruct(QNameTestWSKind, appdef.DefKind_CDoc).
		AddField("IntFld", appdef.DataKind_int32, true).
		AddField("StrFld", appdef.DataKind_string, false).
		SetSingleton()
	sep.ExtensionPoint(wskinds.EPWorkspaceKind).Add(QNameTestWSKind)
}

func ProvideSimpleApp(hvmCfg *vvm.HVMConfig, hvmAPI vvm.HVMAPI, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, sep vvm.IStandardExtensionPoints) {

	// sys package
	sys.Provide(hvmCfg.TimeFunc, cfg, adf, hvmAPI, smtp.Cfg{}, sep)

	const simpleAppBLOBMaxSize = 5
	hvmCfg.BLOBMaxSize = simpleAppBLOBMaxSize
	bp3def.ProvideBP3Defs(adf)
	adf.AddStruct(QNameTestWSKind, appdef.DefKind_CDoc).
		AddField("IntFld", appdef.DataKind_int32, true).
		AddField("StrFld", appdef.DataKind_string, false).
		SetSingleton()

	projectors.ProvideViewDef(adf, QNameTestView, func(b appdef.IViewBuilder) {
		b.PartKeyDef().AddField("ViewIntFld", appdef.DataKind_int32, true)
		b.ClustColsDef().AddField("ViewStrFld", appdef.DataKind_string, true)
	})
	sep.ExtensionPoint(wskinds.EPWorkspaceKind).Add(QNameTestWSKind)

	// for impl_verifier_test
	adf.AddStruct(QNameTestEmailVerificationDoc, appdef.DefKind_CDoc).
		AddVerifiedField("EmailField", appdef.DataKind_string, true, appdef.VerificationKind_EMail).
		AddVerifiedField("PhoneField", appdef.DataKind_string, false, appdef.VerificationKind_Phone).
		AddField("NonVerifiedField", appdef.DataKind_string, false)

	// for impl_uniques_test
	adf.AddStruct(QNameCDocTestConstraints, appdef.DefKind_CDoc).
		AddField("Int", appdef.DataKind_int32, true).
		AddField("Str", appdef.DataKind_string, true).
		AddField("Bool", appdef.DataKind_bool, true).
		AddField("Float32", appdef.DataKind_float32, false).
		AddField("Bytes", appdef.DataKind_bytes, true)
	cfg.Uniques.Add(QNameCDocTestConstraints, []string{"Int", "Bool", "Str"})
	cfg.Uniques.Add(QNameCDocTestConstraints, []string{"Bytes"})

	// for singletons test
	signletonDefBuilder := adf.AddStruct(QNameTestSingleton, appdef.DefKind_CDoc)
	signletonDefBuilder.
		AddField("Fld1", appdef.DataKind_string, true).
		SetSingleton()

	// for rates test
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQryRated,
		appdef.NullQName,
		adf.AddStruct(appdef.NewQName(appdef.SysPackage, "RatedQryParams"), appdef.DefKind_Object).
			AddField("Fld", appdef.DataKind_string, false).QName(),
		istructsmem.NullQueryExec,
	))
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCmdRated,
		adf.AddStruct(appdef.NewQName(appdef.SysPackage, "RatedCmdParams"), appdef.DefKind_Object).
			AddField("Fld", appdef.DataKind_string, false).QName(),
		appdef.NullQName,
		appdef.NullQName,
		istructsmem.NullCommandExec,
	))

	// per-app limits
	cfg.FunctionRateLimits.AddAppLimit(QNameCmdRated, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 2,
	})
	cfg.FunctionRateLimits.AddAppLimit(QNameQryRated, istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 2,
	})

	// per-workspace limits
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameCmdRated, istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 4,
	})
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQryRated, istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 4,
	})

	mockQryQName := appdef.NewQName(appdef.SysPackage, "MockQry")
	mockQryParamsQName := appdef.NewQName(appdef.SysPackage, "MockQryParams")
	adf.AddStruct(mockQryParamsQName, appdef.DefKind_Object).
		AddField(field_Input, appdef.DataKind_string, true)

	mockQryResQName := appdef.NewQName(appdef.SysPackage, "MockQryResult")
	mockQryResScheme := adf.AddStruct(mockQryResQName, appdef.DefKind_Object)
	mockQryResScheme.AddField("Res", appdef.DataKind_string, true)

	mockQry := istructsmem.NewQueryFunction(mockQryQName, mockQryParamsQName, mockQryResQName,
		func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockQryExec(input, callback)
		},
	)
	cfg.Resources.Add(mockQry)

	mockCmdQName := appdef.NewQName(appdef.SysPackage, "MockCmd")
	mockCmdParamsQName := appdef.NewQName(appdef.SysPackage, "MockCmdParams")
	adf.AddStruct(mockCmdParamsQName, appdef.DefKind_Object).
		AddField(field_Input, appdef.DataKind_string, true)

	execCmdMockCmd := func(cf istructs.ICommandFunction, args istructs.ExecCommandArgs) (err error) {
		input := args.ArgumentObject.AsString(field_Input)
		return MockCmdExec(input)
	}
	mockCmd := istructsmem.NewCommandFunction(mockCmdQName, mockCmdParamsQName, appdef.NullQName, appdef.NullQName, execCmdMockCmd)
	cfg.Resources.Add(mockCmd)
}
