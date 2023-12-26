/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package apps

import (
	"embed"

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
)

type APIs struct {
	itokens.ITokens
	istructs.IAppStructsProvider
	istructsmem.AppConfigsType
	istorage.IAppStorageProvider
	payloads.IAppTokensFactory
	coreutils.IFederation
	coreutils.TimeFunc
	NumCommandProcessors coreutils.CommandProcessorsCount
	appparts.IAppPartitions
}

type AppBuilder func(apis APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) AppPackages
type SchemasExportedContent map[string]map[string][]byte // packageName->schemaFilePath->content
// type PackageDesc struct {
// 	FQN string
// 	FS  embed.FS
// }
type AppPackages struct {
	AppQName istructs.AppQName
	Packages []parser.PackageFS
}
