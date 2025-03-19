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
	QName() appdef.QName // e.g. Doc, View, Role
	PartitionID() istructs.PartitionID
	Host() string
	Token() string
	WorkspaceQName() appdef.QName // actually wsKind
}

type IApiPathHandler interface {
	CheckRateLimit(ctx context.Context, qw *queryWork) error
	SetRequestType(ctx context.Context, qw *queryWork) error
	SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error
	AuthorizeResult(ctx context.Context, qw *queryWork) error
	RowsProcessor(ctx context.Context, qw *queryWork) error
	Exec(ctx context.Context, qw *queryWork) error
	RequestOpKind() appdef.OperationKind
}

type implIQueryMessage struct {
	appQName       appdef.AppQName
	wsid           istructs.WSID
	responder      bus.IResponder
	queryParams    QueryParams
	docID          istructs.IDType
	apiPath        ApiPath
	requestCtx     context.Context
	qName          appdef.QName
	partition      istructs.PartitionID
	host           string
	token          string
	workspaceQName appdef.QName
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

func (qm *implIQueryMessage) WorkspaceQName() appdef.QName {
	return qm.workspaceQName
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
func (o objectBackedByMap) SpecifiedValues(cb func(appdef.IField, interface{}) bool) {
	for fieldName, val := range o.data {
		if !cb(&coreutils.MockIField{FieldName: fieldName}, val) {
			return
		}
	}
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
			switch typed := vi.(type) {
			case int32:
				return a.compareInt32(typed, vj.(int32), asc)
			case int64:
				return a.compareInt64(typed, vj.(int64), asc)
			case float32:
				return a.compareFloat32(typed, vj.(float32), asc)
			case float64:
				return a.compareFloat64(typed, vj.(float64), asc)
			case []byte:
				return a.compareString(string(typed), string(vi.([]byte)), asc)
			case string:
				return a.compareString(typed, vj.(string), asc)
			case appdef.QName:
				return a.compareString(typed.String(), vj.(appdef.QName).String(), asc)
			case bool:
				return typed != vj.(bool)
			case istructs.RecordID:
				return a.compareUint64(uint64(typed), uint64(vj.(istructs.RecordID)), asc)
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
	responder  bus.IResponder
	respWriter bus.IResponseWriter
}

func (s *sender) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	if s.respWriter == nil {
		s.respWriter = s.responder.InitResponse(http.StatusOK)
	}
	return work, s.respWriter.Write(work.(objectBackedByMap).data)
}

type filter struct {
	pipeline.AsyncNOOP
	Int32  map[string]map[int32]bool
	String map[string]map[string]bool
}

func newFilter(qw *queryWork, fields []appdef.IField) (o pipeline.IAsyncOperator, err error) {
	f := &filter{
		Int32:  make(map[string]map[int32]bool),
		String: make(map[string]map[string]bool),
	}
	if qw.queryParams.Constraints == nil || qw.queryParams.Constraints.Where == nil || len(qw.queryParams.Constraints.Where) == 0 {
		return nil, nil
	}
	for _, field := range fields {
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
	if len(f.Int32) == 0 && len(f.String) == 0 {
		return nil, nil
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
		int32Val, err := coreutils.ClarifyJSONNumber(v, appdef.DataKind_int32)
		if err != nil {
			return nil, err
		}
		vv = append(vv, int32Val.(int32))
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
			int32Val, err := coreutils.ClarifyJSONNumber(param.(json.Number), appdef.DataKind_int32)
			if err != nil {
				return nil, err
			}
			vv = append(vv, int32Val.(int32))
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

type queryResultWrapper struct {
	istructs.IObject
	qName appdef.QName
}

func (w queryResultWrapper) AsQName(name appdef.FieldName) appdef.QName {
	if name == appdef.SystemField_QName {
		return w.qName
	}
	return w.IObject.AsQName(name)
}

type include struct {
	pipeline.AsyncNOOP
	sss     [][]string
	wsid    istructs.WSID
	records istructs.IRecords
	ad      appdef.IAppDef
}

func newInclude(qw *queryWork) (o pipeline.IAsyncOperator) {
	i := &include{
		sss:     make([][]string, 0),
		records: qw.appStructs.Records(),
		wsid:    qw.msg.WSID(),
		ad:      qw.appStructs.AppDef(),
	}
	for _, s := range qw.queryParams.Constraints.Include {
		i.sss = append(i.sss, strings.Split(s, "."))
	}
	return i
}

func (i include) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	for _, ss := range i.sss {
		err = i.fill(work.(objectBackedByMap).data, ss)
		if err != nil {
			return
		}
	}
	return work, nil
}
func (i include) recordToMap(id istructs.RecordID) (obj map[string]interface{}, err error) {
	record, err := i.records.Get(i.wsid, true, id)
	if err != nil {
		return
	}
	return coreutils.FieldsToMap(record, i.ad), nil
}
func (i include) fill(parent map[string]interface{}, ss []string) (err error) {
	if len(ss) == 0 {
		return nil
	}
	switch v := parent[ss[0]].(type) {
	case istructs.RecordID:
		child, e := i.recordToMap(v)
		if e != nil {
			return e
		}
		parent[ss[0]] = child
		e = i.fill(child, ss[1:])
		if e != nil {
			return e
		}
	case map[string]interface{}:
		e := i.fill(v, ss[1:])
		if e != nil {
			return e
		}
	default:
		return errUnsupportedType
	}
	return
}
