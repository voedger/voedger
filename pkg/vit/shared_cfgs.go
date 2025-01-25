/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"

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

	app1PkgName = "app1pkg"
	App1PkgPath = "github.com/voedger/voedger/pkg/vit/app1pkg"

	app2PkgName = "app2pkg"
	app2PkgPath = "github.com/voedger/voedger/pkg/vit/app2pkg"
)

var (
	QNameApp1_TestWSKind                     = appdef.NewQName(app1PkgName, "test_ws")
	QNameApp1_TestWSKind_another             = appdef.NewQName(app1PkgName, "test_ws_another")
	QNameTestView                            = appdef.NewQName(app1PkgName, "View")
	QNameApp1_TestEmailVerificationDoc       = appdef.NewQName(app1PkgName, "Doc")
	QNameApp1_DocConstraints                 = appdef.NewQName(app1PkgName, "DocConstraints")
	QNameApp1_DocConstraintsString           = appdef.NewQName(app1PkgName, "DocConstraintsString")
	QNameApp1_DocConstraintsFewUniques       = appdef.NewQName(app1PkgName, "DocConstraintsFewUniques")
	QNameApp1_DocConstraintsOldAndNewUniques = appdef.NewQName(app1PkgName, "DocConstraintsOldAndNewUniques")
	QNameApp1_CDocCategory                   = appdef.NewQName(app1PkgName, "category")
	QNameCmdRated                            = appdef.NewQName(app1PkgName, "RatedCmd")
	QNameQryRated                            = appdef.NewQName(app1PkgName, "RatedQry")
	QNameODoc1                               = appdef.NewQName(app1PkgName, "odoc1")
	QNameODoc2                               = appdef.NewQName(app1PkgName, "odoc2")
	TestSMTPCfg                              = smtp.Cfg{
		Host:     "smtp.testserver.com",
		Port:     1,
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
			WithChildWorkspace(QNameApp1_TestWSKind_another, "test_ws_another", "", "", "login", map[string]interface{}{}),
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

			cfg.SmtpConfig = TestSMTPCfg
		}),
		WithCleanup(func(_ *VIT) {
			MockCmdExec = func(input string, args istructs.ExecCommandArgs) error { panic("") }
			MockQryExec = func(input string, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error { panic("") }
		}),
	)
	MockQryExec func(input string, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error
	MockCmdExec func(input string, args istructs.ExecCommandArgs) error
)

func ProvideApp2(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
	sysPackageFS := sysprovide.Provide(cfg)
	app2PackageFS := parser.PackageFS{
		Path: app2PkgPath,
		FS:   SchemaTestApp2FS,
	}
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app2PkgName, "testCmd"), istructsmem.NullCommandExec))
	ep.AddNamed(builtinapps.EPIsDeviceAllowedFunc, func(as istructs.IAppStructs, requestWSID istructs.WSID, deviceProfileWSID istructs.WSID) (ok bool, err error) {
		// simulate we could not work in any non-profile WS
		return false, err
	})
	return builtinapps.Def{
		AppQName:                istructs.AppQName_test1_app2,
		Packages:                []parser.PackageFS{sysPackageFS, app2PackageFS},
		AppDeploymentDescriptor: TestAppDeploymentDescriptor,
	}
}

func ProvideApp2WithJob(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
	sysPackageFS := sysprovide.Provide(cfg)
	app2PackageFS := parser.PackageFS{
		Path: app2PkgPath,
		FS:   SchemaTestApp2WithJobFS,
	}
	ep.AddNamed(builtinapps.EPIsDeviceAllowedFunc, func(as istructs.IAppStructs, requestWSID istructs.WSID, deviceProfileWSID istructs.WSID) (ok bool, err error) {
		// simulate we could not work in any non-profile WS
		return false, err
	})
	cfg.AddJobs(istructsmem.BuiltinJob{
		Name: appdef.NewQName(app2PkgName, "Job1_builtin"),
		Func: func(st istructs.IState, intents istructs.IIntents) error {
			jobsQName := appdef.NewQName("app2pkg", "Jobs")
			kb, err := st.KeyBuilder(sys.Storage_View, jobsQName)
			if err != nil {
				return err
			}
			kb.PutInt64("RunUnixMilli", apis.ITime.Now().UnixMilli())
			kb.PutInt32("Dummy1", 1)
			vb, err := intents.NewValue(kb)
			if err != nil {
				return err
			}
			vb.PutInt32("Dummy2", 1)
			return nil
		},
	})
	return builtinapps.Def{
		AppQName:                istructs.AppQName_test1_app2,
		Packages:                []parser.PackageFS{sysPackageFS, app2PackageFS},
		AppDeploymentDescriptor: TestAppDeploymentDescriptor,
	}
}

func ProvideApp1(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
	// sys package
	sysPackageFS := sysprovide.Provide(cfg)

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
			return MockQryExec(input, args, callback)
		},
	))

	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(app1PkgName, "MockCmd"),
		func(args istructs.ExecCommandArgs) (err error) {
			input := args.ArgumentObject.AsString(field_Input)
			return MockCmdExec(input, args)
		},
	))

	testCmdResult := appdef.NewQName(app1PkgName, "TestCmdResult")
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		appdef.NewQName(app1PkgName, "TestCmd"),
		func(args istructs.ExecCommandArgs) (err error) {
			key, err := args.State.KeyBuilder(sys.Storage_Result, testCmdResult)
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

	cfg.AddAsyncProjectors(
		istructs.Projector{
			Name: appdef.NewQName(app1PkgName, "ProjDummy"),
			Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) (err error) { return nil },
		},
	)

	qNameViewCategoryIdx := appdef.NewQName(app1PkgName, "CategoryIdx")
	cfg.AddSyncProjectors(
		istructs.Projector{
			Name: appdef.NewQName(app1PkgName, "ApplyCategoryIdx"),
			Func: func(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) error {
				for cud := range event.CUDs {
					if cud.QName() != QNameApp1_CDocCategory {
						continue
					}
					kb, err := st.KeyBuilder(sys.Storage_View, qNameViewCategoryIdx)
					if err != nil {
						return err
					}
					kb.PutInt32("IntFld", 43)
					kb.PutInt32("Dummy", 1)
					b, err := intents.NewValue(kb)
					if err != nil {
						return err
					}
					b.PutInt32("Val", 42)
					b.PutString("Name", cud.AsString("name"))
					b.PutInt64(state.ColOffset, int64(event.WLogOffset())) // nolint G115
				}
				return nil
			},
		},
	)

	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app1PkgName, "testCmd"), istructsmem.NullCommandExec))
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app1PkgName, "TestCmdRawArg"), istructsmem.NullCommandExec))
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app1PkgName, "TestDeniedCmd"), istructsmem.NullCommandExec))
	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(app1PkgName, "TestDeniedQuery"), istructsmem.NullQueryExec))

	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(app1PkgName, "QryIntents"), func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		kb, err := args.State.KeyBuilder(sys.Storage_Result, appdef.NewQName(app1PkgName, "QryIntentsResult"))
		if err != nil {
			return err
		}
		vb, err := args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		vb.PutString("Fld1", "hello")
		return nil
	}))

	funcWithResponseIntents := func(args istructs.PrepareArgs, st istructs.IState, intents istructs.IIntents) (err error) {
		kb, err := st.KeyBuilder(sys.Storage_Response, appdef.NullQName)
		if err != nil {
			return err
		}
		vb, err := intents.NewValue(kb)
		if err != nil {
			return err
		}
		vb.PutInt32(sys.Storage_Response_Field_StatusCode, args.ArgumentObject.AsInt32("StatusCodeToReturn"))
		vb.PutString(sys.Storage_Response_Field_ErrorMessage, "error from response intent")
		return nil
	}

	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(app1PkgName, "CmdWithResponseIntent"), func(args istructs.ExecCommandArgs) (err error) {
		return funcWithResponseIntents(args.PrepareArgs, args.State, args.Intents)
	}))

	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(app1PkgName, "QryWithResponseIntent"), func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		return funcWithResponseIntents(args.PrepareArgs, args.State, args.Intents)
	}))

	app1PackageFS := parser.PackageFS{
		Path: App1PkgPath,
		FS:   SchemaTestApp1FS,
	}
	return builtinapps.Def{
		AppQName:                istructs.AppQName_test1_app1,
		Packages:                []parser.PackageFS{sysPackageFS, app1PackageFS},
		AppDeploymentDescriptor: TestAppDeploymentDescriptor,
	}
}
