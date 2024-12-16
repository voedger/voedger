/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package builtinapps

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
)

type Builder func(apis APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) Def

type Def struct {
	appparts.AppDeploymentDescriptor
	AppQName appdef.AppQName
	Packages []parser.PackageFS
}

type APIs struct {
	itokens.ITokens
	istructs.IAppStructsProvider
	istorage.IAppStorageProvider
	payloads.IAppTokensFactory
	federation.IFederation
	coreutils.ITime
	SidecarApps []appparts.SidecarApp
	// IAppPartitions - wrong, wire cycle: `appparts.NewWithActualizerWithExtEnginesFactories(asp, actualizer, eef) IAppPartitions`` accepts engines.ProvideExtEngineFactories()
	//                                     that requires filled AppConfigsType, but AppConfigsType requires apps.APIs with IAppPartitions
}

