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

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
)

func provideQryCDoc(sr istructsmem.IStatelessResources) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		qNameQueryGetCDoc,
		execQryCDoc),
	)
}

func execQryCDoc(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	rkb, err := args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return
	}
	rkb.PutRecordID(sys.Storage_Record_Field_ID, istructs.RecordID(args.ArgumentObject.AsInt64(field_ID))) // nolint G115
	rsv, err := args.State.MustExist(rkb)
	if err != nil {
		return
	}

	vrkb, err := args.State.KeyBuilder(sys.Storage_View, QNameCollectionView)
	if err != nil {
		return
	}
	vrkb.PutQName(Field_DocQName, rsv.AsQName(appdef.SystemField_QName))
	vrkb.PutInt32(Field_PartKey, PartitionKeyCollection)
	vrkb.PutRecordID(Field_DocID, rsv.AsRecordID(appdef.SystemField_ID))

	var doc *collectionObject

	// build tree
	err = args.State.Read(vrkb, func(key istructs.IKey, value istructs.IStateValue) (err error) {
		rec := value.(istructs.IStateViewValue).AsRecord(Field_Record)
		if doc == nil {
			cobj := newCollectionObject(rec)
			doc = cobj
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
	appDef := args.State.AppStructs().AppDef()
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

func convert(doc istructs.IObject, appDef appdef.IAppDef, refs map[istructs.RecordID]bool, parent istructs.RecordID) (obj map[string]interface{}, err error) {
	if doc == nil {
		return nil, nil
	}
	// unable to use ObjectToMap because of filter:
	// field of the root is filtering -> no problem, field of a container is filtering -> `doc` var here ir root, it does not contain fields of container -> panic
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
	for container := range doc.Containers {
		list := make([]interface{}, 0)
		for c := range doc.Children(container) {
			var childObj map[string]interface{}
			childObj, err = convert(c.(*collectionObject), appDef, refs, doc.AsRecord().ID())
			if err != nil {
				break
			}
			list = append(list, childObj)
		}
		obj[container] = list
	}

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
		rkb, err := s.KeyBuilder(sys.Storage_Record, appdef.NullQName)
		if err != nil {
			return err
		}
		rkb.PutRecordID(sys.Storage_Record_Field_ID, recordId)

		rkv, err := s.MustExist(rkb)
		if err != nil {
			return err
		}

		recmap, ok := references[rkv.AsQName(appdef.SystemField_QName).String()]
		if !ok {
			recmap = make(map[string]interface{})
			references[rkv.AsQName(appdef.SystemField_QName).String()] = recmap
		}

		recKey := utils.UintToString(recordId)
		if _, ok := recmap[recKey]; !ok {
			child := newCollectionObject(rkv.(istructs.IStateRecordValue).AsRecord())
			obj, err := convert(child, appDef, nil, istructs.NullRecordID)
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
