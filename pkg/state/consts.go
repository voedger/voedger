/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

var (
	View           = appdef.NewQName(appdef.SysPackage, "View")
	Record         = appdef.NewQName(appdef.SysPackage, "Record")
	WLog           = appdef.NewQName(appdef.SysPackage, "WLog")
	PLog           = appdef.NewQName(appdef.SysPackage, "PLog")
	Http           = appdef.NewQName(appdef.SysPackage, "Http")
	SendMail       = appdef.NewQName(appdef.SysPackage, "SendMail")
	AppSecret      = appdef.NewQName(appdef.SysPackage, "AppSecret")
	RequestSubject = appdef.NewQName(appdef.SysPackage, "RequestSubject")
	Result         = appdef.NewQName(appdef.SysPackage, "Result")
	Event          = appdef.NewQName(appdef.SysPackage, "Event")
	CommandContext = appdef.NewQName(appdef.SysPackage, "CommandContext")
)

const (
	S_GET       = 1
	S_GET_BATCH = 2
	S_READ      = 4
	S_INSERT    = 8
	S_UPDATE    = 16
)

const (
	Field_Url                           = "Url"
	Field_Method                        = "Method"
	Field_Header                        = "Header"
	Field_Offset                        = "Offset"
	Field_Error                         = "Error"
	Field_ErrStr                        = "ErrStr"
	Field_QNameFromParams               = "QNameFromParams"
	Field_ValidEvent                    = "ValidEvent"
	Field_QName                         = "QName"
	Field_ArgumentObject                = "ArgumentObject"
	Field_ArgumentUnloggedObject        = "ArgumentUnloggedObject"
	Field_Synced                        = "Synced"
	Field_Count                         = "Count"
	Field_ID                            = "ID"
	Field_From                          = "From"
	Field_To                            = "To"
	Field_CC                            = "CC"
	Field_BCC                           = "BCC"
	Field_Subject                       = "Subject"
	Field_Body                          = "Body"
	Field_PartitionID                   = "PartitionID"
	Field_WSID                          = "WSID"
	Field_HTTPClientTimeoutMilliseconds = "HTTPClientTimeoutMilliseconds"
	Field_Singleton                     = "Singleton"
	Field_Secret                        = "Secret"
	Field_RegisteredAt                  = "RegisteredAt"
	Field_DeviceID                      = "DeviceID"
	Field_SyncedAt                      = "SyncedAt"
	Field_WLogOffset                    = "WLogOffset"
	Field_Workspace                     = "Workspace"
	Field_Username                      = "Username"
	Field_Password                      = "Password"
	Field_Host                          = "Host"
	Field_Port                          = "Port"
	Field_StatusCode                    = "StatusCode"
	Field_Kind                          = "Kind"
	Field_ProfileWSID                   = "ProfileWSID"
	Field_CUDs                          = "CUDs"
	Field_IsNew                         = "IsNew"
	Field_Name                          = "Name"
	Field_Token                         = "Token"
)

const (
	ColOffset                             = "offs"
	defaultHTTPClientTimeout              = 20_000 * time.Millisecond
	httpStorageKeyBuilderStringerSliceCap = 3
)

var (
	emptyApplyBatchItem = ApplyBatchItem{}
)
