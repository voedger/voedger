/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	istructs "github.com/untillpro/voedger/pkg/istructs"
	istructsmem "github.com/untillpro/voedger/pkg/istructsmem"
)

func ProvideAsyncActualizerFactory() AsyncActualizerFactory {
	return asyncActualizerFactory
}

func ProvideSyncActualizerFactory() SyncActualizerFactory {
	return syncActualizerFactory
}

func ProvideOffsetsSchema(cfg *istructsmem.AppConfigType) {
	provideOffsetsSchemaImpl(cfg)
}

func ProvideViewSchema(app *istructsmem.AppConfigType, qname istructs.QName, buildFunc BuildViewSchemaFunc) {
	provideViewSchemaImpl(app, qname, buildFunc)
}
