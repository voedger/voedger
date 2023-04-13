/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * * @author Michael Saigachenko
 */

package queryprocessor

import (
	"context"

	iprocbus "github.com/heeus/core-iprocbus"
	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/istructsmem"
	imetrics "github.com/untillpro/voedger/pkg/metrics"
	"github.com/untillpro/voedger/pkg/pipeline"
)

// RowsProcessorFactory is the function for building pipeline from query params and row meta
// The pipeline is used to process data fetched by QueryHandler
// TODO In my opinion we have to remove it from export
type RowsProcessorFactory func(ctx context.Context, schemas istructs.ISchemas, state istructs.IState,
	params IQueryParams, resultMeta istructs.ISchema, rs IResultSenderClosable, metrics IMetrics) pipeline.IAsyncPipeline

type ServiceFactory func(serviceChannel iprocbus.ServiceChannel, resultSenderClosableFactory ResultSenderClosableFactory,
	appStructsProvider istructs.IAppStructsProvider, maxPrepareQueries int, metrics imetrics.IMetrics, hvm string,
	authn iauthnz.IAuthenticator, authz iauthnz.IAuthorizer, appCfgs istructsmem.AppConfigsType) pipeline.IService
