/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/parser"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

type APIs struct {
	itokens.ITokens
	istructs.IAppStructsProvider
	istorage.IAppStorageProvider
	payloads.IAppTokensFactory
	federation.IFederation
	coreutils.TimeFunc
	SidecarApps []appparts.SidecarApp
	// IAppPartitions - wrong, wire cycle: `appparts.NewWithActualizerWithExtEnginesFactories(asp, actualizer, eef) IAppPartitions`` accepts engines.ProvideExtEngineFactories()
	//                                     that requires filled AppConfigsType, but AppConfigsType requires apps.APIs with IAppPartitions
}

type AppBuilder func(apis APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) BuiltInAppDef
type SchemasExportedContent map[string]map[string][]byte // packageName->schemaFilePath->content
type CLIParams struct {
	Storage string
}
type BuiltInAppDef struct {
	appparts.AppDeploymentDescriptor
	AppQName appdef.AppQName
	Packages []parser.PackageFS
}
