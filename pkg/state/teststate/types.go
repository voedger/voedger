/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */
package teststate

import (
	"fmt"
	"io"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

type TestWorkspace struct {
	WorkspaceDescriptor string
	WSID                istructs.WSID
}

type TestViewValue struct {
	wsid istructs.WSID
	vr   istructs.IViewRecords
	Key  istructs.IKeyBuilder
	Val  istructs.IValueBuilder
}

type HttpRequest struct {
	Timeout time.Duration
	Method  string
	URL     string
	Body    io.Reader
	Headers map[string]string
}

type HttpResponse struct {
	Status  int
	Body    []byte
	Headers map[string][]string
}

type recordItem struct {
	entity      IFullQName
	qName       appdef.QName
	isSingleton bool
	isView      bool
	isNew       bool
	id          istructs.RecordID
	// fileReference is used to store the file reference in test where the record was added
	fileReference string
	keyValueList  []any
}

type intentItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
	isNew bool
}

func (ri recordItem) toIRecord() istructs.IRecord {
	kvMap, err := parseKeyValues(ri.keyValueList)
	if err != nil {
		panic(fmt.Errorf("recordItem.toObject: %w", err))
	}

	return &coreutils.TestObject{
		Id:     ri.id,
		Name:   appdef.NewQName(ri.entity.PkgPath(), ri.entity.Entity()),
		Data:   kvMap,
		IsNew_: ri.isNew,
	}
}
