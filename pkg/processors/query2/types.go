/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"sort"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/sys/collection"
)

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
	APIPath() processors.APIPath
	RequestCtx() context.Context
	QName() appdef.QName // e.g. Doc, View, Role
	PartitionID() istructs.PartitionID
	Host() string
	Token() string
	WorkspaceQName() appdef.QName // actually wsKind
	Accept() string
}

type apiPathHandler struct {
	isArrayResult   bool
	requestOpKind   appdef.OperationKind
	checkRateLimit  func(ctx context.Context, qw *queryWork) error
	setRequestType  func(ctx context.Context, qw *queryWork) error
	setResultType   func(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error
	authorizeResult func(ctx context.Context, qw *queryWork) error
	rowsProcessor   func(ctx context.Context, qw *queryWork) error
	exec            func(ctx context.Context, qw *queryWork) error
}

type SchemaMeta struct {
	SchemaTitle   string
	SchemaVersion string
	Description   string
	AppName       appdef.AppQName
}

type PublishedTypesFunc func(ws appdef.IWorkspace, role appdef.QName) iter.Seq2[appdef.IType,
	iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]

type ischema interface {
	appdef.IType
	appdef.IWithFields
}

type pathItem struct {
	Method  string
	Path    string
	APIPath processors.APIPath
}

type implIQueryMessage struct {
	appQName       appdef.AppQName
	wsid           istructs.WSID
	responder      bus.IResponder
	queryParams    QueryParams
	docID          istructs.IDType
	apiPath        processors.APIPath
	requestCtx     context.Context
	qName          appdef.QName
	partition      istructs.PartitionID
	host           string
	token          string
	workspaceQName appdef.QName
	headerAccept   string
}

var _ IQueryMessage = (*implIQueryMessage)(nil)

func (qm *implIQueryMessage) Accept() string {
	return qm.headerAccept
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
func (qm *implIQueryMessage) APIPath() processors.APIPath {
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
	keys map[string]interface{}
}

func newKeys(paths []string) (o pipeline.IAsyncOperator) {
	k := &keys{keys: make(map[string]interface{})}

	var f func(keysMap map[string]interface{}, keys []string)
	f = func(keysMap map[string]interface{}, keys []string) {
		key := keys[0]
		if len(keys) == 1 {
			keysMap[key] = true
		} else {
			intf, ok := keysMap[key]
			if !ok {
				intf = make(map[string]interface{})
			}
			f(intf.(map[string]interface{}), keys[1:])
			keysMap[key] = intf
		}
	}
	for _, path := range paths {
		f(k.keys, splitPath(path))
	}
	return k
}

func (k *keys) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	var f func(keysMap map[string]interface{}, data map[string]interface{})
	f = func(keysMap map[string]interface{}, data map[string]interface{}) {
		for key := range data {
			switch v1 := keysMap[key].(type) {
			case bool:
				// Do nothing
			case map[string]interface{}:
				switch v2 := data[key].(type) {
				case map[string]interface{}:
					f(v1, v2)
				case []map[string]interface{}:
					for i := range v2 {
						f(v1, v2[i])
					}
				}
			case nil:
				delete(data, key)
			}
		}
	}
	f(k.keys, work.(objectBackedByMap).data)
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
	a.ww = a.ww[a.params.Constraints.Skip : a.params.Constraints.Skip+a.params.Constraints.Limit]
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
	responder          bus.IResponder
	rowsProcessorErrCh chan error
}

type arraySender struct {
	sender
	respWriter bus.IResponseWriter
}

type objectSender struct {
	sender
	contentType string
}

func (s *arraySender) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	if s.respWriter == nil {
		s.respWriter = s.responder.StreamJSON(http.StatusOK)
	}
	return work, s.respWriter.Write(work.(objectBackedByMap).data)
}

func (s *objectSender) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	return work, s.responder.Respond(bus.ResponseMeta{ContentType: s.contentType, StatusCode: http.StatusOK}, work.(objectBackedByMap).data)
}
func (s *sender) OnError(_ context.Context, err error) {
	s.rowsProcessorErrCh <- coreutils.WrapSysError(err, http.StatusBadRequest)
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
	refFieldsAndContainers [][]string
	wsid                   istructs.WSID
	ad                     appdef.IAppDef
	records                istructs.IRecords
	viewRecords            istructs.IViewRecords
	cdoc                   bool
	events                 istructs.IEvents
}

func newInclude(qw *queryWork, cdoc bool) (o pipeline.IAsyncOperator) {
	i := &include{
		refFieldsAndContainers: make([][]string, 0),
		wsid:                   qw.msg.WSID(),
		ad:                     qw.appStructs.AppDef(),
		records:                qw.appStructs.Records(),
		viewRecords:            qw.appStructs.ViewRecords(),
		cdoc:                   cdoc,
		events:                 qw.appStructs.Events(),
	}
	for _, s := range qw.queryParams.Constraints.Include {
		i.refFieldsAndContainers = append(i.refFieldsAndContainers, strings.Split(s, "."))
	}
	return i
}

func (i include) DoAsync(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	relations, err := i.getRelations(ctx, work)
	if err != nil {
		return
	}
	for _, refFieldsAndContainers := range i.refFieldsAndContainers {
		err = i.fill(work.(objectBackedByMap).data, refFieldsAndContainers, relations, refFieldsAndContainers)
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
	if record.AsQName(appdef.SystemField_QName) != appdef.NullQName {
		return coreutils.FieldsToMap(record, i.ad), nil
	}
	record, err = i.events.GetORec(i.wsid, id, istructs.NullOffset)
	if err != nil {
		return
	}
	return coreutils.FieldsToMap(record, i.ad), nil
}
func (i include) fill(parent map[string]interface{}, refFieldsAndContainers []string, relations map[istructs.RecordID]map[string][]istructs.RecordID, refFieldOrContainerExpression []string) (err error) {
	if len(refFieldsAndContainers) == 0 {
		return nil
	}

	err = i.checkField(parent, refFieldsAndContainers[0], refFieldOrContainerExpression)
	if err != nil {
		return
	}

	if id, ok := parent[appdef.SystemField_ID].(istructs.RecordID); ok {
		for container, ids := range relations[id] {
			if container != refFieldsAndContainers[0] {
				continue
			}
			_, hasChildren := parent[container]
			if hasChildren {
				continue
			}
			parent[container] = ids
		}
	}

	switch v := parent[refFieldsAndContainers[0]].(type) {
	case istructs.RecordID:
		child, e := i.recordToMap(v)
		if e != nil {
			return e
		}
		parent[refFieldsAndContainers[0]] = child
		e = i.fill(child, refFieldsAndContainers[1:], relations, refFieldOrContainerExpression)
		if e != nil {
			return e
		}
	case map[string]interface{}:
		e := i.fill(v, refFieldsAndContainers[1:], relations, refFieldOrContainerExpression)
		if e != nil {
			return e
		}
	case []istructs.RecordID:
		items := make([]map[string]interface{}, 0)
		for _, id := range v {
			item, e := i.recordToMap(id)
			if e != nil {
				return e
			}
			e = i.fill(item, refFieldsAndContainers[1:], relations, refFieldOrContainerExpression)
			if e != nil {
				return e
			}
			items = append(items, item)
		}
		parent[refFieldsAndContainers[0]] = items
	case []map[string]interface{}:
		items := make([]map[string]interface{}, 0)
		for _, item := range v {
			err = i.fill(item, refFieldsAndContainers[1:], relations, refFieldOrContainerExpression)
			if err != nil {
				return
			}
			items = append(items, item)
		}
		parent[refFieldsAndContainers[0]] = items
	case nil:
		// Do nothing
	default:
		return errUnsupportedType
	}
	return
}
func (i include) getRelations(ctx context.Context, work pipeline.IWorkpiece) (relations map[istructs.RecordID]map[string][]istructs.RecordID, err error) {
	relations = make(map[istructs.RecordID]map[string][]istructs.RecordID)
	if !i.cdoc {
		return
	}
	kbCollectionView := i.viewRecords.KeyBuilder(collection.QNameCollectionView)
	kbCollectionView.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
	kbCollectionView.PutQName(collection.Field_DocQName, work.(objectBackedByMap).data[appdef.SystemField_QName].(appdef.QName))
	kbCollectionView.PutRecordID(collection.Field_DocID, work.(objectBackedByMap).data[appdef.SystemField_ID].(istructs.RecordID))
	err = i.viewRecords.Read(ctx, i.wsid, kbCollectionView, func(key istructs.IKey, value istructs.IValue) (err error) {
		record := value.AsRecord(collection.Field_Record)
		if record.AsRecordID(appdef.SystemField_ParentID) == istructs.NullRecordID {
			return
		}

		containers, okContainers := relations[record.AsRecordID(appdef.SystemField_ParentID)]
		if !okContainers {
			containers = make(map[string][]istructs.RecordID)
		}
		container, okContainer := containers[record.AsString(appdef.SystemField_Container)]
		if !okContainer {
			container = make([]istructs.RecordID, 0)
		}
		container = append(container, record.ID())

		containers[record.AsString(appdef.SystemField_Container)] = container
		relations[record.AsRecordID(appdef.SystemField_ParentID)] = containers
		return
	})
	return
}
func (i include) checkField(parent map[string]interface{}, refFieldOrContainer string, refFieldOrContainerExpression []string) (err error) {
	iType := i.ad.Type(parent[appdef.SystemField_QName].(appdef.QName))
	if withFields, ok := iType.(appdef.IWithFields); ok {
		for _, field := range withFields.RefFields() {
			if field.Name() == refFieldOrContainer {
				return nil
			}
		}
	}
	if withContainers, ok := iType.(appdef.IWithContainers); ok {
		for _, field := range withContainers.Containers() {
			if field.Name() == refFieldOrContainer {
				return nil
			}
		}
	}
	return fmt.Errorf("field expression - '%s', '%s' - %w", strings.Join(refFieldOrContainerExpression, "."), refFieldOrContainer, errUnexpectedField)
}
