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
func (b *bStorageType) writeBLOB(ctx context.Context, blobKey []byte, descr iblobstorage.DescrType, reader io.Reader, quoter iblobstorage.WLimiterType, duration iblobstorage.DurationType) (err error) {
	var (
		bytesRead      uint64
		chunkNumber    uint64
		bucketNumber   uint64 = 1
		pKeyWithBucket []byte
	)
	state := iblobstorage.BLOBState{
		Descr:     descr,
		StartedAt: istructs.UnixMilli(b.time.Now().UnixMilli()),
		Status:    iblobstorage.BLOBStatus_InProcess,
		Duration:  duration,
	}

	pKeyState, cColState, err := getStateKeys(blobKey)
	if err != nil {
		// notest
		return err
	}

	err = b.writeState(&state, pKeyState, cColState)
	if err != nil {
		return err
	}

	chunkBuf := make([]byte, 0, chunkSize)

	pKeyWithBucket = newKeyWithBucketNumber(blobKey, bucketNumber)

	cCol := make([]byte, uint64Size)

	for ctx.Err() == nil && err == nil {
		var currentChunkSize int
		currentChunkSize, err = reader.Read(chunkBuf[:cap(chunkBuf)])
		if currentChunkSize > 0 {
			chunkBuf = chunkBuf[:currentChunkSize]
			bytesRead += uint64(len(chunkBuf))
			if err = quoter(uint64(len(chunkBuf))); err != nil {
				break
			}
			if bytesRead > chunkSize*bucketSize*bucketNumber {
				bucketNumber++
				pKeyWithBucket = mutateBucketNumber(pKeyWithBucket, bucketNumber)
			}
			cCol = mutateChunkNumber(cCol, chunkNumber)
			if err = (*(b.appStorage)).Put(pKeyWithBucket, cCol, chunkBuf); err != nil {
				break
			}
			chunkNumber++
		}
	}

	if ctx.Err() != nil && err == nil {
		// err has priority over ctx.Err
		err = ctx.Err()
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

	if errStatus := b.writeState(&state, pKeyState, cColState); errStatus != nil {
		if err == nil {
			// err as priority over errStatus
			return errStatus
		}
		logger.Error("failed to write blob state: " + errStatus.Error())
	}
	return err
}

func mutateChunkNumber(key []byte, chunkNumber uint64) (mutadedKey []byte) {
	binary.LittleEndian.PutUint64(key, chunkNumber)
	return key
}

func mutateBucketNumber(key []byte, bucketNumber uint64) (mutatedKey []byte) {
	binary.LittleEndian.PutUint64(key[len(key)-uint64Size:], bucketNumber)
	return key
}

func getStateKeys(blobKey []byte) (pKeyState, cColState []byte, err error) {
	pKeyState = newKeyWithBucketNumber(blobKey, zeroBucket)
	cColStateBuf, err := createKey(zeroCCol)
	if err != nil {
		// notest
		return nil, nil, err
	}
	cColState = cColStateBuf.Bytes()
	return
}

func (b *bStorageType) WriteBLOB(ctx context.Context, key iblobstorage.PersistentBLOBKeyType, descr iblobstorage.DescrType, reader io.Reader, limiter iblobstorage.WLimiterType) (err error) {
	return b.writeBLOB(ctx, key.Bytes(), descr, reader, limiter, 0)
}

func (b *bStorageType) WriteTempBLOB(ctx context.Context, key iblobstorage.TempBLOBKeyType, descr iblobstorage.DescrType, reader io.Reader, limiter iblobstorage.WLimiterType, duration iblobstorage.DurationType) (err error) {
	return b.writeBLOB(ctx, key.Bytes(), descr, reader, limiter, duration)
}

func (b *bStorageType) ReadBLOB(ctx context.Context, blobKey iblobstorage.IBLOBKey, stateCallback func(state iblobstorage.BLOBState) error, writer io.Writer) (err error) {
	blobKeyBytes := blobKey.Bytes()
	pKeyState, cColState, err := getStateKeys(blobKeyBytes)
	if err != nil {
		// notest
		return err
	}
	state, stateExists, err := b.readState(pKeyState, cColState)
	if err != nil {
		return err
	}

	if stateExists && stateCallback != nil {
		if err = stateCallback(state); err != nil {
			return err
		}
	}

	bucketNumber := uint64(1)
	pKeyWithBucket := newKeyWithBucketNumber(blobKeyBytes, bucketNumber)

	blobDataExists := false
	for ctx.Err() == nil && err == nil {
		bucketExists := false
		err = (*(b.appStorage)).Read(ctx, pKeyWithBucket, nil, nil,
			func(ccols []byte, viewRecord []byte) (err error) {
				bucketExists = true
				blobDataExists = true
				if writer != nil {
					_, err = writer.Write(viewRecord)
				}
				return err
			})

		if bucketExists {
			if bucketNumber == 1 && !stateExists {
				return fmt.Errorf("%w: BLOB state exists but the corresponding first bucket does not exist", iblobstorage.ErrBLOBCorrupted)
			}
			if writer == nil {
				break
			}
			bucketNumber++
			pKeyWithBucket = mutateBucketNumber(pKeyWithBucket, bucketNumber)
		} else {
			if bucketNumber == 1 && stateExists {
				return fmt.Errorf("%w: bucket 1 exists but the corresponding BLOB state does not exist", iblobstorage.ErrBLOBCorrupted)
			}
			break
		}
	}

	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !stateExists && !blobDataExists {
		return iblobstorage.ErrBLOBNotFound
	}

	return nil
}

// func (b *bStorageType) ReadBLOB(ctx context.Context, blobKey iblobstorage.IBLOBKey, stateCallback func(state iblobstorage.BLOBState) error, writer io.Writer) (err error) {
// 	var (
// 		bucketNumber uint64 = 1
// 		isFound      bool
// 		state        iblobstorage.BLOBState
// 		pKeyBuf      *bytes.Buffer
// 	)
// 	if stateCallback != nil {
// 		if err := b.readState(key, &state); err != nil {
// 			return err
// 		}
// 		isFound = true
// 		if err := stateCallback(state); err != nil {
// 			return err
// 		}
// 	}
// 	if writer != nil {
// 		for ctx.Err() == nil {
// 			if pKeyBuf, err = createKey(blobberAppID, key.AppID, key.WSID, key.ID, bucketNumber); err != nil {
// 				return err
// 			}
// 			var n int
// 			err = (*(b.appStorage)).Read(ctx, pKeyBuf.Bytes(), nil, nil,
// 				func(ccols []byte, viewRecord []byte) (err error) {
// 					isFound = true
// 					n, err = writer.Write(viewRecord)
// 					return err
// 				})
// 			if err != nil {
// 				logger.Error(fmt.Sprintf("failed to send a BLOB chunk: id %d, appID %d, wsid %d: %s", key.ID, key.AppID, key.WSID, err.Error()))
// 				break
// 			}
// 			if n > 0 {
// 				bucketNumber++
// 			} else {
// 				break
// 			}
// 		}
// 	}

// 	if !isFound && err == nil {
// 		err = iblobstorage.ErrBLOBNotFound
// 	}
// 	return err
// }

func (b *bStorageType) QueryBLOBState(ctx context.Context, key iblobstorage.IBLOBKey) (state iblobstorage.BLOBState, err error) {
	err = b.ReadBLOB(ctx, key,
		func(blobState iblobstorage.BLOBState) (err error) {
			state = blobState
			return nil
		}, nil)
	return
}

// warning: the bucket number must be the last value!
func createKey(columns ...interface{}) (buf *bytes.Buffer, err error) {
	buf = new(bytes.Buffer)
	for _, col := range columns {
		switch v := col.(type) {
		case nil:
			return nil, fmt.Errorf("key column with type «%s» is missed: %w", reflect.ValueOf(col).Type(), errPKeyCreateError)
		case istructs.ClusterAppID, istructs.WSID, istructs.RecordID, uint64:
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

func (b *bStorageType) readState(pKey, cCol []byte) (state iblobstorage.BLOBState, ok bool, err error) {
	var stateBytes []byte
	ok, err = (*(b.appStorage)).Get(pKey, cCol, &stateBytes)
	if err != nil || !ok {
		return state, ok, err
	}
	err = json.Unmarshal(stateBytes, &state)
	return state, true, err
}

func (b *bStorageType) writeState(state *iblobstorage.BLOBState, pKey, cCol []byte) (err error) {
	value, err := json.Marshal(state)
	if err != nil {
		// notest
		panic(err)
	}
	return (*(b.appStorage)).Put(pKey, cCol, value)
}

func newKeyWithBucketNumber(blobKey []byte, bucket uint64) []byte {
	res := make([]byte, len(blobKey)+8)
	copy(res, blobKey)
	binary.LittleEndian.PutUint64(res[len(res)-8:], bucket)
	return res
}

func (q *implSizeLimiter) limit(wantToWriteBytes uint64) error {
	q.uploadedSize += wantToWriteBytes
	if q.uploadedSize > uint64(q.maxSize) {
		return iblobstorage.ErrBLOBSizeQuotaExceeded
	}
	return nil
}
