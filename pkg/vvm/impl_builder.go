/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func (hap VVMAppsBuilder) Add(appQName istructs.AppQName, builder apps.AppBuilder) {
	builders := hap[appQName]
	builders = append(builders, builder)
	hap[appQName] = builders
}

func (hap VVMAppsBuilder) PrepareAppsExtensionPoints() map[istructs.AppQName]extensionpoints.IExtensionPoint {
	seps := map[istructs.AppQName]extensionpoints.IExtensionPoint{}
	for appQName := range hap {
		seps[appQName] = extensionpoints.NewRootExtensionPoint()
	}
	return seps
}

func buildSchemasASTs(adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	epPackageSchemasASTs := ep.ExtensionPoint(apps.EPPackageSchemasASTs)
	packageSchemaASTs := []*parser.PackageSchemaAST{}
	epPackageSchemasASTs.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
		qualifiedPackageName := eKey.(string)
		packageFilesSchemasASTsEP := value.(extensionpoints.IExtensionPoint)
		packageFilesSchemasASTs := []*parser.FileSchemaAST{}
		packageFilesSchemasASTsEP.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
			fileSchemaAST := value.(*parser.FileSchemaAST)
			packageFilesSchemasASTs = append(packageFilesSchemasASTs, fileSchemaAST)
		})
		packageSchemaAST, err := parser.MergeFileSchemaASTs(qualifiedPackageName, packageFilesSchemasASTs)
		if err != nil {
			panic(err)
		}
		packageSchemaASTs = append(packageSchemaASTs, packageSchemaAST)
	})
	packageSchemas, err := parser.MergePackageSchemas(packageSchemaASTs)
	if err != nil {
		panic(err)
	}
	if err := parser.BuildAppDefs(packageSchemas, adf); err != nil {
		panic(err)
	}
}

func buildPostDocs(adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	epPostDocs := ep.ExtensionPoint(apps.EPPostDocs)
	epPostDocs.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
		epPostDoc := value.(extensionpoints.IExtensionPoint)
		postDocQName := eKey.(appdef.QName)
		postDocDesc := epPostDoc.Value().(PostDocDesc)

		var doc appdef.IFieldsBuilder
		switch postDocDesc.Kind {
		case appdef.DefKind_GDoc:
			doc = adf.AddGDoc(postDocQName)
		case appdef.DefKind_CDoc:
			if postDocDesc.IsSingleton {
				doc = adf.AddSingleton(postDocQName)
			} else {
				doc = adf.AddCDoc(postDocQName)
			}
		case appdef.DefKind_WDoc:
			doc = adf.AddWDoc(postDocQName)
		case appdef.DefKind_ODoc:
			doc = adf.AddODoc(postDocQName)
		default:
			panic(fmt.Errorf("document «%s» has unexpected definition kind «%v»", postDocQName, postDocDesc.Kind))
		}

		epPostDoc.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
			postDocField := value.(PostDocFieldType)
			if len(postDocField.VerificationKinds) > 0 {
				doc.AddVerifiedField(eKey.(string), postDocField.Kind, postDocField.Required, postDocField.VerificationKinds...)
			} else {
				doc.AddField(eKey.(string), postDocField.Kind, postDocField.Required)
			}
		})
	})
}

func (hap VVMAppsBuilder) Build(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (vvmApps VVMApps) {
	for appQName, appBuilders := range hap {
		adf := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adf)
		for _, builder := range appBuilders {
			builder(apis, cfg, adf, appEPs)
		}
		buildSchemasASTs(adf, appEPs)
		buildPostDocs(adf, appEPs)
		vvmApps = append(vvmApps, appQName)
		if _, err := adf.Build(); err != nil {
			panic(err)
		}
	}
	return vvmApps
}
