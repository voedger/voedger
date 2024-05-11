/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	//go:embed logo.png
	blob    []byte
	maxSize int64 = 19266
)

func TestBasicUsage(t *testing.T) {
	var (
		key = iblobstorage.KeyType{
			AppID: 2,
			WSID:  2,
			ID:    2,
		}
		desc = iblobstorage.DescrType{
			Name:     "logo.png",
			MimeType: "image/png",
		}
	)
	require := require.New(t)

	asf := mem.Provide()
	asp := istorageimpl.Provide(asf)
	storage, err := asp.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	blobber := Provide(&storage, time.Now)
	ctx := context.TODO()
	reader := provideTestData()

	t.Run("Read blob that is absent. We MUST get error iblobstorage.ErrBLOBNotFound.", func(t *testing.T) {
		var (
			buf   bytes.Buffer
			state iblobstorage.BLOBState
		)
		writer := bufio.NewWriter(&buf)
		err := blobber.ReadBLOB(ctx, key, func(blobState iblobstorage.BLOBState) (err error) {
			state = blobState
			require.Nil(state)
			return nil
		}, writer)
		require.ErrorIs(err, iblobstorage.ErrBLOBNotFound)
	})

	t.Run("Write blob to storage, return must be without errors", func(t *testing.T) {
		err := blobber.WriteBLOB(ctx, key, desc, reader, maxSize)
		require.NoError(err)
	})

	t.Run("Read blob status, return must be without errors", func(t *testing.T) {
		bs, err := blobber.QueryBLOBState(ctx, key)
		require.NoError(err)
		require.Equal(iblobstorage.BLOBStatus_Completed, bs.Status)
	})

	t.Run("Read blob that present in storage and compare with reference", func(t *testing.T) {
		var (
			buf   bytes.Buffer
			state iblobstorage.BLOBState
		)
		writer := bufio.NewWriter(&buf)

		// Reset reader and read anew
		reader.Reset(blob)
		v, err := readData(ctx, reader)
		require.NoError(err)

		// Read
		err = blobber.ReadBLOB(ctx, key, func(blobState iblobstorage.BLOBState) (err error) {
			state = blobState
			log.Println(state.Error)
			return nil
		}, writer)

		require.NoError(err)
		err = writer.Flush()
		require.NoError(err)

		// Compare
		require.Equal(v, buf.Bytes())
	})
}

func TestQuotaExceed(t *testing.T) {
	var (
		key = iblobstorage.KeyType{
			AppID: 2,
			WSID:  2,
			ID:    2,
		}
		desc = iblobstorage.DescrType{
			Name:     "logo.png",
			MimeType: "image/png",
		}
	)
	require := require.New(t)
	asf := mem.Provide()
	asp := istorageimpl.Provide(asf)
	storage, err := asp.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)
	blobber := Provide(&storage, time.Now)
	reader := provideTestData()
	ctx := context.Background()
	// Quota (maxSize -1 = 19265) assigned to reader less then filesize logo.png (maxSize)
	// So, it must be error
	err = blobber.WriteBLOB(ctx, key, desc, reader, maxSize-1)
	require.Error(err, "Reading a file larger than the quota assigned to the reader. It must be a error.")
}

func provideTestData() (reader *bytes.Reader) {
	return bytes.NewReader(blob)
}

func readData(ctx context.Context, reader io.Reader) (data []byte, err error) {
	var bytesRead int
	var entity []byte
	nBytes := int64(0)
	buf := make([]byte, 0, chunkSize)
	for err == nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		bytesRead, err = reader.Read(buf[:cap(buf)])
		if bytesRead > 0 {
			buf = buf[:bytesRead]
			nBytes += int64(len(buf))
			entity = append(entity, buf...)
		}
	}
	if err == io.EOF {
		err = nil
	}
	return entity, err
}
