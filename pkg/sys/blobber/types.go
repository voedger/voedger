/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobber

import "github.com/voedger/voedger/pkg/istructs"

type BLOB struct {
	FieldName string
	Content   []byte
	Name      string
	MimeType  string
}

type StoredBLOB struct {
	BLOB
	RecordID istructs.RecordID
}
