/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package workspace

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func execCmdInitChildWorkspace(args istructs.ExecCommandArgs) (err error) {
	wsName := args.ArgumentObject.AsString(authnz.Field_WSName)
	kb, err := args.State.KeyBuilder(sys.Storage_View, QNameViewChildWorkspaceIdx)
	if err != nil {
		return
	}
	kb.PutInt32(field_dummy, 1)
	kb.PutString(authnz.Field_WSName, wsName)
	_, ok, err := args.State.CanExist(kb)
	if err != nil {
		return
	}

	if ok {
		return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf("child workspace with name %s already exists", wsName))
	}

	wsKind := args.ArgumentObject.AsQName(authnz.Field_WSKind)
	appDef := args.State.AppStructs().AppDef()
	if appDef.WorkspaceByDescriptor(wsKind) == nil {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("provided WSKind %s is not a QName of a workspace descriptor", wsKind))
	}

	wsKindInitializationData := args.ArgumentObject.AsString(authnz.Field_WSKindInitializationData)
	templateName := args.ArgumentObject.AsString(field_TemplateName)
	wsClusterID := args.ArgumentObject.AsInt32(authnz.Field_WSClusterID)
	if wsClusterID == 0 {
		wsClusterID = int32(istructs.CurrentClusterID())
	}
	templateParams := args.ArgumentObject.AsString(Field_TemplateParams)

	// Create cdoc.sys.ChildWorkspace
	kb, err = args.State.KeyBuilder(sys.Storage_Record, authnz.QNameCDocChildWorkspace)
	if err != nil {
		return
	}
	cdocChildWS, err := args.Intents.NewValue(kb)
	if err != nil {
		return
	}
	cdocChildWS.PutRecordID(appdef.SystemField_ID, 1)
	cdocChildWS.PutString(authnz.Field_WSName, wsName)
	cdocChildWS.PutQName(authnz.Field_WSKind, wsKind)
	cdocChildWS.PutString(authnz.Field_WSKindInitializationData, wsKindInitializationData)
	cdocChildWS.PutString(field_TemplateName, templateName)
	cdocChildWS.PutInt32(authnz.Field_WSClusterID, wsClusterID)
	cdocChildWS.PutString(Field_TemplateParams, templateParams)

	return err
}

var childWorkspaceIdxProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) error {
	for rec := range event.CUDs {
		if rec.QName() != authnz.QNameCDocChildWorkspace || !rec.IsNew() {
			continue
		}

		kb, err := s.KeyBuilder(sys.Storage_View, QNameViewChildWorkspaceIdx)
		if err != nil {
			return err
		}
		kb.PutInt32(field_dummy, 1)
		wsName := rec.AsString(authnz.Field_WSName)
		kb.PutString(authnz.Field_WSName, wsName)

		vb, err := intents.NewValue(kb)
		if err != nil {
			return err
		}
		vb.PutInt64(Field_ChildWorkspaceID, int64(rec.ID()))
	}
	return nil
}

// targetApp/parentWSID/q.sys.QueryChildWorkspaceByName
func qcwbnQryExec(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
	wsName := args.ArgumentObject.AsString(authnz.Field_WSName)
	kb, err := args.State.KeyBuilder(sys.Storage_View, QNameViewChildWorkspaceIdx)
	if err != nil {
		return err
	}
	kb.PutInt32(field_dummy, 1)
	kb.PutString(authnz.Field_WSName, wsName)
	childWSIdx, ok, err := args.State.CanExist(kb)
	if err != nil {
		return err
	}
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusNotFound, "child workspace ", wsName, " not found")
	}
	kb, err = args.State.KeyBuilder(sys.Storage_Record, appdef.NullQName)
	if err != nil {
		return err
	}
	kb.PutRecordID(sys.Storage_Record_Field_ID, istructs.RecordID(childWSIdx.AsInt64(Field_ChildWorkspaceID))) // nolint G115
	rec, err := args.State.MustExist(kb)
	if err != nil {
		return err
	}
	return callback(&qcwbnRR{
		wsName:                   rec.AsString(authnz.Field_WSName),
		wsKind:                   rec.AsQName(authnz.Field_WSKind),
		wsKindInitializationData: rec.AsString(authnz.Field_WSKindInitializationData),
		templateName:             rec.AsString(field_TemplateName),
		templateParams:           rec.AsString(Field_TemplateParams),
		wsid:                     rec.AsInt64(authnz.Field_WSID),
		wsError:                  rec.AsString(authnz.Field_WSError),
	})
}

// q.sys.QueryChildWorkspaceByName
type qcwbnRR struct {
	istructs.NullObject
	wsName                   string
	wsKind                   appdef.QName
	wsKindInitializationData string
	templateName             string
	templateParams           string
	wsid                     int64
	wsError                  string
}

func (q *qcwbnRR) AsInt64(string) int64 { return q.wsid }
func (q *qcwbnRR) AsString(name string) string {
	switch name {
	case authnz.Field_WSName:
		return q.wsName
	case authnz.Field_WSKindInitializationData:
		return q.wsKindInitializationData
	case field_TemplateName:
		return q.templateName
	case authnz.Field_WSError:
		return q.wsError
	case authnz.Field_WSKind:
		return q.wsKind.String()
	case Field_TemplateParams:
		return q.templateParams
	default:
		panic("unexpected field to return: " + name)
	}
}
