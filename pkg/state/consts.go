/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/sys"
)

var (
	// Deprecated: use sys.Storage_View instead
	View = sys.Storage_View

	// Deprecated: use sys.Storage_Record instead
	Record = sys.Storage_Record

	// Deprecated: use sys.Storage_WLog instead
	WLog = sys.Storage_WLog

	// Deprecated: use sys.Storage_HTTP instead
	HTTP = sys.Storage_HTTP

	// Deprecated: use sys.Storage_SendMail instead
	SendMail = sys.Storage_SendMail

	// Deprecated: use sys.Storage_AppSecret instead
	AppSecret = sys.Storage_AppSecret

	// Deprecated: use sys.Storage_RequestSubject instead
	RequestSubject = sys.Storage_RequestSubject

	// Deprecated: use sys.Storage_Result instead
	Result = sys.Storage_Result

	// Deprecated: use sys.Storage_Event instead
	Event = sys.Storage_Event

	// Deprecated: use sys.Storage_CommandContext instead
	CommandContext = sys.Storage_CommandContext

	// Deprecated: use sys.Storage_QueryContext instead
	QueryContext = sys.Storage_QueryContext

	// Deprecated: use sys.Storage_Response instead
	Response = sys.Storage_Response

	// Deprecated: use sys.Storage_FederationCommand instead
	FederationCommand = sys.Storage_FederationCommand

	// Deprecated: use sys.Storage_FederationBlob instead
	FederationBlob = sys.Storage_FederationBlob

	// Deprecated: use sys.Storage_Uniq instead
	Uniq = sys.Storage_Uniq
)

const (
	Field_URL                           = sys.Storage_HTTP_Field_URL                              // Deprecated: use sys.Storage_HTTP_Field_URL instead
	Field_Method                        = sys.Storage_HTTP_Field_Method                           // Deprecated: use sys.Storage_HTTP_Field_Method instead
	Field_Header                        = sys.Storage_HTTP_Field_Header                           // Deprecated: use sys.Storage_HTTP_Field_Header instead
	Field_Offset                        = "Offset"                                                // Deprecated: use sys.Storage_WLog_Field_Offset or sys.Storage_Event_Field_Offset instead
	Field_Error                         = sys.Storage_Event_Field_Error                           // Deprecated: use sys.Storage_Event_Field_Error instead
	Field_ErrStr                        = sys.Storage_Event_Field_ErrStr                          // Deprecated: use sys.Storage_Event_Field_ErrStr instead
	Field_QNameFromParams               = sys.Storage_Event_Field_QNameFromParams                 // Deprecated: use sys.Storage_Event_Field_QNameFromParams instead
	Field_ValidEvent                    = sys.Storage_Event_Field_ValidEvent                      // Deprecated: use sys.Storage_Event_Field_ValidEvent instead
	Field_QName                         = sys.Storage_Event_Field_QName                           // Deprecated: use sys.Storage_Event_Field_QName instead
	Field_ArgumentObject                = "ArgumentObject"                                        // Deprecated: use sys.Storage_Event_Field_ArgumentObject, Storage_CommandContext_Field_ArgumentObject, Storage_QueryContext_Field_ArgumentObject, Storage_WLog_Field_ArgumentObject instead
	Field_ArgumentUnloggedObject        = sys.Storage_CommandContext_Field_ArgumentUnloggedObject // Deprecated: use sys.Storage_CommandContext_Field_ArgumentUnloggedObject instead
	Field_Synced                        = sys.Storage_Event_Field_Synced                          // Deprecated: use sys.Storage_Event_Field_Synced instead
	Field_Count                         = sys.Storage_WLog_Field_Count                            // Deprecated: use sys.Storage_WLog_Field_Count instead
	Field_ID                            = "ID"                                                    // Deprecated: use sys.Storage_Record_Field_ID or sys.Storage_Uniq_Field_ID instead
	Field_From                          = sys.Storage_SendMail_Field_From                         // Deprecated: use sys.Storage_SendMail_Field_From instead
	Field_To                            = sys.Storage_SendMail_Field_To                           // Deprecated: use sys.Storage_SendMail_Field_To instead
	Field_CC                            = sys.Storage_SendMail_Field_CC                           // Deprecated: use sys.Storage_SendMail_Field_CC instead
	Field_BCC                           = sys.Storage_SendMail_Field_BCC                          // Deprecated: use sys.Storage_SendMail_Field_BCC instead
	Field_Subject                       = sys.Storage_SendMail_Field_Subject                      // Deprecated: use sys.Storage_SendMail_Field_Subject instead
	Field_Host                          = sys.Storage_SendMail_Field_Host                         // Deprecated: use sys.Storage_SendMail_Field_Host instead
	Field_Port                          = sys.Storage_SendMail_Field_Port                         // Deprecated: use sys.Storage_SendMail_Field_Port instead
	Field_Username                      = sys.Storage_SendMail_Field_Username                     // Deprecated: use sys.Storage_SendMail_Field_Username instead
	Field_Password                      = sys.Storage_SendMail_Field_Password                     // Deprecated: use sys.Storage_SendMail_Field_Password instead
	Field_Command                       = sys.Storage_FederationCommand_Field_Command             // Deprecated: use sys.Storage_FederationCommand_Field_Command instead
	Field_Body                          = "Body"                                                  // Deprecated: use sys.Storage_FederationCommand_Field_Body, sys.Storage_FederationBlob_Field_Body instead, sys.Storage_HTTP_Field_Body instead
	Field_WSID                          = "WSID"                                                  // Deprecated: use sys.Storage_Record_Field_WSID, sys.Storage_WLog_Field_WSID, sys.Storage_FederationCommand_Field_WSID, sys.Storage_FederationBlob_Field_WSID, sys.Storage_View_Field_WSID instead
	Field_HTTPClientTimeoutMilliseconds = sys.Storage_HTTP_Field_HTTPClientTimeoutMilliseconds    // Deprecated: use sys.Storage_HTTP_Field_HTTPClientTimeoutMilliseconds instead
	Field_Singleton                     = sys.Storage_Record_Field_Singleton                      // Deprecated: use sys.Storage_Record_Field_Singleton instead
	Field_IsSingleton                   = sys.Storage_Record_Field_IsSingleton                    // Deprecated: use sys.Storage_Record_Field_IsSingleton instead
	Field_Secret                        = sys.Storage_AppSecretField_Secret                       // Deprecated: use sys.Storage_AppSecretField_Secret instead
	Field_RegisteredAt                  = "RegisteredAt"                                          // Deprecated: use sys.Storage_WLog_Field_RegisteredAt, sys.Storage_Event_Field_RegisteredAt instead
	Field_DeviceID                      = "DeviceID"                                              // Deprecated: use sys.Storage_WLog_Field_DeviceID, sys.Storage_Event_Field_DeviceID instead
	Field_SyncedAt                      = "SyncedAt"                                              // Deprecated: use sys.Storage_WLog_Field_SyncedAt, sys.Storage_Event_Field_SyncedAt instead
	Field_WLogOffset                    = "WLogOffset"                                            // Deprecated: use sys.Storage_Event_Field_WLogOffset, sys.Storage_CommandContext_Field_WLogOffset instead
	Field_Workspace                     = "Workspace"                                             // Deprecated: use sys.Storage_Event_Field_Workspace, sys.Storage_CommandContext_Field_Workspace, sys.Storage_QueryContext_Field_Workspace instead
	Field_StatusCode                    = "StatusCode"                                            // Deprecated: use sys.Storage_HTTP_Field_StatusCode, Storage_FederationCommand_Field_StatusCode, Storage_Response_Field_StatusCode instead
	Field_Kind                          = sys.Storage_RequestSubject_Field_Kind                   // Deprecated: use sys.Storage_RequestSubject_Field_Kind instead
	Field_ProfileWSID                   = sys.Storage_RequestSubject_Field_ProfileWSID            // Deprecated: use sys.Storage_RequestSubject_Field_ProfileWSID instead
	Field_Name                          = sys.Storage_RequestSubject_Field_Name                   // Deprecated: use sys.Storage_RequestSubject_Field_Name instead
	Field_CUDs                          = "CUDs"                                                  // Deprecated: use sys.Storage_WLog_Field_CUDs, sys.Storage_Event_Field_CUDs instead
	Field_IsNew                         = sys.CUDs_Field_IsNew                                    // Deprecated: use Storage_WLog_Field_CUDs_IsNew, Storage_Event_Field_CUDs_IsNew instead
	Field_Token                         = "Token"                                                 // Deprecated: use sys.Storage_FederationBlob_Field_Token, Storage_FederationCommand_Field_Token, Storage_RequestSubject_Field_Token instead
	Field_Owner                         = "Owner"                                                 // Deprecated: use sys.Storage_FederationBlob_Field_Owner, sys.Storage_FederationCommand_Field_Owner instead
	Field_AppName                       = "AppName"                                               // Deprecated: use sys.Storage_FederationBlob_Field_AppName, sys.Storage_FederationCommand_Field_AppName instead
	Field_ExpectedCodes                 = "ExpectedCodes"                                         // Deprecated: use sys.Storage_FederationBlob_Field_ExpectedCodes, sys.Storage_FederationCommand_Field_ExpectedCodes instead
	Field_BlobID                        = sys.Storage_FederationBlob_Field_BlobID                 // Deprecated: use sys.Storage_FederationBlob_Field_BlobID instead
	Field_NewIDs                        = sys.Storage_FederationCommand_Field_NewIDs              // Deprecated: use sys.Storage_FederationCommand_Field_NewIDs instead
	Field_Result                        = sys.Storage_FederationCommand_Field_Result              // Deprecated: use sys.Storage_FederationCommand_Field_Result instead
	Field_ErrorMessage                  = sys.Storage_Response_Field_ErrorMessage                 // Deprecated: use sys.Storage_Response_Field_ErrorMessage instead
)

const (
	ColOffset = "offs"
)
