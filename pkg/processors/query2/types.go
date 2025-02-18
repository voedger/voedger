/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
)

type ApiPath int

type QueryParams struct {
	Constraints *Constraints
	Argument    map[string]interface{}
}

type Constraints struct {
	Order   []string
	Limit   int
	Skip    int
	Include []string
	Keys    []string
	Where   map[string]interface{}
}

type IQueryMessage interface {
	AppQName() appdef.AppQName
	WSID() istructs.WSID
	Responder() bus.IResponder
	QueryParams() QueryParams
	DocID() istructs.IDType
	ApiPath() ApiPath
	RequestCtx() context.Context
	QName() appdef.QName
	PartitionID() istructs.PartitionID
	Host() string
	Token() string
}

type IApiPathHandler interface {
	CheckRateLimit(ctx context.Context, qw *queryWork) error
	CheckType(ctx context.Context, qw *queryWork) error
	ResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error
	AuthorizeRequest(ctx context.Context, qw *queryWork) error
	AuthorizeResult(ctx context.Context, qw *queryWork) error
	RowsProcessor(ctx context.Context, qw *queryWork) error
	Exec(ctx context.Context, qw *queryWork) error
}

type implIQueryMessage struct {
	appQName    appdef.AppQName
	wsid        istructs.WSID
	responder   bus.IResponder
	queryParams QueryParams
	docID       istructs.IDType
	apiPath     ApiPath
	requestCtx  context.Context
	qName       appdef.QName
	partition   istructs.PartitionID
	host        string
	token       string
}

func (qm *implIQueryMessage) AppQName() appdef.AppQName {
	return qm.appQName
}

func (qm *implIQueryMessage) WSID() istructs.WSID {
	return qm.wsid
}

func (qm *implIQueryMessage) Responder() bus.IResponder {
	return qm.responder
}

func (qm *implIQueryMessage) QueryParams() QueryParams {
	return qm.queryParams
}
func (qm *implIQueryMessage) DocID() istructs.IDType {
	return qm.docID
}

func (qm *implIQueryMessage) ApiPath() ApiPath {
	return qm.apiPath
}

func (qm *implIQueryMessage) RequestCtx() context.Context {
	return qm.requestCtx
}
func (qm *implIQueryMessage) QName() appdef.QName {
	return qm.qName
}

func (qm *implIQueryMessage) PartitionID() istructs.PartitionID {
	return qm.partition
}

func (qm *implIQueryMessage) Host() string {
	return qm.host
}

func (qm *implIQueryMessage) Token() string {
	return qm.token
}

type objectBackedByMap struct {
	istructs.IObject
	data map[string]interface{}
}

func (o objectBackedByMap) Release()                                { /*Do nothing*/ }
func (o objectBackedByMap) AsInt32(name appdef.FieldName) int32     { return o.data[name].(int32) }
func (o objectBackedByMap) AsInt64(name appdef.FieldName) int64     { return o.data[name].(int64) }
func (o objectBackedByMap) AsFloat32(name appdef.FieldName) float32 { return o.data[name].(float32) }
func (o objectBackedByMap) AsFloat64(name appdef.FieldName) float64 { return o.data[name].(float64) }
func (o objectBackedByMap) AsBytes(name appdef.FieldName) []byte    { return o.data[name].([]byte) }
func (o objectBackedByMap) AsString(name appdef.FieldName) string   { return o.data[name].(string) }
func (o objectBackedByMap) AsQName(name appdef.FieldName) appdef.QName {
	return o.data[name].(appdef.QName)
}
func (o objectBackedByMap) AsBool(name appdef.FieldName) bool { return o.data[name].(bool) }
func (o objectBackedByMap) AsRecordID(name appdef.FieldName) istructs.RecordID {
	return o.data[name].(istructs.RecordID)
}

type keys struct {
	pipeline.AsyncNOOP
	keys map[string]bool
}

func newKeys(ss []string) (o pipeline.IAsyncOperator) {
	k := &keys{keys: make(map[string]bool)}
	for _, s := range ss {
		k.keys[s] = true
	}
	return k
}

func (f *keys) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	for k := range work.(objectBackedByMap).data {
		if f.keys[k] {
			continue
		}
		delete(work.(objectBackedByMap).data, k)
	}
	return work, nil
}

type aggregator struct {
	pipeline.AsyncNOOP
	params QueryParams
	ww     []pipeline.IWorkpiece
}

func newAggregator(params QueryParams) pipeline.IAsyncOperator {
	return &aggregator{
		params: params,
		ww:     make([]pipeline.IWorkpiece, 0),
	}
}

func (a *aggregator) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	a.ww = append(a.ww, work)
	return
}
func (a *aggregator) Flush(callback pipeline.OpFuncFlush) (err error) {
	err = a.order()
	if err != nil {
		return
	}
	a.subList()
	for _, w := range a.ww {
		callback(w)
	}
	return
}
func (a *aggregator) order() (err error) {
	if len(a.params.Constraints.Order) == 0 {
		return
	}
	sort.Slice(a.ww, func(i, j int) bool {
		for _, orderBy := range a.params.Constraints.Order {
			vi := a.ww[i].(objectBackedByMap).data[orderBy]
			vj := a.ww[j].(objectBackedByMap).data[orderBy]
			switch vi.(type) {
			case int32:
				return compare(vi.(int32), vj.(int32))
			case int64:
				return compare(vi.(int64), vj.(int64))
			case float32:
				return compare(vi.(float32), vj.(float32))
			case float64:
				return compare(vi.(float64), vj.(float64))
			case []byte:
				return compare(string(vi.([]byte)), string(vi.([]byte)))
			case string:
				return compare(vi.(string), vj.(string))
			case appdef.QName:
				return compare(vi.(appdef.QName).String(), vj.(appdef.QName).String())
			case bool:
				return vi.(bool) != vj.(bool)
			case istructs.RecordID:
				return compare(uint64(vi.(istructs.RecordID)), uint64(vj.(istructs.RecordID)))
			default:
				err = errors.New("unsupported type")
			}
		}
		return false
	})
	return
}
func (a *aggregator) subList() {
	if a.params.Constraints.Limit == 0 && a.params.Constraints.Skip == 0 {
		return
	}
	if a.params.Constraints.Skip >= len(a.ww) {
		a.ww = []pipeline.IWorkpiece{}
		return
	}
	if a.params.Constraints.Skip+a.params.Constraints.Limit > len(a.ww) {
		a.ww = a.ww[a.params.Constraints.Skip:]
		return
	}
	a.ww = a.ww[a.params.Constraints.Skip:a.params.Constraints.Limit]
	return
}

type sender struct {
	pipeline.AsyncNOOP
	responder bus.IResponder
	sender    bus.IResponseSender
}

func (s *sender) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	if s.sender == nil {
		s.sender = s.responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
	}
	return work, s.sender.Send(work.(objectBackedByMap).data)
}
