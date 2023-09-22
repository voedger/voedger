/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func provideQryCDoc(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		qNameCDocFunc,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CDocParams")).
			AddField(field_ID, appdef.DataKind_int64, true).(appdef.IType).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "CDocResult")).
			AddField("Result", appdef.DataKind_string, false).(appdef.IType).QName(),
		execQryCDoc(appDefBuilder)))
}

func execQryCDoc(appDef appdef.IAppDef) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, qf istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		rkb, err := args.State.KeyBuilder(state.RecordsStorage, appdef.NullQName)
		if err != nil {
			return
		}
		rkb.PutRecordID(state.Field_ID, istructs.RecordID(args.ArgumentObject.AsInt64(field_ID)))
		rsv, err := args.State.MustExist(rkb)
		if err != nil {
			return
		}

		vrkb, err := args.State.KeyBuilder(state.ViewRecordsStorage, QNameViewCollection)
		if err != nil {
			return
		}
		vrkb.PutQName(Field_DocQName, rsv.AsQName(appdef.SystemField_QName))
		vrkb.PutInt32(Field_PartKey, PartitionKeyCollection)
		vrkb.PutRecordID(field_DocID, rsv.AsRecordID(appdef.SystemField_ID))

		var doc *collectionElement

		// build tree
		err = args.State.Read(vrkb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			rec := value.AsRecord(Field_Record)
			if doc == nil {
				cobj := newCollectionElement(rec)
				doc = &cobj
			} else {
				doc.addRawRecord(rec)
			}
			return
		})
		if err != nil {
			return
		}

		if doc == nil {
			return coreutils.NewHTTPErrorf(http.StatusNotFound, "Document not found")
		}

		doc.handleRawRecords()

		var bytes []byte
		var obj map[string]interface{}
		refs := make(map[istructs.RecordID]bool)
		obj, err = convert(doc, appDef, refs, istructs.NullRecordID)
		if err != nil {
			return
		}
		err = addRefs(obj, refs, args.State, appDef)
		if err != nil {
			return
		}
		bytes, err = marshal(obj)
		if err != nil {
			return
		}
		return callback(&cdocObject{data: string(bytes)})
	}
}
func convert(doc istructs.IElement, appDef appdef.IAppDef, refs map[istructs.RecordID]bool, parent istructs.RecordID) (obj map[string]interface{}, err error) {
	if doc == nil {
		return nil, nil
	}
	obj = coreutils.FieldsToMap(doc, appDef, coreutils.Filter(func(fieldName string, kind appdef.DataKind) bool {
		if skipField(fieldName) {
			return false
		}
		if refs != nil {
			if kind == appdef.DataKind_RecordID && fieldName != appdef.SystemField_ID {
				// the field is a reference
				if parent != doc.AsRecordID(fieldName) {
					refs[doc.AsRecordID(fieldName)] = true
				}
			}
		}
		return true
	}))
	doc.Containers(func(container string) {
		list := make([]interface{}, 0)
		doc.Elements(container, func(el istructs.IElement) {
			var elObj map[string]interface{}
			if err == nil {
				elObj, err = convert(el.(*collectionElement), appDef, refs, doc.AsRecord().ID())
				if err == nil {
					list = append(list, elObj)
				}
			}
		})
		if container != "" {
			obj[container] = list
		}
	})

	return obj, nil
}
func addRefs(obj map[string]interface{}, refs map[istructs.RecordID]bool, s istructs.IState, appDef appdef.IAppDef) error {
	if len(refs) == 0 {
		return nil
	}

	references := make(map[string]map[string]interface{})
	for recordId := range refs {
		if recordId == istructs.NullRecordID {
			continue
		}
		rkb, err := s.KeyBuilder(state.RecordsStorage, appdef.NullQName)
		if err != nil {
			return err
		}
		rkb.PutRecordID(state.Field_ID, recordId)

		rkv, err := s.MustExist(rkb)
		if err != nil {
			return err
		}

		recmap, ok := references[rkv.AsQName(appdef.SystemField_QName).String()]
		if !ok {
			recmap = make(map[string]interface{})
			references[rkv.AsQName(appdef.SystemField_QName).String()] = recmap
		}
		recKey := strconv.FormatInt(int64(recordId), DEC)
		if _, ok := recmap[recKey]; !ok {
			elem := newCollectionElement(rkv.AsRecord(""))
			obj, err := convert(&elem, appDef, nil, istructs.NullRecordID)
			if err != nil {
				return err
			}
			recmap[recKey] = obj
		}
	}
	obj[field_xrefs] = references
	return nil
}
func marshal(obj map[string]interface{}) ([]byte, error) {
	if obj == nil {
		return nil, nil
	}
	return json.Marshal(obj)
}

func skipField(fieldName string) bool {
	return fieldName == appdef.SystemField_QName ||
		fieldName == appdef.SystemField_Container ||
		fieldName == appdef.SystemField_ParentID

}

type cdocObject struct {
	istructs.NullObject
	data string
}

func (o cdocObject) AsString(string) string { return o.data }
