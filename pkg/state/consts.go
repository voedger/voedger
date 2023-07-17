/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

var (
	ViewRecordsStorage = appdef.NewQName(appdef.SysPackage, "ViewRecordsStorage")
	RecordsStorage     = appdef.NewQName(appdef.SysPackage, "RecordsStorage")
	WLogStorage        = appdef.NewQName(appdef.SysPackage, "WLogStorage")
	PLogStorage        = appdef.NewQName(appdef.SysPackage, "PLogStorage")
	HTTPStorage        = appdef.NewQName(appdef.SysPackage, "HTTPStorage")
	SendMailStorage    = appdef.NewQName(appdef.SysPackage, "SendMailStorage")
	AppSecretsStorage  = appdef.NewQName(appdef.SysPackage, "AppSecretsStorage")
	SubjectStorage     = appdef.NewQName(appdef.SysPackage, "SubjectStorage")
	CmdResultStorage   = appdef.NewQName(appdef.SysPackage, "CmdResultStorage")
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
