/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
)

const (
	TestEmail       = "123@123.com"
	TestEmail2      = "124@124.com"
	TestServicePort = 10000
	app1PkgName     = "app1pkg"
	app2PkgName     = "app2pkg"
)

var (
	QNameApp1_TestWSKind                     = appdef.NewQName(app1PkgName, "test_ws")
	QNameTestView                            = appdef.NewQName(app1PkgName, "View")
	QNameApp1_TestEmailVerificationDoc       = appdef.NewQName(app1PkgName, "Doc")
	QNameApp1_DocConstraints                 = appdef.NewQName(app1PkgName, "DocConstraints")
	QNameApp1_DocConstraintsString           = appdef.NewQName(app1PkgName, "DocConstraintsString")
	QNameApp1_DocConstraintsFewUniques       = appdef.NewQName(app1PkgName, "DocConstraintsFewUniques")
	QNameApp1_DocConstraintsOldAndNewUniques = appdef.NewQName(app1PkgName, "DocConstraintsOldAndNewUniques")
	QNameCmdRated                            = appdef.NewQName(app1PkgName, "RatedCmd")
	QNameQryRated                            = appdef.NewQName(app1PkgName, "RatedQry")
	QNameODoc1                               = appdef.NewQName(app1PkgName, "odoc1")
	QNameODoc2                               = appdef.NewQName(app1PkgName, "odoc2")
	TestSMTPCfg                              = smtp.Cfg{
		Username: "username@gmail.com",
	}

	// BLOBMaxSize 5
	SharedConfig_App1 = NewSharedVITConfig(
		WithApp(istructs.AppQName_test1_app1, ProvideApp1,
			WithWorkspaceTemplate(QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
			WithUserLogin("login", "pwd"),
			WithUserLogin(TestEmail, "1"),
			WithUserLogin(TestEmail2, "1"),
			WithChildWorkspace(QNameApp1_TestWSKind, "test_ws", "test_template", "", "login", map[string]interface{}{"IntFld": 42},
				WithChild(QNameApp1_TestWSKind, "test_ws2", "test_template", "", "login", map[string]interface{}{"IntFld": 42},
					WithSubject(TestEmail, istructs.SubjectKind_User, []appdef.QName{iauthnz.QNameRoleWorkspaceOwner}))),
		),
		WithApp(istructs.AppQName_test1_app2, ProvideApp2, WithUserLogin("login", "1")),
		WithVVMConfig(func(cfg *vvm.VVMConfig) {
			// for impl_reverseproxy_test
			cfg.Routes["/grafana"] = fmt.Sprintf("http://127.0.0.1:%d", TestServicePort)
			cfg.RoutesRewrite["/grafana-rewrite"] = fmt.Sprintf("http://127.0.0.1:%d/rewritten", TestServicePort)
			cfg.RouteDefault = fmt.Sprintf("http://127.0.0.1:%d/not-found", TestServicePort)
			cfg.RouteDomains["localhost"] = fmt.Sprintf("http://127.0.0.1:%d", TestServicePort)

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

func ProvideApp2(apis apps.APIs, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) apps.AppPackages {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("no build info")
	}
	sysPackageFS := sys.Provide(cfg, adf, TestSMTPCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
		apis.NumCommandProcessors, buildInfo, apis.IAppStorageProvider)
	app2PackageFS := parser.PackageFS{
		QualifiedPackageName: "github.com/voedger/voedger/pkg/vit/app2pkg",
		FS:                   SchemaTestApp2FS,
	}
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app2PkgName, "testCmd"), istructsmem.NullCommandExec))
	return apps.AppPackages{
		AppQName: istructs.AppQName_test1_app2,
		Packages: []parser.PackageFS{sysPackageFS, app2PackageFS},
	}
}

func ProvideApp1(apis apps.APIs, cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) apps.AppPackages {
	// sys package
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("no build info")
	}
	sysPackageFS := sys.Provide(cfg, adf, TestSMTPCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
		apis.NumCommandProcessors, buildInfo, apis.IAppStorageProvider)

	// for rates test
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		QNameQryRated,
		istructsmem.NullQueryExec,
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCmdRated,
		istructsmem.NullCommandExec,
	))

	// per-app limits
	cfg.FunctionRateLimits.AddAppLimit(QNameCmdRated, maxRateLimit2PerMinute)
	cfg.FunctionRateLimits.AddAppLimit(QNameQryRated, maxRateLimit2PerMinute)

	// per-workspace limits
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameCmdRated, maxRateLimit4PerHour)
	cfg.FunctionRateLimits.AddWorkspaceLimit(QNameQryRated, maxRateLimit4PerHour)

	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(app1PkgName, "MockQry"),
		func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockQryExec(input, callback)
		},
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(app1PkgName, "MockCmd"),
		func(args istructs.ExecCommandArgs) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockCmdExec(input)
		},
	))

	testCmdResult := appdef.NewQName(app1PkgName, "TestCmdResult")
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(app1PkgName, "TestCmd"),
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
		appdef.NewQName(app1PkgName, "CmdODocOne"),
		istructsmem.NullCommandExec,
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(app1PkgName, "CmdODocTwo"),
		istructsmem.NullCommandExec,
	))

	cfg.AddAsyncProjectors(func(istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name: appdef.NewQName(app1PkgName, "ProjDummy"),
			Func: func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
				return nil
			},
		}
	})

	app1PackageFS := parser.PackageFS{
		QualifiedPackageName: "github.com/voedger/voedger/pkg/vit/app1pkg",
		FS:                   SchemaTestApp1FS,
	}
	return apps.AppPackages{
		AppQName: istructs.AppQName_test1_app1,
		Packages: []parser.PackageFS{sysPackageFS, app1PackageFS},
	}
}
