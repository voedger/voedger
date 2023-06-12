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

func (hap VVMAppsBuilder) Build(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (vvmApps VVMApps) {
	for appQName, builders := range hap {
		adf := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adf)
		for _, builder := range builders {
			builder(apis, cfg, adf, appEPs)
		}
		epPostDocs := appEPs.ExtensionPoint(apps.EPPostDocs)
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
		vvmApps = append(vvmApps, appQName)
		// TODO: remove it after https://github.com/voedger/voedger/issues/56
		if _, err := adf.Build(); err != nil {
			panic(err)
		}
	}
	return vvmApps
}
