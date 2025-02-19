/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

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
	Where   Where
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
	params      QueryParams
	orderParams map[string]bool
	ww          []pipeline.IWorkpiece
}

func newAggregator(params QueryParams) pipeline.IAsyncOperator {
	a := &aggregator{
		params:      params,
		orderParams: make(map[string]bool),
		ww:          make([]pipeline.IWorkpiece, 0),
	}
	for _, s := range params.Constraints.Order {
		if strings.HasPrefix(s, "-") {
			a.orderParams[strings.ReplaceAll(s, "-", "")] = false
		} else {
			a.orderParams[s] = true
		}
	}
	return a
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
	if len(a.orderParams) == 0 {
		return
	}
	sort.Slice(a.ww, func(i, j int) bool {
		for orderBy, asc := range a.orderParams {
			vi := a.ww[i].(objectBackedByMap).data[orderBy]
			vj := a.ww[j].(objectBackedByMap).data[orderBy]
			switch vi.(type) {
			case int32:
				return a.compareInt32(vi.(int32), vj.(int32), asc)
			case int64:
				return a.compareInt64(vi.(int64), vj.(int64), asc)
			case float32:
				return a.compareFloat32(vi.(float32), vj.(float32), asc)
			case float64:
				return a.compareFloat64(vi.(float64), vj.(float64), asc)
			case []byte:
				return a.compareString(string(vi.([]byte)), string(vi.([]byte)), asc)
			case string:
				return a.compareString(vi.(string), vj.(string), asc)
			case appdef.QName:
				return a.compareString(vi.(appdef.QName).String(), vj.(appdef.QName).String(), asc)
			case bool:
				return vi.(bool) != vj.(bool)
			case istructs.RecordID:
				return a.compareUint64(uint64(vi.(istructs.RecordID)), uint64(vj.(istructs.RecordID)), asc)
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
func (a *aggregator) compareInt32(v1, v2 int32, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
}
func (a *aggregator) compareInt64(v1, v2 int64, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
}
func (a *aggregator) compareFloat32(v1, v2 float32, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
}
func (a *aggregator) compareFloat64(v1, v2 float64, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
}
func (a *aggregator) compareString(v1, v2 string, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
}
func (a *aggregator) compareUint64(v1, v2 uint64, asc bool) bool {
	if asc {
		return v1 < v2
	}
	return v1 > v2
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

type filter struct {
	pipeline.AsyncNOOP
	Int32  map[string]map[int32]bool
	String map[string]map[string]bool
}

func newFilter(qw *queryWork) (o pipeline.IAsyncOperator, err error) {
	f := &filter{
		Int32:  make(map[string]map[int32]bool),
		String: make(map[string]map[string]bool),
	}
	fields := make([]appdef.IField, 0)
	fields = append(fields, qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Key().ClustCols().Fields()...)
	fields = append(fields, qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Value().Fields()...)
	for _, field := range qw.appStructs.AppDef().Type(qw.iView.QName()).(appdef.IView).Fields() {
		switch field.DataKind() {
		case appdef.DataKind_int32:
			vv, err := qw.queryParams.Constraints.Where.getAsInt32(field.Name())
			if err != nil {
				return nil, err
			}
			for _, v := range vv {
				m, ok := f.Int32[field.Name()]
				if !ok {
					m = make(map[int32]bool)
					f.Int32[field.Name()] = m
				}
				m[v] = true
			}
		case appdef.DataKind_string:
			vv, err := qw.queryParams.Constraints.Where.getAsString(field.Name())
			if err != nil {
				return nil, err
			}
			for _, v := range vv {
				m, ok := f.String[field.Name()]
				if !ok {
					m = make(map[string]bool)
					f.String[field.Name()] = m
				}
				m[v] = true
			}
		default:
			// Do nothing
		}
	}
	return f, nil
}

func (f filter) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	for fieldName, values := range f.Int32 {
		if !values[work.(istructs.IRowReader).AsInt32(fieldName)] {
			return nil, nil
		}
	}
	for fieldName, values := range f.String {
		if !values[work.(istructs.IRowReader).AsString(fieldName)] {
			return nil, nil
		}
	}
	return work, nil
}

type Where map[string]interface{}

func (w Where) getAsInt32(k string) (vv []int32, err error) {
	switch v := w[k].(type) {
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return nil, err
		}
		vv = append(vv, int32(i))
		return vv, nil
	case map[string]interface{}:
		in, ok := v["$in"]
		if !ok {
			return nil, errUnsupportedConstraint
		}
		params, ok := in.([]interface{})
		if !ok {
			return nil, errUnexpectedParams
		}
		for _, param := range params {
			i, err := param.(json.Number).Int64()
			if err != nil {
				return nil, err
			}
			vv = append(vv, int32(i))
		}
		return vv, nil
	case nil:
		return
	default:
		return nil, errUnsupportedType
	}
}
func (w Where) getAsString(k string) (vv []string, err error) {
	switch v := w[k].(type) {
	case string:
		vv = append(vv, v)
		return
	case map[string]interface{}:
		in, ok := v["$in"]
		if !ok {
			return nil, errUnsupportedConstraint
		}
		params, ok := in.([]interface{})
		if !ok {
			return nil, errUnexpectedParams
		}
		for _, param := range params {
			vv = append(vv, param.(string))
		}
		return
	case nil:
		return
	default:
		return nil, errUnsupportedType
	}
}
