/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type bStorageType struct {
	appStorage BlobAppStoragePtr
	now        coreutils.TimeFunc
}

func (b *bStorageType) WriteBLOB(ctx context.Context, key iblobstorage.KeyType, descr iblobstorage.DescrType, reader io.Reader, maxSize int64) (err error) {
	var (
		bytesRead    int64
		cCol         uint64
		bucketNumber uint64 = 1
		pKeyBuf      *bytes.Buffer
	)
	state := iblobstorage.BLOBState{
		Descr:     descr,
		StartedAt: istructs.UnixMilli(b.now().UnixMilli()),
		Status:    iblobstorage.BLOBStatus_InProcess,
	}

	err = b.writeState(key, &state)
	if err != nil {
		return err
	}

	buf := make([]byte, 0, chunkSize)

	if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, bucketNumber); err != nil {
		return
	}

	for err == nil {
		var chunkBytes int
		if ctx.Err() != nil {
			state.Error = ctx.Err().Error()
			state.Status = iblobstorage.BLOBStatus_Unknown
			break
		}
		chunkBytes, err = reader.Read(buf[:cap(buf)])

		if chunkBytes > 0 {
			buf = buf[:chunkBytes]
			bytesRead += int64(len(buf))
			if bytesRead > maxSize {
				err = iblobstorage.ErrBLOBSizeQuotaExceeded
				break
			}
			if err = b.writeChunk(pKeyBuf, cCol, &bucketNumber, bytesRead, &buf); err != nil {
				break
			}
			cCol++
		}
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	state.FinishedAt = istructs.UnixMilli(b.now().UnixMilli())
	state.Status = iblobstorage.BLOBStatus_Completed
	state.Size = bytesRead
	if err != nil {
		state.Error = err.Error()
		state.Status = iblobstorage.BLOBStatus_Unknown
	}
	if errStatus := b.writeState(key, &state); errStatus != nil {
		err = errStatus
	}
	return err
}

func (b *bStorageType) writeChunk(pKeyBuf *bytes.Buffer, cCol uint64, bucketNumber *uint64, bytesRead int64, buf *[]byte) (err error) {
	var cColBuf *bytes.Buffer
	if uint64(bytesRead) > chunkSize*bucketSize*(*bucketNumber) {
		*bucketNumber++
		if errBucket := addBucket(pKeyBuf, *bucketNumber); errBucket != nil {
			err = errBucket
			return
		}
	}
	if cColBuf, err = createKey(cCol); err != nil {
		return
	}
	err = (*(b.appStorage)).Put(pKeyBuf.Bytes(), cColBuf.Bytes(), *buf)
	return
}

func addBucket(pKeyBuf *bytes.Buffer, bucketNumber uint64) (err error) {
	bucketKey := bytes.NewBuffer(pKeyBuf.Bytes()[:keyLength])
	err = binary.Write(bucketKey, binary.LittleEndian, bucketNumber)
	return
}

func (b *bStorageType) ReadBLOB(ctx context.Context, key iblobstorage.KeyType, stateWriter func(state iblobstorage.BLOBState) error, writer io.Writer) (err error) {
	var (
		bucketNumber uint64 = 1
		isFound      bool
		state        iblobstorage.BLOBState
		pKeyBuf      *bytes.Buffer
	)
	if stateWriter != nil {
		if err := b.readState(key, &state); err != nil {
			return err
		}
		isFound = true
		if err := stateWriter(state); err != nil {
			return err
		}
	}
	if writer != nil {
		for ctx.Err() == nil {
			if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, bucketNumber); err != nil {
				return err
			}
			var n int
			err = (*(b.appStorage)).Read(ctx, pKeyBuf.Bytes(), nil, nil,
				func(ccols []byte, viewRecord []byte) (err error) {
					isFound = true
					n, err = writer.Write(viewRecord)
					return err
				})
			if err != nil {
				logger.Error(fmt.Sprintf("failed to send a BLOB chunk: id %d, appID %d, wsid %d: %s", key.ID, key.AppID, key.WSID, err.Error()))
				break
			}
			if n > 0 {
				bucketNumber++
			} else {
				break
			}
		}
	}

	if !isFound && err == nil {
		err = iblobstorage.ErrBLOBNotFound
	}
	return err
}

func (b *bStorageType) QueryBLOBState(ctx context.Context, key iblobstorage.KeyType) (state iblobstorage.BLOBState, err error) {
	err = b.ReadBLOB(ctx, key,
		func(blobState iblobstorage.BLOBState) (err error) {
			state = blobState
			return nil
		},
		nil)
	return
}

func createKey(columns ...interface{}) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	for _, col := range columns {
		switch v := col.(type) {
		case nil:
			return nil, fmt.Errorf("key column with type «%s» is missed: %w", reflect.ValueOf(col).Type(), errPKeyCreateError)
		case appType, istructs.ClusterAppID, istructs.WSID, istructs.RecordID, uint64:
			if errWrite := binary.Write(buf, binary.LittleEndian, v); errWrite != nil {
				err = errWrite
				return nil, fmt.Errorf("error create key: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported data type %s:  %w", reflect.ValueOf(col).Type(), errPKeyCreateError)
		}
	}
	return buf, nil
}

func (b *bStorageType) readState(key iblobstorage.KeyType, state *iblobstorage.BLOBState) (err error) {
	var (
		currentState []byte
		ok           bool
		pKeyBuf      *bytes.Buffer
		cColBuf      *bytes.Buffer
	)
	if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, zeroBucket); err != nil {
		return
	}
	if cColBuf, err = createKey(zeroCcCol); err != nil {
		return
	}
	ok, err = (*(b.appStorage)).Get(
		pKeyBuf.Bytes(),
		cColBuf.Bytes(),
		&currentState)
	if ok {
		return json.Unmarshal(currentState, &state)
	}
	if err == nil {
		err = iblobstorage.ErrBLOBNotFound
	}
	return err
}

func (b *bStorageType) writeState(key iblobstorage.KeyType, s interface{}) (err error) {
	var (
		value   []byte
		pKeyBuf *bytes.Buffer
		cColBuf *bytes.Buffer
	)
	if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, zeroBucket); err != nil {
		return
	}
	if cColBuf, err = createKey(zeroCcCol); err != nil {
		return
	}
	value, err = json.Marshal(s)
	if err != nil {
		return fmt.Errorf("error write meta information of blob appType: %d, wsid: %d, blobid: %d,  error: %w - marshal to JSON failed ",
			key.AppID, key.WSID, key.ID, err)
	}
	return (*(b.appStorage)).Put(
		pKeyBuf.Bytes(),
		cColBuf.Bytes(),
		value)
}
