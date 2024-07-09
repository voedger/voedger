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
	entity       appdef.IFullQName
	isSingleton  bool
	isNew        bool
	id           int
	keyValueList []any
}

type intentItem struct {
	key   istructs.IStateKeyBuilder
	value istructs.IStateValueBuilder
	isNew bool
}

func (i *intentItem) String() string {
	//storageName := i.key.Storage().String()
	if i.key == nil {
		return "Key: empty"
	}

	storage := i.key.Storage()
	if i.key == nil {
		return "Key: empty"
	}

	entity := i.key.Entity()

	fmt.Println("Storage: ", storage)
	fmt.Println("Entity: ", entity)

	////return fmt.Sprintf("Storage: %s, entity: %s, IsNew: %v", storageName, entity, i.isNew)
	//return fmt.Sprintf("{Storage: %s, IsNew: %v},", storageName, i.isNew)
	return fmt.Sprintf(
		"Storage: %s, entity: %s, IsNew: %v",
		i.key.Storage().String(),
		i.key.Entity().String(),
		i.isNew,
	)
}
