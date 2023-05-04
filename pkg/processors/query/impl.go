/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * * @author Michael Saigachenko
 */

package queryprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func implRowsProcessorFactory(ctx context.Context, appDef appdef.IAppDef, state istructs.IState, params IQueryParams,
	resultMeta appdef.IDef, rs IResultSenderClosable, metrics IMetrics) pipeline.IAsyncPipeline {
	operators := make([]*pipeline.WiredOperator, 0)
	if resultMeta.QName() == istructs.QNameJSON {
		operators = append(operators, pipeline.WireAsyncOperator("Raw result", &RawResultOperator{
			metrics: metrics,
		}))
	} else {
		fieldsDefs := &fieldsDefs{
			appDef: appDef,
			fields: make(map[appdef.QName]coreutils.FieldsDef),
		}
		rootFields := coreutils.NewFieldsDef(resultMeta)
		operators = append(operators, pipeline.WireAsyncOperator("Result fields", &ResultFieldsOperator{
			elements:   params.Elements(),
			rootFields: rootFields,
			fieldsDefs: fieldsDefs,
			metrics:    metrics,
		}))
		operators = append(operators, pipeline.WireAsyncOperator("Enrichment", &EnrichmentOperator{
			state:      state,
			elements:   params.Elements(),
			fieldsDefs: fieldsDefs,
			metrics:    metrics,
		}))
		if len(params.Filters()) != 0 {
			operators = append(operators, pipeline.WireAsyncOperator("Filter", &FilterOperator{
				filters:    params.Filters(),
				rootFields: rootFields,
				metrics:    metrics,
			}))
		}
		if len(params.OrderBy()) != 0 {
			operators = append(operators, pipeline.WireAsyncOperator("Order", newOrderOperator(params.OrderBy(), metrics)))
		}
		if params.StartFrom() != 0 || params.Count() != 0 {
			operators = append(operators, pipeline.WireAsyncOperator("Counter", newCounterOperator(
				params.StartFrom(),
				params.Count(),
				metrics)))
		}
	}
	operators = append(operators, pipeline.WireAsyncOperator("Send to bus", &SendToBusOperator{
		rs:      rs,
		metrics: metrics,
	}))
	return pipeline.NewAsyncPipeline(ctx, "Rows processor", operators[0], operators[1:]...)
}

func implServiceFactory(serviceChannel iprocbus.ServiceChannel, resultSenderClosableFactory ResultSenderClosableFactory,
	appStructsProvider istructs.IAppStructsProvider, maxPrepareQueries int, metrics imetrics.IMetrics, hvm string,
	authn iauthnz.IAuthenticator, authz iauthnz.IAuthorizer, appCfgs istructsmem.AppConfigsType) pipeline.IService {
	secretReader := isecretsimpl.ProvideSecretReader()
	return pipeline.NewService(func(ctx context.Context) {
		var p pipeline.ISyncPipeline
		for ctx.Err() == nil {
			select {
			case intf := <-serviceChannel:
				now := time.Now()
				msg := intf.(IQueryMessage)
				qpm := &queryProcessorMetrics{
					hvm:     hvm,
					app:     msg.AppQName(),
					metrics: metrics,
				}
				qpm.Increase(queriesTotal, 1.0)
				rs := resultSenderClosableFactory(msg.RequestCtx(), msg.Sender())
				rs = &resultSenderClosableOnlyOnce{IResultSenderClosable: rs}
				qwork := newQueryWork(msg, rs, appStructsProvider, maxPrepareQueries, qpm, secretReader)
				if p == nil {
					p = newQueryProcessorPipeline(ctx, authn, authz, appCfgs)
				}
				err := p.SendSync(&qwork)
				if err != nil {
					qpm.Increase(errorsTotal, 1.0)
					p.Close()
					p = nil
				}
				rs.Close(err)
				qpm.Increase(queriesSeconds, time.Since(now).Seconds())
			case <-ctx.Done():
			}
		}
		if p != nil {
			p.Close()
		}
	})
}

func newQueryProcessorPipeline(requestCtx context.Context, authn iauthnz.IAuthenticator, authz iauthnz.IAuthorizer,
	appCfgs istructsmem.AppConfigsType) pipeline.ISyncPipeline {
	ops := []*pipeline.WiredOperator{
		operator("get app structs", func(ctx context.Context, qw *queryWork) (err error) {
			qw.appStructs, err = qw.appStructsProvider.AppStructs(qw.msg.AppQName())
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("check function call rate", func(ctx context.Context, qw *queryWork) (err error) {
			if qw.appStructs.IsFunctionRateLimitsExceeded(qw.msg.Resource().QName(), qw.msg.WSID()) {
				return coreutils.NewSysError(http.StatusTooManyRequests)
			}
			return nil
		}),
		operator("unmarshal JSON", func(ctx context.Context, qw *queryWork) (err error) {
			err = json.Unmarshal(qw.msg.Body(), &qw.requestData)
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("authenticate query request", func(ctx context.Context, qw *queryWork) (err error) {
			req := iauthnz.AuthnRequest{
				Host:        qw.msg.Host(),
				RequestWSID: qw.msg.WSID(),
				Token:       qw.msg.Token(),
			}
			if qw.principals, qw.principalPayload, err = authn.Authenticate(qw.msg.RequestCtx(), qw.appStructs, qw.appStructs.AppTokens(), req); err != nil {
				return coreutils.WrapSysError(err, http.StatusUnauthorized)
			}
			return
		}),
		operator("authorize query request", func(ctx context.Context, qw *queryWork) (err error) {
			req := iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_EXECUTE,
				Resource:      qw.msg.Resource().QName(),
			}
			ok, err := authz.Authorize(qw.appStructs, qw.principals, req)
			if err != nil {
				return err
			}
			if !ok {
				return coreutils.WrapSysError(errors.New(""), http.StatusForbidden)
			}
			return nil
		}),
		operator("get AppConfig", func(ctx context.Context, qw *queryWork) (err error) {
			cfg, ok := appCfgs[qw.msg.AppQName()]
			if !ok {
				return errors.New("failed to get AppConfig")
			}
			qw.appCfg = cfg
			return nil
		}),
		operator("validate: get exec query args", func(ctx context.Context, qw *queryWork) (err error) {
			qw.queryFunction = qw.msg.Resource().(istructs.IQueryFunction)
			parDef := qw.appStructs.AppDef().Def(qw.queryFunction.ParamsDef())
			qw.execQueryArgs, err = newExecQueryArgs(qw.requestData, qw.msg.WSID(), parDef, qw.appCfg, qw)
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("create state", func(ctx context.Context, qw *queryWork) (err error) {
			qw.state = state.ProvideQueryProcessorStateFactory()(
				qw.msg.RequestCtx(),
				qw.appStructs,
				state.SimplePartitionIDFunc(qw.msg.Partition()),
				state.SimpleWSIDFunc(qw.msg.WSID()),
				qw.secretReader,
				func() []iauthnz.Principal { return qw.principals },
				func() string { return qw.msg.Token() })
			qw.execQueryArgs.State = qw.state
			return
		}),
		operator("validate: get result definition", func(ctx context.Context, qw *queryWork) (err error) {
			def := qw.queryFunction.ResultDef(qw.execQueryArgs.PrepareArgs)
			qw.resultDef = qw.appStructs.AppDef().Def(def)
			err = errIfFalse(qw.resultDef.Kind() != appdef.DefKind_null, func() error {
				return fmt.Errorf("result definition %s: %w", def, ErrNotFound)
			})
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("validate: get query params", func(ctx context.Context, qw *queryWork) (err error) {
			qw.queryParams, err = newQueryParams(qw.requestData, NewElement, NewFilter, NewOrderBy, coreutils.NewFieldsDef(qw.resultDef))
			return coreutils.WrapSysError(err, http.StatusBadRequest)
		}),
		operator("authorize result", func(ctx context.Context, qw *queryWork) (err error) {
			req := iauthnz.AuthzRequest{
				OperationKind: iauthnz.OperationKind_SELECT,
				Resource:      qw.msg.Resource().QName(),
			}
			for _, elem := range qw.queryParams.Elements() {
				for _, resultField := range elem.ResultFields() {
					req.Fields = append(req.Fields, resultField.Field())
				}
			}
			if len(req.Fields) == 0 {
				return nil
			}
			ok, err := authz.Authorize(qw.appStructs, qw.principals, req)
			if err != nil {
				return err
			}
			if !ok {
				return coreutils.NewSysError(http.StatusForbidden)
			}
			return nil
		}),
		operator("build rows processor", func(ctx context.Context, qw *queryWork) error {
			now := time.Now()
			defer func() {
				qw.metrics.Increase(buildSeconds, time.Since(now).Seconds())
			}()
			qw.rowsProcessor = ProvideRowsProcessorFactory()(qw.msg.RequestCtx(), qw.appStructs.AppDef(),
				qw.state, qw.queryParams, qw.resultDef, qw.rs, qw.metrics)
			return nil
		}),
		operator("exec function", func(ctx context.Context, qw *queryWork) (err error) {
			now := time.Now()
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())
					err = fmt.Errorf("%v\n%s", r, stack)
				}
				qw.rowsProcessor.Close()
				qw.metrics.Increase(execSeconds, time.Since(now).Seconds())
			}()
			err = qw.queryFunction.Exec(ctx, qw.execQueryArgs, func(object istructs.IObject) error {
				pathToIdx := make(map[string]int)
				if qw.resultDef.QName() == istructs.QNameJSON {
					pathToIdx[Field_JSONDef_Body] = 0
				} else {
					for i, element := range qw.queryParams.Elements() {
						pathToIdx[element.Path().Name()] = i
					}
				}
				return qw.rowsProcessor.SendAsync(workpiece{
					object: object,
					outputRow: &outputRow{
						keyToIdx: pathToIdx,
						values:   make([]interface{}, len(pathToIdx)),
					},
					enrichedRootFields: make(map[string]appdef.DataKind),
				})
			})
			return coreutils.WrapSysError(err, http.StatusInternalServerError)
		}),
	}
	return pipeline.NewSyncPipeline(requestCtx, "Query Processor", ops[0], ops[1:]...)
}

type queryWork struct {
	// input
	msg                IQueryMessage
	rs                 IResultSenderClosable
	appStructsProvider istructs.IAppStructsProvider
	// work
	requestData       map[string]interface{}
	state             istructs.IState
	queryParams       IQueryParams
	appStructs        istructs.IAppStructs
	queryFunction     istructs.IQueryFunction
	resultDef         appdef.IDef
	execQueryArgs     istructs.ExecQueryArgs
	maxPrepareQueries int
	rowsProcessor     pipeline.IAsyncPipeline
	metrics           IMetrics
	principals        []iauthnz.Principal
	principalPayload  payloads.PrincipalPayload
	secretReader      isecrets.ISecretReader
	appCfg            *istructsmem.AppConfigType
}

func newQueryWork(msg IQueryMessage, rs IResultSenderClosable, appStructsProvider istructs.IAppStructsProvider,
	maxPrepareQueries int, metrics *queryProcessorMetrics, secretReader isecrets.ISecretReader) queryWork {
	return queryWork{
		msg:                msg,
		rs:                 rs,
		appStructsProvider: appStructsProvider,
		requestData:        make(map[string]interface{}),
		maxPrepareQueries:  maxPrepareQueries,
		metrics:            metrics,
		secretReader:       secretReader,
	}
}

// need for q.sys.EnrichPrincipalToken
// failed to implement this via stroage because payloads.PrincipalPayload is too complex structrue -> need to get via .AsJSON() only
// but it is bad idea to parse json in an extension, so just let q.sys.EnrichPrincipalToken work using this hidden func
func (qw *queryWork) GetPrincipalPayload() payloads.PrincipalPayload {
	return qw.principalPayload
}

// need for q.sys.EnrichPrincipalToken
func (qw *queryWork) GetPrincipals() []iauthnz.Principal {
	return qw.principals
}

func operator(name string, doSync func(ctx context.Context, qw *queryWork) (err error)) *pipeline.WiredOperator {
	return pipeline.WireFunc(name, func(ctx context.Context, work interface{}) (err error) {
		return doSync(ctx, work.(*queryWork))
	})
}

func errIfFalse(cond bool, errIfFalse func() error) error {
	if !cond {
		return errIfFalse()
	}
	return nil
}

type queryMessage struct {
	requestCtx context.Context
	appQName   istructs.AppQName
	wsid       istructs.WSID
	partition  istructs.PartitionID
	sender     interface{}
	body       []byte
	resource   istructs.IResource
	host       string
	token      string
}

func (m queryMessage) AppQName() istructs.AppQName     { return m.appQName }
func (m queryMessage) WSID() istructs.WSID             { return m.wsid }
func (m queryMessage) Sender() interface{}             { return m.sender }
func (m queryMessage) RequestCtx() context.Context     { return m.requestCtx }
func (m queryMessage) Resource() istructs.IResource    { return m.resource }
func (m queryMessage) Host() string                    { return m.host }
func (m queryMessage) Token() string                   { return m.token }
func (m queryMessage) Partition() istructs.PartitionID { return m.partition }
func (m queryMessage) Body() []byte {
	if len(m.body) != 0 {
		return m.body
	}
	return []byte("{}")
}

func NewQueryMessage(requestCtx context.Context, appQName istructs.AppQName, wsid istructs.WSID, sender interface{}, body []byte,
	resource istructs.IResource, host string, token string) IQueryMessage {
	return queryMessage{
		appQName:   appQName,
		wsid:       wsid,
		sender:     sender,
		body:       body,
		requestCtx: requestCtx,
		resource:   resource,
		host:       host,
		token:      token,
	}
}

type workpiece struct {
	pipeline.IWorkpiece
	object             istructs.IObject
	outputRow          IOutputRow
	enrichedRootFields coreutils.FieldsDef
}

func (w workpiece) Object() istructs.IObject                { return w.object }
func (w workpiece) OutputRow() IOutputRow                   { return w.outputRow }
func (w workpiece) EnrichedRootFields() coreutils.FieldsDef { return w.enrichedRootFields }
func (w workpiece) PutEnrichedRootField(name string, kind appdef.DataKind) {
	w.enrichedRootFields[name] = kind
}
func (w workpiece) Release() {
	//TODO implement it someday
	//Release goes here
}

type outputRow struct {
	keyToIdx map[string]int
	values   []interface{}
}

func (r *outputRow) Set(alias string, value interface{}) { r.values[r.keyToIdx[alias]] = value }
func (r *outputRow) Values() []interface{}               { return r.values }
func (r *outputRow) Value(alias string) interface{}      { return r.values[r.keyToIdx[alias]] }
func (r *outputRow) MarshalJSON() ([]byte, error)        { return json.Marshal(r.values) }

func newExecQueryArgs(data coreutils.MapObject, wsid istructs.WSID, argsDef appdef.IDef, appCfg *istructsmem.AppConfigType,
	qw *queryWork) (execQueryArgs istructs.ExecQueryArgs, err error) {
	args, _, err := data.AsObject("args")
	if err != nil {
		return execQueryArgs, err
	}
	requestArgs := istructs.NewNullObject()
	switch argsDef.QName() {
	case appdef.NullQName:
		//Do nothing
	case istructs.QNameJSON:
		requestArgs, err = newJsonObject(data)
	default:
		requestArgsBuilder := istructsmem.NewIObjectBuilder(appCfg, argsDef.QName())
		if err := istructsmem.FillElementFromJSON(args, argsDef, requestArgsBuilder); err != nil {
			return execQueryArgs, err
		}
		requestArgs, err = requestArgsBuilder.Build()
	}
	if err != nil {
		return execQueryArgs, err
	}
	return istructs.ExecQueryArgs{
		PrepareArgs: istructs.PrepareArgs{
			ArgumentObject: requestArgs,
			Workspace:      wsid,
			Workpiece:      qw,
		},
	}, nil
}

type path []string

func (p path) IsRoot() bool      { return p[0] == rootDocument }
func (p path) Name() string      { return strings.Join(p, "/") }
func (p path) AsArray() []string { return p }

type element struct {
	path   path
	fields []IResultField
	refs   []IRefField
}

func (e element) NewOutputRow() IOutputRow {
	fields := make([]string, 0)
	for _, field := range e.fields {
		fields = append(fields, field.Field())
	}
	for _, field := range e.refs {
		fields = append(fields, field.Key())
	}
	fieldToIdx := make(map[string]int)
	for j, field := range fields {
		fieldToIdx[field] = j
	}
	return &outputRow{
		keyToIdx: fieldToIdx,
		values:   make([]interface{}, len(fieldToIdx)),
	}
}

func (e element) Path() IPath                  { return e.path }
func (e element) ResultFields() []IResultField { return e.fields }
func (e element) RefFields() []IRefField       { return e.refs }

type fieldsDefs struct {
	appDef appdef.IAppDef
	fields map[appdef.QName]coreutils.FieldsDef
	lock   sync.Mutex
}

func newFieldsDefs(appDef appdef.IAppDef) *fieldsDefs {
	return &fieldsDefs{
		appDef: appDef,
		fields: make(map[appdef.QName]coreutils.FieldsDef),
	}
}

func (c *fieldsDefs) get(name appdef.QName) coreutils.FieldsDef {
	c.lock.Lock()
	defer c.lock.Unlock()
	fd, ok := c.fields[name]
	if !ok {
		fd = coreutils.NewFieldsDef(c.appDef.Def(name))
		c.fields[name] = fd
	}
	return fd
}

type resultSenderClosableOnlyOnce struct {
	IResultSenderClosable
	sync.Once
}

func (s *resultSenderClosableOnlyOnce) Close(err error) {
	s.Once.Do(func() {
		s.IResultSenderClosable.Close(err)
	})
}

type queryProcessorMetrics struct {
	hvm     string
	app     istructs.AppQName
	metrics imetrics.IMetrics
}

func (m *queryProcessorMetrics) Increase(metricName string, valueDelta float64) {
	m.metrics.IncreaseApp(metricName, m.hvm, m.app, valueDelta)
}

// need or q.sys.EnrichPrincipalToken
func (qw *queryWork) AppQName() istructs.AppQName {
	return qw.msg.AppQName()
}

type jsonObject struct {
	istructs.NullObject
	body []byte
}

func newJsonObject(data coreutils.MapObject) (object istructs.IObject, err error) {
	jo := &jsonObject{}
	jo.body, err = json.Marshal(data)
	return jo, err
}

func (o *jsonObject) AsString(name string) string {
	if name == Field_JSONDef_Body {
		return string(o.body)
	}
	return o.NullObject.AsString(name)
}
