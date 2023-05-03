/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import "github.com/voedger/voedger/pkg/istructs"

type cud struct {
	Fields map[string]interface{} `json:"fields"`
}
type cuds struct {
	Cuds []cud `json:"cuds"`
}

// template recordID -> template fieldName -> uploaded blobID to set to fieldName
type blobsMap map[int64]map[string]int64

type BLOB struct {
	RecordID  istructs.RecordID
	FieldName string
	Content   []byte
	Name      string
	MimeType  string
}
