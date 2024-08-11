/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/iterate"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/utils/federation"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// Projector<A, InvokeCreateWorkspaceID>
// triggered by CDoc<ChildWorkspace> (not a singleton)
// targetApp/userProfileWSID
func invokeCreateWorkspaceIDProjector(federation federation.IFederation, tokensAPI itokens.ITokens) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		return iterate.ForEachError(event.CUDs, func(rec istructs.ICUDRow) error {
			if rec.QName() != authnz.QNameCDocChildWorkspace || !rec.IsNew() {
				return nil
			}
			ownerWSID := event.Workspace()
			wsName := rec.AsString(authnz.Field_WSName)
			wsKind := rec.AsQName(authnz.Field_WSKind)
			templateName := rec.AsString(field_TemplateName)
			templateParams := rec.AsString(Field_TemplateParams)
			appQName := s.App()
			targetApp := appQName.String()
			targetClusterID := istructs.MainClusterID // TODO: on https://github.com/voedger/voedger/commit/1e7ce3f2c546e9bf1332edb31a5beed5954bc476 was NullClusetrID!
			wsidToCallCreateWSIDAt := coreutils.GetPseudoWSID(ownerWSID, wsName, targetClusterID)
			return ApplyInvokeCreateWorkspaceID(federation, appQName, tokensAPI, wsName, wsKind, wsidToCallCreateWSIDAt, targetApp,
				templateName, templateParams, rec, ownerWSID)
		})
	}
}

// triggered by cdoc.registry.Login or by cdoc.sys.ChildWorkspace
// wsid - pseudoProfile: crc32(wsName) or crc32(login)
// sys/registry app
func ApplyInvokeCreateWorkspaceID(federation federation.IFederation, appQName appdef.AppQName, tokensAPI itokens.ITokens,
	wsName string, wsKind appdef.QName, wsidToCallCreateWSIDAt istructs.WSID, targetApp string, templateName string, templateParams string,
	ownerDoc istructs.ICUDRow, ownerWSID istructs.WSID) error {
	// Call WS[$PseudoWSID].c.CreateWorkspaceID()
	ownerApp := appQName.String()
	ownerQName := ownerDoc.QName()
	ownerID := ownerDoc.ID()
	wsKindInitializationData := ownerDoc.AsString(authnz.Field_WSKindInitializationData)
	createWSIDCmdURL := fmt.Sprintf("api/%s/%d/c.sys.CreateWorkspaceID", targetApp, wsidToCallCreateWSIDAt)
	logger.Info("aproj.sys.InvokeCreateWorkspaceID: request to " + createWSIDCmdURL)
	body := fmt.Sprintf(`{"args":{"OwnerWSID":%d,"OwnerQName2":"%s","OwnerID":%d,"OwnerApp":"%s","WSName":%q,"WSKind":"%s","WSKindInitializationData":%q,"TemplateName":"%s","TemplateParams":%q}}`,
		ownerWSID, ownerQName.String(), ownerID, ownerApp, wsName, wsKind.String(), wsKindInitializationData, templateName, templateParams)
	targetAppQName, err := appdef.ParseAppQName(targetApp)
	if err != nil {
		// parsed already by c.registry.CreateLogin
		// notest
		return err
	}
	systemPrincipalToken, err := payloads.GetSystemPrincipalToken(tokensAPI, targetAppQName)
	if err != nil {
		return fmt.Errorf("aproj.sys.InvokeCreateWorkspaceID: %w", err)
	}

	if _, err = federation.Func(createWSIDCmdURL, body,
		coreutils.WithAuthorizeBy(systemPrincipalToken),
		coreutils.WithDiscardResponse(),
		coreutils.WithExpectedCode(http.StatusOK),
		coreutils.WithExpectedCode(http.StatusConflict),
	); err != nil {
		return fmt.Errorf("aproj.sys.InvokeCreateWorkspaceID: c.sys.CreateWorkspaceID failed: %w. Body:\n%s", err, body)
	}
	return nil
}

// c.sys.CreateWorkspaceID
// ChildWorkspace -> pseudoWSID(profileWSID+"/"+wsName, targetCluster) translated to AppWSID
// Login -> ((PseudoWSID->AppWSID).Base, targetCluster)
// targetApp
func execCmdCreateWorkspaceID(args istructs.ExecCommandArgs) (err error) {
	// TODO: AuthZ: System,SystemToken in header
	ownerWSID := args.ArgumentObject.AsInt64(Field_OwnerWSID)
	wsName := args.ArgumentObject.AsString(authnz.Field_WSName)
	// Check that ownerWSID + wsName does not exist yet: View<WorkspaceIDIdx> to deduplication
	kb, err := args.State.KeyBuilder(sys.Storage_View, QNameViewWorkspaceIDIdx)
	if err != nil {
		return err
	}
	kb.PutInt64(Field_OwnerWSID, ownerWSID)
	kb.PutString(authnz.Field_WSName, wsName)
	_, ok, err := args.State.CanExist(kb)
	if err != nil {
		return err
	}
	if ok {
		return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf("workspace with name %s and ownerWSID %d already exists", wsName, ownerWSID))
	}

	// Get new WSID from View<NextBaseWSID>
	as := args.State.AppStructs()
	newWSID, err := GetNextWSID(args.Workpiece.(interface{ Context() context.Context }).Context(), as, args.WSID.ClusterID())
	if err != nil {
		return err
	}

	// Create CDoc<WorkspaceID>{wsParams, WSID: $NewWSID}
	kb, err = args.State.KeyBuilder(sys.Storage_Record, QNameCDocWorkspaceID)
	if err != nil {
		return err
	}
	cdocWorkspaceID, err := args.Intents.NewValue(kb)
	if err != nil {
		return err
	}
	cdocWorkspaceID.PutRecordID(appdef.SystemField_ID, 1)
	cdocWorkspaceID.PutInt64(Field_OwnerWSID, args.ArgumentObject.AsInt64(Field_OwnerWSID))       // CDoc<Login> -> pseudoWSID->AppWSID, CDoc<ChildWorkspace> -> owner profile WSID
	cdocWorkspaceID.PutString(Field_OwnerQName2, args.ArgumentObject.AsString(Field_OwnerQName2)) // registry.Login or sys.UserProfile
	cdocWorkspaceID.PutInt64(Field_OwnerID, args.ArgumentObject.AsInt64(Field_OwnerID))           // CDoc<Login>.ID or CDoc<ChildWorkspace>.ID
	cdocWorkspaceID.PutString(Field_OwnerApp, args.ArgumentObject.AsString(Field_OwnerApp))
	cdocWorkspaceID.PutString(authnz.Field_WSName, args.ArgumentObject.AsString(authnz.Field_WSName)) // CDoc<Login> -> crc32(loginHash), CDoc<ChildWorkspace> -> wsName
	cdocWorkspaceID.PutQName(authnz.Field_WSKind, args.ArgumentObject.AsQName(authnz.Field_WSKind))   // CDoc<Login> -> sys.DeviceProfile or sys.UserProfile, CDoc<ChildWorkspace> -> provided wsKind (e.g. air.Restaurant)
	cdocWorkspaceID.PutString(authnz.Field_WSKindInitializationData, args.ArgumentObject.AsString(authnz.Field_WSKindInitializationData))
	cdocWorkspaceID.PutString(field_TemplateName, args.ArgumentObject.AsString(field_TemplateName))
	cdocWorkspaceID.PutString(Field_TemplateParams, args.ArgumentObject.AsString(Field_TemplateParams))
	cdocWorkspaceID.PutInt64(authnz.Field_WSID, int64(newWSID))

	return
}

// sp.sys.WorkspaceIDIdx
// triggered by cdoc.sys.WorkspaceID
// targetApp/appWS
func workspaceIDIdxProjector(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return iterate.ForEachError(event.CUDs, func(rec istructs.ICUDRow) error {
		if rec.QName() != QNameCDocWorkspaceID || !rec.IsNew() { // skip on update cdoc.sys.WorkspaceID on e.g. deactivate workspace
			return nil
		}
		kb, err := s.KeyBuilder(sys.Storage_View, QNameViewWorkspaceIDIdx)
		if err != nil {
			// notest
			return nil
		}
		ownerWSID := rec.AsInt64(Field_OwnerWSID)
		wsName := rec.AsString(authnz.Field_WSName)
		wsid := rec.AsInt64(authnz.Field_WSID)
		kb.PutInt64(Field_OwnerWSID, ownerWSID)
		kb.PutString(authnz.Field_WSName, wsName)
		wsIdxVB, err := intents.NewValue(kb)
		if err != nil {
			// notest
			return nil
		}
		wsIdxVB.PutInt64(authnz.Field_WSID, wsid)
		wsIdxVB.PutRecordID(field_IDOfCDocWorkspaceID, rec.ID())
		return nil
	})
}

// Projector<A, InvokeCreateWorkspace>
// triggered by CDoc<WorkspaceID>
// targetApp/appWS
func invokeCreateWorkspaceProjector(federation federation.IFederation, tokensAPI itokens.ITokens) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		return iterate.ForEachError(event.CUDs, func(rec istructs.ICUDRow) error {
			if rec.QName() != QNameCDocWorkspaceID || !rec.IsNew() { // skip on update cdoc.sys.WorkspaceID on e.g. deactivate workspace
				return nil
			}

			newWSID := rec.AsInt64(authnz.Field_WSID)
			wsName := rec.AsString(authnz.Field_WSName)
			wsKind := rec.AsQName(authnz.Field_WSKind)
			wsKindInitializationData := rec.AsString(authnz.Field_WSKindInitializationData)
			templateName := rec.AsString(field_TemplateName)
			ownerWSID := rec.AsInt64(Field_OwnerWSID)
			ownerQName := rec.AsString(Field_OwnerQName2)
			ownerID := rec.AsInt64(Field_OwnerID)
			ownerApp := rec.AsString(Field_OwnerApp)
			templateParams := rec.AsString(Field_TemplateParams)
			body := fmt.Sprintf(`{"args":{"OwnerWSID":%d,"OwnerQName2":"%s","OwnerID":%d,"OwnerApp":"%s","WSName":%q,"WSKind":"%s","WSKindInitializationData":%q,"TemplateName":"%s","TemplateParams":%q}}`,
				ownerWSID, ownerQName, ownerID, ownerApp, wsName, wsKind.String(), wsKindInitializationData, templateName, templateParams)
			appQName := s.App()
			createWSCmdURL := fmt.Sprintf("api/%s/%d/c.sys.CreateWorkspace", appQName.String(), newWSID)
			logger.Info("aproj.sys.InvokeCreateWorkspace: request to " + createWSCmdURL)
			systemPrincipalToken, err := payloads.GetSystemPrincipalToken(tokensAPI, appQName)
			if err != nil {
				return fmt.Errorf("aproj.sys.InvokeCreateWorkspace: %w", err)
			}
			if _, err = federation.Func(createWSCmdURL, body, coreutils.WithAuthorizeBy(systemPrincipalToken), coreutils.WithDiscardResponse()); err != nil {
				return fmt.Errorf("aproj.sys.InvokeCreateWorkspace: c.sys.CreateWorkspace failed: %w", err)
			}
			return nil
		})
	}
}

// c.sys.CreateWorkspace
// должно быть вызвано в целевом приложении, т.к. профиль пользователя находится в целевом приложении на схеме!!!
func execCmdCreateWorkspace(now coreutils.TimeFunc) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) error {
		// TODO: AuthZ: System, SystemToken in header
		// Check that CDoc<sys.WorkspaceDescriptor> does not exist yet (IRecords.GetSingleton())
		wsKindInitializationDataStr := args.ArgumentObject.AsString(authnz.Field_WSKindInitializationData)
		wsKind := args.ArgumentObject.AsQName(authnz.Field_WSKind)
		newWSID := args.WSID

		wsKindInitializationData := map[string]interface{}{}

		e := func() error {
			as := args.State.AppStructs()
			wsKindType := as.AppDef().Type(wsKind)
			if wsKindType.Kind() == appdef.TypeKind_null {
				return fmt.Errorf("unknown workspace kind: %s", wsKind.String())
			}
			if len(wsKindInitializationDataStr) == 0 {
				return nil
			}
			// validate wsKindInitializationData
			if err := json.Unmarshal([]byte(wsKindInitializationDataStr), &wsKindInitializationData); err != nil {
				return fmt.Errorf("failed to unmarshal workspace initialization data: %w", err)
			}
			if err := validateWSKindInitializationData(as, wsKindInitializationData, wsKindType); err != nil {
				return fmt.Errorf("failed to validate workspace initialization data: %w", err)
			}
			return nil
		}()

		// create CDoc<sys.WorkspaceDescriptor> (singleton)
		kb, err := args.State.KeyBuilder(sys.Storage_Record, authnz.QNameCDocWorkspaceDescriptor)
		if err != nil {
			return err
		}
		cdocWSDesc, err := args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
		cdocWSDesc.PutInt64(Field_OwnerWSID, args.ArgumentObject.AsInt64(Field_OwnerWSID))           // CDoc<Login> -> pseudo WSID, CDoc<ChildWorkspace> -> owner profile WSID
		cdocWSDesc.PutString(Field_OwnerQName2, args.ArgumentObject.AsString(Field_OwnerQName2))     // registry.Login or sys.UserProfile
		cdocWSDesc.PutInt64(Field_OwnerID, args.ArgumentObject.AsInt64(Field_OwnerID))               // CDoc<Login>.ID or CDoc<ChildWorkspace>.ID
		cdocWSDesc.PutString(authnz.Field_WSName, args.ArgumentObject.AsString(authnz.Field_WSName)) // CDoc<Login> -> "hardcoded", CDoc<ChildWorkspace> -> wsName
		cdocWSDesc.PutQName(authnz.Field_WSKind, wsKind)                                             // CDoc<Login> -> sys.DeviceProfile or sys.UserProfile, CDoc<ChildWorkspace> -> provided wsKind (e.g. air.Restaurant)
		cdocWSDesc.PutString(Field_OwnerApp, args.ArgumentObject.AsString(Field_OwnerApp))
		cdocWSDesc.PutString(authnz.Field_WSKindInitializationData, wsKindInitializationDataStr)
		cdocWSDesc.PutString(field_TemplateName, args.ArgumentObject.AsString(field_TemplateName))
		cdocWSDesc.PutString(Field_TemplateParams, args.ArgumentObject.AsString(Field_TemplateParams))
		cdocWSDesc.PutInt64(authnz.Field_WSID, int64(newWSID))
		cdocWSDesc.PutInt64(authnz.Field_CreatedAtMs, now().UnixMilli())
		cdocWSDesc.PutInt32(authnz.Field_Status, int32(authnz.WorkspaceStatus_Active))
		if e != nil {
			cdocWSDesc.PutString(Field_CreateError, e.Error())
			logger.Info("c.sys.CreateWorkspace: ", e.Error())
		} else {
			// if no error create CDoc{$wsKind}
			kb, err := args.State.KeyBuilder(sys.Storage_Record, wsKind)
			if err != nil {
				return err
			}
			cdocWSKind, err := args.Intents.NewValue(kb)
			if err != nil {
				return err
			}
			cdocWSKind.PutRecordID(appdef.SystemField_ID, 2)
			return coreutils.MapToObject(wsKindInitializationData, cdocWSKind) // validated already in func()
		}
		return nil
	}
}

// Projector<A, InitializeWorkspace>
// triggered by CDoc<WorkspaceDescriptor>
func initializeWorkspaceProjector(nowFunc coreutils.TimeFunc, federation federation.IFederation, eps map[appdef.AppQName]extensionpoints.IExtensionPoint,
	tokensAPI itokens.ITokens, wsPostInitFunc WSPostInitFunc) func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
		return iterate.ForEachError(event.CUDs, func(rec istructs.ICUDRow) error {
			if rec.QName() != authnz.QNameCDocWorkspaceDescriptor {
				return nil
			}
			if rec.AsQName(authnz.Field_WSKind) == authnz.QNameCDoc_WorkspaceKind_AppWorkspace {
				// AppWS -> self-initialized already
				return nil
			}
			// If updated return. We do NOT react on update since we update record from projector
			if !rec.IsNew() {
				return nil
			}
			ownerUpdated := false
			wsDescr := rec
			newWSID := rec.AsInt64(authnz.Field_WSID)
			newWSName := wsDescr.AsString(authnz.Field_WSName)
			ownerApp := rec.AsString(Field_OwnerApp)
			var wsError error
			logPrefix := fmt.Sprintf("aproj.sys.InitializeWorkspace[%s:%d]>:", newWSName, newWSID)
			info := func(args ...interface{}) {
				logger.Info(logPrefix, args)
			}

			er := func(args ...interface{}) {
				logger.Error(logPrefix, args)
			}
			defer func() {
				if ownerUpdated {
					if wsError != nil {
						info("initialization completed with error:", wsError)
					} else {
						info("initialization completed")
					}
				} else {
					info("initialization not completed because updateOwner() failed")
				}
			}()

			info(workspace, newWSName, "init started")

			targetAppQName := s.App()

			systemPrincipalToken_TargetApp, err := payloads.GetSystemPrincipalToken(tokensAPI, targetAppQName)
			if err != nil {
				return fmt.Errorf("%s: %w", logPrefix, err)
			}
			ownerAppQName, err := appdef.ParseAppQName(ownerApp)
			if err != nil {
				// parsed already by c.registry.CreateLogin and InitChildWorkspace ?????????
				// notest
				return err
			}
			systemPrincipalToken_OwnerApp, err := payloads.GetSystemPrincipalToken(tokensAPI, ownerAppQName)
			if err != nil {
				return fmt.Errorf("%s: %w", logPrefix, err)
			}

			// If len(new.createError) > 0 -> UpdateOwner(wsParams, new.WSID, new.createError), return
			createErrorStr := wsDescr.AsString(Field_CreateError)
			if len(createErrorStr) > 0 {
				wsError = errors.New(createErrorStr)
				info("have new.createError, will just updateOwner():", createErrorStr)
				ownerUpdated = updateOwner(rec, ownerApp, newWSID, wsError, systemPrincipalToken_OwnerApp, federation, info, er)
				return nil
			}

			updateWSDescrURL := fmt.Sprintf("api/%s/%d/c.sys.CUD", targetAppQName.String(), event.Workspace())
			// if wsDecr.initStartedAtMs == 0
			if wsDescr.AsInt64(Field_InitStartedAtMs) == 0 {
				info("initStartedAtMs = 0. WS init was not started")
				// WS[currentWS].c.sys.CUD(wsDescr.ID, initStartedAtMs)
				body := fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"sys.QName": "%s","%s": %d}}]}`,
					wsDescr.ID(), authnz.QNameCDocWorkspaceDescriptor, Field_InitStartedAtMs, nowFunc().UnixMilli())
				info("updating initStartedAtMs:", updateWSDescrURL)

				if _, err := federation.Func(updateWSDescrURL, body, coreutils.WithAuthorizeBy(systemPrincipalToken_TargetApp), coreutils.WithDiscardResponse()); err != nil {
					er("failed to update initStartedAtMs:", err)
					return nil
				}

				wsKind := wsDescr.AsQName(authnz.Field_WSKind)
				ep := eps[s.App()]
				if wsError = buildWorkspace(wsDescr.AsString(field_TemplateName), ep, wsKind, federation, newWSID,
					targetAppQName, newWSName, systemPrincipalToken_TargetApp); wsError != nil {
					wsError = fmt.Errorf("workspace %s building: %w", wsDescr.AsString(field_TemplateName), wsError)
				}

				wsErrStr := ""
				if wsError != nil {
					wsErrStr = wsError.Error()
				}
				body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"%s","%s":%q,"%s":%d}}]}`,
					wsDescr.ID(), authnz.QNameCDocWorkspaceDescriptor, Field_InitError, wsErrStr, Field_InitCompletedAtMs, nowFunc().UnixMilli())
				if _, err = federation.Func(updateWSDescrURL, body, coreutils.WithAuthorizeBy(systemPrincipalToken_TargetApp), coreutils.WithDiscardResponse()); err != nil {
					er("failed to update initError+initCompletedAtMs:", err)
					return nil
				}
			} else if wsDescr.AsInt64(Field_InitCompletedAtMs) == 0 {
				info("initCompletedAtMs = 0. WS data init was interrupted")
				wsError = errors.New("workspace data initialization was interrupted")
				body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.QName":"%s","%s":%q,"%s":%d}}]}`,
					authnz.QNameCDocWorkspaceDescriptor, Field_InitError, wsError.Error(), Field_InitCompletedAtMs, nowFunc().UnixMilli())
				if _, err = federation.Func(updateWSDescrURL, body, coreutils.WithAuthorizeBy(systemPrincipalToken_TargetApp), coreutils.WithDiscardResponse()); err != nil {
					er("failed to update initError+initCompletedAtMs:", err)
					return nil
				}
			} else { // initCompletedAtMs > 0
				info("initStartedAtMs > 0 && initCompletedAtMs > 0")
				if initError := wsDescr.AsString(Field_InitError); len(initError) > 0 {
					wsError = errors.New(initError)
				}
			}

			if wsError == nil && wsPostInitFunc != nil {
				wsError = wsPostInitFunc(targetAppQName, wsDescr.AsQName(authnz.Field_WSKind), istructs.WSID(newWSID), federation, systemPrincipalToken_TargetApp)
			}

			ownerUpdated = updateOwner(rec, ownerApp, newWSID, wsError, systemPrincipalToken_OwnerApp, federation, info, er)
			return nil
		})
	}
}

func updateOwner(rec istructs.ICUDRow, ownerApp string, newWSID int64, err error, principalToken string, federation federation.IFederation,
	infoLogger func(args ...interface{}), errorLogger func(args ...interface{})) (ok bool) {
	ownerWSID := rec.AsInt64(Field_OwnerWSID)
	ownerID := rec.AsInt64(Field_OwnerID)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	updateOwnerURL := fmt.Sprintf("api/%s/%d/c.sys.CUD", ownerApp, ownerWSID)
	ownerQName := rec.AsString(Field_OwnerQName2)
	infoLogger(fmt.Sprintf("updating owner cdoc.%s at %s/%d: NewWSID=%d, WSError='%s'", ownerQName,
		ownerApp, ownerWSID, newWSID, errStr))
	body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"%s":%d,"%s":%q}}]}`,
		ownerID, authnz.Field_WSID, newWSID, authnz.Field_WSError, errStr)
	if _, err = federation.Func(updateOwnerURL, body, coreutils.WithAuthorizeBy(principalToken), coreutils.WithDiscardResponse()); err != nil {
		errorLogger("failed to updateOwner:", err)
	}
	return err == nil
}

func parseWSTemplateBLOBs(fsEntries []fs.DirEntry, blobIDs map[int64]map[string]struct{}, wsTemplateFS coreutils.EmbedFS) (blobs []coreutils.BLOBWorkspaceTemplateField, err error) {
	for _, ent := range fsEntries {
		switch ent.Name() {
		case "data.json", "provide.go":
		default:
			underscorePos := strings.Index(ent.Name(), "_")
			if underscorePos < 0 {
				return nil, fmt.Errorf("wrong blob file name format: %s", ent.Name())
			}
			recordIDStr := ent.Name()[:underscorePos]
			recordID, err := strconv.Atoi(recordIDStr)
			if err != nil {
				return nil, fmt.Errorf("wrong recordID in blob %s: %w", ent.Name(), err)
			}
			fieldName := strings.Replace(ent.Name()[underscorePos+1:], filepath.Ext(ent.Name()), "", -1)
			if len(fieldName) == 0 {
				return nil, fmt.Errorf("no fieldName in blob %s", ent.Name())
			}
			fieldNames, ok := blobIDs[int64(recordID)]
			if !ok {
				fieldNames = map[string]struct{}{}
				blobIDs[int64(recordID)] = fieldNames
			}
			if _, exists := fieldNames[fieldName]; exists {
				return nil, fmt.Errorf("recordID %d: blob for field %s is met again: %s", recordID, fieldName, ent.Name())
			}
			fieldNames[fieldName] = struct{}{}
			blobContent, err := wsTemplateFS.ReadFile(ent.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to read blob %s content: %w", ent.Name(), err)
			}
			blobs = append(blobs, coreutils.BLOBWorkspaceTemplateField{
				BLOBDesc: coreutils.BLOBDesc{
					Name:     ent.Name(),
					MimeType: filepath.Ext(ent.Name())[1:], // excluding dot
				},
				FieldName: fieldName,
				Content:   blobContent,
				RecordID:  istructs.RecordID(recordID),
			})
		}
	}
	return blobs, nil
}

func checkOrphanedBLOBs(blobIDs map[int64]map[string]struct{}, workspaceData []map[string]interface{}) error {
	orphanedBLOBRecordIDs := map[int64]struct{}{}
	for blobRecID := range blobIDs {
		orphanedBLOBRecordIDs[blobRecID] = struct{}{}
	}

	for _, record := range workspaceData {
		recIDIntf, ok := record[appdef.SystemField_ID]
		if !ok {
			return errors.New("record with missing sys.ID field is met")
		}
		recID := int64(recIDIntf.(float64))
		blobFields, ok := blobIDs[recID]
		if !ok {
			continue
		}
		delete(orphanedBLOBRecordIDs, recID)
		for blobField := range blobFields {
			if _, ok := record[blobField]; !ok {
				return fmt.Errorf("have blob for an unknown field for recordID %d: %s", recID, blobField)
			}
		}
	}

	if len(orphanedBLOBRecordIDs) > 0 {
		return fmt.Errorf("orphaned blobs met for ids %v", orphanedBLOBRecordIDs)
	}
	return nil
}

func ValidateTemplate(wsTemplateName string, ep extensionpoints.IExtensionPoint, wsKind appdef.QName) (wsBLOBs []coreutils.BLOBWorkspaceTemplateField, wsData []map[string]interface{}, err error) {
	if len(wsTemplateName) == 0 {
		return nil, nil, nil
	}
	epWSTemplates := ep.ExtensionPoint(EPWSTemplates)
	epWSKindTemplatesIntf, ok := epWSTemplates.Find(wsKind)
	if !ok {
		return nil, nil, fmt.Errorf("no templates for workspace kind %s", wsKind.String())
	}
	epWSKindTemplates := epWSKindTemplatesIntf.(extensionpoints.IExtensionPoint)
	wsTemplateFSIntf, ok := epWSKindTemplates.Find(wsTemplateName)
	if !ok {
		return nil, nil, fmt.Errorf("unknown workspace template name %s for workspace kind %s", wsTemplateName, wsKind.String())
	}
	wsTemplateFS := wsTemplateFSIntf.(coreutils.EmbedFS)
	fsEntries, err := wsTemplateFS.ReadDir(".")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read dir content: %w", err)
	}
	wsData = []map[string]interface{}{}
	dataBytes, err := wsTemplateFS.ReadFile("data.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data.json: %w", err)
	}
	if err := json.Unmarshal(dataBytes, &wsData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal data.json: %w", err)
	}

	// check blob entries
	//          newBLOBID   fieldName
	blobIDs := map[int64]map[string]struct{}{}
	wsBLOBs, err = parseWSTemplateBLOBs(fsEntries, blobIDs, wsTemplateFS)
	if err != nil {
		return nil, nil, err
	}
	if err := checkOrphanedBLOBs(blobIDs, wsData); err != nil {
		return nil, nil, err
	}
	return wsBLOBs, wsData, nil
}
