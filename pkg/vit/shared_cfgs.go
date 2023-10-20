/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"

	"github.com/voedger/voedger/pkg/appdef"
	registryapp "github.com/voedger/voedger/pkg/apps/sys/registry"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/sys"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

const (
	TestEmail       = "123@123.com"
	TestEmail2      = "124@124.com"
	TestEmail3      = "125@125.com"
	TestServicePort = 10000
	app1Name        = "app1"
)

var (
	QNameApp1_TestWSKind               = appdef.NewQName(app1Name, "WSKind")
	QNameTestView                      = appdef.NewQName("my", "View")
	QNameApp1_TestEmailVerificationDoc = appdef.NewQName(app1Name, "Doc")
	QNameApp1_CDocTestConstraints      = appdef.NewQName(app1Name, "DocConstraints")
	QNameCmdRated                      = appdef.NewQName(appdef.SysPackage, "RatedCmd")
	QNameQryRated                      = appdef.NewQName(appdef.SysPackage, "RatedQry")
	QNameODoc1                         = appdef.NewQName(app1Name, "odoc1")
	QNameODoc2                         = appdef.NewQName(app1Name, "odoc2")
	TestSMTPCfg                        = smtp.Cfg{
		Username: "username@gmail.com",
	}

	// BLOBMaxSize 5
	SharedConfig_App1 = NewSharedVITConfig(
		WithApp(istructs.AppQName_test1_app1, ProvideApp1,
			WithWorkspaceTemplate(QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
			WithUserLogin("login", "pwd"),
			WithUserLogin(TestEmail, "1"),
			WithUserLogin(TestEmail2, "1"),
			WithUserLogin(TestEmail3, "1"),
			WithChildWorkspace(QNameApp1_TestWSKind, "test_ws", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
		),
		WithApp(istructs.AppQName_test1_app2, ProvideApp1, WithUserLogin("login", "1")),
		WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// for impl_reverseproxy_test
			cfg.Routes["/grafana"] = fmt.Sprintf("http://127.0.0.1:%d", TestServicePort)
			cfg.RoutesRewrite["/grafana-rewrite"] = fmt.Sprintf("http://127.0.0.1:%d/rewritten", TestServicePort)
			cfg.RouteDefault = fmt.Sprintf("http://127.0.0.1:%d/not-found", TestServicePort)
			cfg.RouteDomains["localhost"] = "http://127.0.0.1"

			const app1_BLOBMaxSize = 5
			cfg.BLOBMaxSize = app1_BLOBMaxSize
		}),
		WithCleanup(func(_ *VIT) {
			MockCmdExec = func(input string) error { panic("") }
			MockQryExec = func(input string, callback istructs.ExecQueryCallback) error { panic("") }
		}),
	)
	MockQryExec func(input string, callback istructs.ExecQueryCallback) error
	MockCmdExec func(input string) error
)

func ProvideApp2(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	registryapp.Provide(smtp.Cfg{}, false)(apis, cfg, appDefBuilder, ep)
	apps.Parse(SchemaTestApp2, "app2", ep)
}

func ProvideApp1(apis apps.APIs, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	// sys package
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("no build info")
	}
	sys.Provide(cfg, adf, TestSMTPCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
		apis.NumCommandProcessors, buildInfo, apis.IAppStorageProvider, false)

	apps.Parse(SchemaTestApp1, "app1", ep)

	projectors.ProvideViewDef(adf, QNameTestView, func(view appdef.IViewBuilder) {
		view.KeyBuilder().PartKeyBuilder().AddField("ViewIntFld", appdef.DataKind_int32)
		view.KeyBuilder().ClustColsBuilder().AddStringField("ViewStrFld", appdef.DefaultFieldMaxLength)
		view.ValueBuilder().AddBytesField("ViewByteFld", false, appdef.FLD_MaxLen(512))
	})

	// for rates test
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQryRated,
		appdef.NullQName,
		adf.AddObject(appdef.NewQName(appdef.SysPackage, "RatedQryParams")).
			AddField("Fld", appdef.DataKind_string, false).(appdef.IType).QName(),
		istructsmem.NullQueryExec,
	))
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCmdRated,
		adf.AddObject(appdef.NewQName(appdef.SysPackage, "RatedCmdParams")).
			AddField("Fld", appdef.DataKind_string, false).(appdef.IType).QName(),
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
	adf.AddObject(mockQryParamsQName).
		AddField(field_Input, appdef.DataKind_string, true)

	mockQryResQName := appdef.NewQName(appdef.SysPackage, "MockQryResult")
	mockQryResScheme := adf.AddObject(mockQryResQName)
	mockQryResScheme.AddField("Res", appdef.DataKind_string, true)

	mockQry := istructsmem.NewQueryFunction(mockQryQName, mockQryParamsQName, mockQryResQName,
		func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockQryExec(input, callback)
		},
	)
	cfg.Resources.Add(mockQry)

	mockCmdQName := appdef.NewQName(appdef.SysPackage, "MockCmd")
	mockCmdParamsQName := appdef.NewQName(appdef.SysPackage, "MockCmdParams")
	adf.AddObject(mockCmdParamsQName).
		AddField(field_Input, appdef.DataKind_string, true)

	execCmdMockCmd := func(args istructs.ExecCommandArgs) (err error) {
		input := args.ArgumentObject.AsString(field_Input)
		return MockCmdExec(input)
	}
	mockCmd := istructsmem.NewCommandFunction(mockCmdQName, mockCmdParamsQName, appdef.NullQName, appdef.NullQName, execCmdMockCmd)
	cfg.Resources.Add(mockCmd)

	testCmdResult := appdef.NewQName(appdef.SysPackage, "TestCmdResult")
	testCmdParams := appdef.NewQName(appdef.SysPackage, "TestCmdParams")
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "TestCmd"),
		adf.AddObject(testCmdParams).
			AddField("Arg1", appdef.DataKind_int32, true).(appdef.IType).QName(),
		appdef.NullQName,
		adf.AddObject(testCmdResult).
			AddField("Int", appdef.DataKind_int32, true).
			AddField("Str", appdef.DataKind_string, false).(appdef.IType).QName(),
		func(args istructs.ExecCommandArgs) (err error) {
			key, err := args.State.KeyBuilder(state.Result, testCmdResult)
			if err != nil {
				return err
			}

			value, err := args.Intents.NewValue(key)
			if err != nil {
				return err
			}

			arg1 := args.ArgumentObject.AsInt32("Arg1")
			switch arg1 {
			case 1:
				value.PutString("Str", "Str")
				value.PutInt32("Int", 42)
			case 2:
				value.PutInt32("Int", 42)
			case 3:
				value.PutString("Str", "Str")
			case 4:
				value.PutString("Int", "wrong")
			}
			return nil
		},
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "CmdODocOne"),
		QNameODoc1,
		appdef.NullQName,
		appdef.NullQName,
		istructsmem.NullCommandExec,
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(appdef.SysPackage, "CmdODocTwo"),
		QNameODoc2,
		QNameODoc2,
		appdef.NullQName,
		istructsmem.NullCommandExec,
	))
}
