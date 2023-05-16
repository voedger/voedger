/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func (hap VVMAppsBuilder) Add(appQName istructs.AppQName, builder VVMAppBuilder) {
	builders := hap[appQName]
	builders = append(builders, builder)
	hap[appQName] = builders
}

func (hap VVMAppsBuilder) PrepareStandardExtensionPoints() map[istructs.AppQName]IStandardExtensionPoints {
	seps := map[istructs.AppQName]IStandardExtensionPoints{}
	for appQName := range hap {
		seps[appQName] = &standardExtensionPointsImpl{rootExtensionPoint: &implIExtensionPoint{}}
	}
	return seps
}

func (hap VVMAppsBuilder) Build(vvmCfg *VVMConfig, cfgs istructsmem.AppConfigsType, vvmAPI VVMAPI, seps map[istructs.AppQName]IStandardExtensionPoints) (vvmApps VVMApps) {
	for appQName, builders := range hap {
		adf := appdef.New()
		sep := seps[appQName]
		cfg := cfgs.AddConfig(appQName, adf)
		for _, builder := range builders {
			builder(vvmCfg, vvmAPI, cfg, adf, sep)
		}
		epPostDocs := sep.ExtensionPoint(EPPostDocs)
		epPostDocs.Iterate(func(eKey EKey, value interface{}) {
			epPostDoc := value.(IExtensionPoint)
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

			epPostDoc.Iterate(func(eKey EKey, value interface{}) {
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

func (ar *standardExtensionPointsImpl) ExtensionPoint(epKey EPKey) IExtensionPoint {
	return ar.rootExtensionPoint.ExtensionPoint(epKey)
}
func (ar *standardExtensionPointsImpl) EPWSTemplates() IEPWSTemplates {
	return ar.rootExtensionPoint.ExtensionPoint(EPWSTemplates)
}
func (ar *standardExtensionPointsImpl) EPJournalIndices() IEPJournalIndices {
	return ar.rootExtensionPoint.ExtensionPoint(EPJournalIndices)
}
func (ar *standardExtensionPointsImpl) EPJournalPredicates() IEPJournalPredicates {
	return ar.rootExtensionPoint.ExtensionPoint(EPJournalPredicates)
}
