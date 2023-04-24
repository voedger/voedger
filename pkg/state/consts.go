/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"time"

	"github.com/voedger/voedger/pkg/schemas"
)

var (
	ViewRecordsStorage = schemas.NewQName(schemas.SysPackage, "ViewRecordsStorage")
	RecordsStorage     = schemas.NewQName(schemas.SysPackage, "RecordsStorage")
	WLogStorage        = schemas.NewQName(schemas.SysPackage, "WLogStorage")
	PLogStorage        = schemas.NewQName(schemas.SysPackage, "PLogStorage")
	HTTPStorage        = schemas.NewQName(schemas.SysPackage, "HTTPStorage")
	SendMailStorage    = schemas.NewQName(schemas.SysPackage, "SendMailStorage")
	AppSecretsStorage  = schemas.NewQName(schemas.SysPackage, "AppSecretsStorage")
	SubjectStorage     = schemas.NewQName(schemas.SysPackage, "SubjectStorage")
)

const (
	S_GET_BATCH = 1
	S_READ      = 2
	S_INSERT    = 4
	S_UPDATE    = 8
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
