/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package istorage

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
)

// Same as IAppStructsProvider, called per request or frequently inside services
// implemented in istorageimpl
type IAppStorageProvider interface {
	// converts AppQname to string and calls internal IAppStorageFactory.AppStorage
	// storage for a brand new app is queried and failed to create the storage for it -> storage init error is persisted and returned until admin is handle with that incident
	// could return ErrStorageNotFound, ErrStorageExistsAlready
	AppStorage(appName appdef.AppQName) (storage IAppStorage, err error)
}

// implemented by a certain driver
type IAppStorageFactory interface {
	// returns IAppStorage for an existing storage
	// returns ErrStorageNotFound
	AppStorage(appName SafeAppName) (storage IAppStorage, err error)

	// creates new storage
	// returns ErrStorageExistsAlready
	Init(appName SafeAppName) error
}

type IAppStorage interface {
	// cCols - clustering columns
	// len(cCols) may be 0 (nil or empty array)
	// Clustering columns MUST be filled from left to right
	// Example: PRIMARY KEY(wsid, qname_id, id)
	//   Clusterting columns: qname_id, id
	//   qname_id bytes must be written first, then id bytes
	// @ConcurrentAccess
	Put(pKey []byte, cCols []byte, value []byte) (err error)

	PutBatch(items []BatchItem) (err error)

	// len(cCols) may be 0, in this case the record which was written with zero len(cCols) will be returned
	// ok == false means that viewrecord does not exist
	// @ConcurrentAccess
	Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)

	// get and appends result to items[i].Data
	// items[i].Ok==false means record is not found
	// items[i].Ok & Data are undefined in case of error
	GetBatch(pKey []byte, items []GetBatchItem) (err error)

	// startCCols can be empty (nil or zero len), in this case reads from start of partition.
	// finishCCols can be empty (nil or zero len) too. In this case reads to the end of partition
	// @ConcurrentAccess
	Read(ctx context.Context, pKey []byte, startCCols, finishCCols []byte, cb ReadCallback) (err error)
}

// ccols and viewRecord are temporary internal values, must NOT be changed
type ReadCallback func(ccols []byte, viewRecord []byte) (err error)

type BatchItem struct {
	PKey  []byte
	CCols []byte
	Value []byte
}

type GetBatchItem struct {
	CCols []byte
	Ok    bool
	Data  *[]byte
}
