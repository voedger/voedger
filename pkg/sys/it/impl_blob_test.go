/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Persistent(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
		coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)
	log.Println(blobID)

	// read, authorize over headers
	blobReader := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)

	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// read, authorize over unescaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer "+ws.Owner.Token),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// read, authorize over escaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer%20"+ws.Owner.Token),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)
}

func TestBlobberErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("403 forbidden on write without token", func(t *testing.T) {
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, []byte{},
			coreutils.Expect403(),
		)
	})

	t.Run("403 forbidden on read without token", func(t *testing.T) {
		expBLOB := []byte{1, 2, 3, 4, 5}
		blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
		)
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID, coreutils.Expect403())
	})

	t.Run("403 forbidden on blob size quota exceeded", func(t *testing.T) {
		bigBLOB := make([]byte, 150)
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, bigBLOB,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403(),
		)
	})

	t.Run("404 not found on querying an unexsting blob", func(t *testing.T) {
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, 1,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect404(),
		)
	})

	t.Run("400 on wrong Content-Type and name+mimeType query params", func(t *testing.T) {
		t.Run("neither Content-Type nor name+mimeType query params are not provided", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("no name+mimeType query params and non-(mutipart/form-data) Content-Type", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("both name+mimeType query params and Conten-Type are specified", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("boundary of multipart/form-data is not specified", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
	})
}

func TestBasicUsage_Temporary(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	blobSUUID := vit.UploadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB, iblobstorage.DurationType_1Day,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
		coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)
	log.Println(blobSUUID)

	// read
	blobReader := vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, blobSUUID, coreutils.WithAuthorizeBy(ws.Owner.Token))
	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	t.Run("expiration", func(t *testing.T) {

		// make the temp blob almost expired
		vit.TimeAdd(time.Duration(iblobstorage.DurationType_1Day.Seconds()-1) * time.Second)
		// testingu.MockTime.Add(time.Duration(iblobstorage.DurationType_1Day.Seconds()-1) * time.Second)

		// re-take the token because it is expired
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		// token := vit.GetPrincipal(istructs.AppQName_test1_app1, ws.Owner.Name).Token

		// check the temp blob still exists
		blobReader := vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, blobSUUID, coreutils.WithAuthorizeBy(ws.Owner.Token))
		actualBLOBContent, err := io.ReadAll(blobReader)
		require.NoError(err)
		require.Equal(expBLOB, actualBLOBContent)

		// cross the temp blob expiration instant
		testingu.MockTime.Add(time.Second)

		// check the temp blob is disappeared
		vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, blobSUUID,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect404(),
		)
	})
}

func TestTemporaryBLOBErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	vit.UploadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB, iblobstorage.DurationType_1Day,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
		coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)

	t.Run("404 on not found", func(t *testing.T) {
		vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, "unknownSUUIDaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect404())
	})
}
