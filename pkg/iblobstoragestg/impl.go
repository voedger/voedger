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

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type bStorageType struct {
	appStorage BlobAppStoragePtr
	time       coreutils.ITime
}

// key does not cotain the bucket number
func (b *bStorageType) writeBLOB(ctx context.Context, blobKey []byte, descr iblobstorage.DescrType, reader io.Reader, quoter iblobstorage.WQuoterType, duration iblobstorage.DurationType) (err error) {
	var (
		bytesRead    uint64
		cCol         uint64
		bucketNumber uint64 = 1
		pKeyBuf      []byte
	)
	state := iblobstorage.BLOBState{
		Descr:     descr,
		StartedAt: istructs.UnixMilli(b.time.Now().UnixMilli()),
		Status:    iblobstorage.BLOBStatus_InProcess,
		Duration:  duration,
	}

	err = b.writeState(blobKey, &state)
	if err != nil {
		return err
	}

	buf := make([]byte, 0, chunkSize)

	pKeyBuf = newKeyWithBucketNumber(blobKey, bucketNumber)

	// if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, bucketNumber); err != nil {
	// 	return
	// }

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
			bytesRead += uint64(len(buf))
			isAllowed, err := quoter(uint64(len(buf)))
			if err != nil {
				return err
			}
			if !isAllowed {
				err = iblobstorage.ErrBLOBSizeQuotaExceeded
				break
			}
			// if bytesRead > uint64(maxSize) {
			// 	err = iblobstorage.ErrBLOBSizeQuotaExceeded
			// 	break
			// }
			if err = b.writeChunk(pKeyBuf, cCol, &bucketNumber, bytesRead, &buf); err != nil {
				break
			}
			cCol++
		}
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	state.FinishedAt = istructs.UnixMilli(b.time.Now().UnixMilli())
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

func (b *bStorageType) WriteBLOB(ctx context.Context, key *bytes.Buffer, descr iblobstorage.DescrType, reader io.Reader, maxSize iblobstorage.BLOBMaxSizeType) (err error) {

}

func (b *bStorageType) writeChunk(pKeyBuf *bytes.Buffer, cCol uint64, bucketNumber *uint64, bytesRead uint64, buf *[]byte) (err error) {
	var cColBuf *bytes.Buffer
	if bytesRead > chunkSize*bucketSize*(*bucketNumber) {
		*bucketNumber++
		binary.LittleEndian.PutUint64(pKeyBuf.Bytes()[keyLength:], *bucketNumber)
	}
	if cColBuf, err = createKey(cCol); err != nil {
		return
	}
	err = (*(b.appStorage)).Put(pKeyBuf.Bytes(), cColBuf.Bytes(), *buf)
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

func (b *bStorageType) WriteTempBLOB(ctx context.Context, key iblobstorage.TempKeyType, descr iblobstorage.DescrType, reader io.Reader,
	duration iblobstorage.DurationType, quoter iblobstorage.WQuoterType) error {

	return nil
}

// warning: the bucket number must be the last value!
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

func (b *bStorageType) writeState(blobKey []byte, state *iblobstorage.BLOBState) (err error) {
	var (
		value   []byte
		pKeyBuf []byte
		cColBuf *bytes.Buffer
	)
	pKeyBuf = newKeyWithBucketNumber(blobKey, zeroBucket)
	// if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, zeroBucket); err != nil {
	// 	return
	// }
	if cColBuf, err = createKey(zeroCcCol); err != nil {
		return
	}
	value, err = json.Marshal(state)
	if err != nil {
		// notest
		return err
	}
	return (*(b.appStorage)).Put(
		pKeyBuf,
		cColBuf.Bytes(),
		value,
	)
}

func newKeyWithBucketNumber(blobKey []byte, bucket uint64) []byte {
	res := make([]byte, len(blobKey)+8)
	copy(res, blobKey)
	binary.LittleEndian.PutUint64(res[len(res)-8:], bucket)
	return res
}
