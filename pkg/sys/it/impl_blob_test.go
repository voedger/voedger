/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
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

	// [~server.apiv2.blobs/it.TestBlobsCreate~impl]
	// write
	blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
		it.QNameDocWithBLOB, it.Field_Blob, coreutils.WithAuthorizeBy(ws.Owner.Token))
	log.Println(blobID)

	// check wdoc.sys.BLOB - OwnerRecordID is not filled yet
	res := vit.SqlQuery(ws, "select * from sys.BLOB.%d", blobID)
	require.EqualValues(iblobstorage.BLOBStatus_Completed, res["status"])
	require.EqualValues(it.QNameDocWithBLOB.String(), res["OwnerRecord"])
	require.EqualValues("Blob", res["OwnerRecordField"])
	require.Zero(res["OwnerRecordID"])

	// put the blob into the owner field
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","Blob":%d}}]}`, blobID)
	ownerID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// check wdoc.sys.BLOB - OwnerRecordID must be automatically filled
	res = vit.SqlQuery(ws, "select * from sys.BLOB.%d", blobID)
	require.EqualValues(iblobstorage.BLOBStatus_Completed, res["status"])
	require.EqualValues(it.QNameDocWithBLOB.String(), res["OwnerRecord"])
	require.EqualValues("Blob", res["OwnerRecordField"])
	require.EqualValues(ownerID, istructs.RecordID(res["OwnerRecordID"].(float64)))

	// read, authorize over headers
	blobReader := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", ownerID,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)

	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// [~server.apiv2.blobs/it.TestBlobsRead~impl]
	// read, authorize over unescaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", ownerID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer "+ws.Owner.Token),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// read, authorize over escaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", ownerID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer%20"+ws.Owner.Token),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)
}

func TestBlobberErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	expBLOB := []byte{1, 2, 3, 4, 5}

	t.Run("403 forbidden on write without token", func(t *testing.T) {
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, []byte{},
			it.QNameDocWithBLOB, it.Field_Blob, coreutils.Expect403(),
		)
	})

	t.Run("403 forbidden on read without token", func(t *testing.T) {
		blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
			it.QNameDocWithBLOB, it.Field_Blob, coreutils.WithAuthorizeBy(ws.Owner.Token))
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", blobID, coreutils.Expect403())
	})

	t.Run("403 forbidden on blob size quota exceeded", func(t *testing.T) {
		bigBLOB := make([]byte, 150)
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, bigBLOB,
			it.QNameDocWithBLOB, it.Field_Blob,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403(),
		)
	})

	t.Run("404 not found on querying an unexsting blob", func(t *testing.T) {
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", 1,
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

	t.Run("write: wrong owner", func(t *testing.T) {
		t.Run("doc", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/unknown.doc/blobs/someField", ws.WSID),
				"blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders(
					coreutils.BlobName, "newBlob",
					coreutils.ContentType, coreutils.ContentType_ApplicationXBinary,
				),
				coreutils.Expect400("blob owner QName unknown.doc is unknown"),
			).Println()
		})
		t.Run("field", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.DocWithBLOB/blobs/someField", ws.WSID),
				"blobContent",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders(
					coreutils.BlobName, "newBlob",
					coreutils.ContentType, coreutils.ContentType_ApplicationXBinary,
				),
				coreutils.Expect400("blob owner field someField does not exist in blob owner app1pkg.DocWithBLOB"),
			).Println()
		})

		t.Run("field type", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.DocWithBLOB/blobs/IntFld", ws.WSID),
				"blob",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithHeaders(
					coreutils.BlobName, "newBlob",
					coreutils.ContentType, coreutils.ContentType_ApplicationXBinary,
				),
				coreutils.Expect400("blob owner app1pkg.DocWithBLOB.IntFld must be of blob type"),
			).Println()
		})
	})

	t.Run("read: wrong owner", func(t *testing.T) {
		t.Run("doc", func(t *testing.T) {
			vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, appdef.NewQName("unknown", "doc"), "Name", 1,
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400("document or record unknown.doc is not defined in Workspace «app1pkg.test_wsWS»"),
			)
		})

		t.Run("id", func(t *testing.T) {
			vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameApp1_CDocCountry, "Name", 1,
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect404("document app1pkg.Country with ID 1 not found"),
			)
		})

		// insert owner record but do not specify owner field
		body := `{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB", "IntFld": 42}}]}`
		docWithBLOBID_noBLOB := vit.PostWS(ws, "c.sys.CUD", body).NewID()

		t.Run("target field is not set", func(t *testing.T) {
			vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", docWithBLOBID_noBLOB,
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400("no value for owner field Blob in blob owner doc app1pkg.DocWithBLOB"),
			)
		})

		t.Run("non-blob field", func(t *testing.T) {
			vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "IntFld", docWithBLOBID_noBLOB,
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400("owner field app1pkg.DocWithBLOB.IntFld is not of blob type"),
			)
		})
	})

	t.Run("update owner", func(t *testing.T) {
		blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
			it.QNameDocWithBLOB, it.Field_Blob, coreutils.WithAuthorizeBy(ws.Owner.Token))
		t.Run("403 forbidden on set blob to another owner record", func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.air_table_plan","image":%d}}]}`, blobID)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403("intended for app1pkg.DocWithBLOB.Blob whereas it is being used in app1pkg.air_table_plan.image"))
		})
		t.Run("403 forbidden on set blob to another owner record field", func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","AnotherBlob":%d}}]}`, blobID)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403("intended for app1pkg.DocWithBLOB.Blob whereas it is being used in app1pkg.DocWithBLOB.AnotherBlob"))
		})

		t.Run("403 forbidden on use the same BLOB twice in CUDs", func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds":[
				{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","Blob":%[1]d}},
				{"fields":{"sys.ID": 2,"sys.QName":"app1pkg.DocWithBLOB","Blob":%[1]d}}
			]}`, blobID)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403("mentioned in CUD app1pkg.DocWithBLOB.Blob is used already in CUD app1pkg.DocWithBLOB.1"))
		})

		// assign the blob to the owner
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","Blob":%d}}]}`, blobID)
		assignedBLOBOwnerID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

		t.Run("403 forbidden on re-use the blob", func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","Blob":%d}}]}`, blobID)
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403(fmt.Sprintf(" mentioned in CUD app1pkg.DocWithBLOB.Blob is used already in app1pkg.DocWithBLOB.%d.Blob", assignedBLOBOwnerID)))
		})
	})

	t.Run("read from denied field", func(t *testing.T) {
		blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
			it.QNameDocWithBLOB, it.Field_BlobReadDenied, coreutils.WithAuthorizeBy(ws.Owner.Token))

		// ok put the blob into the read-denied field
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","BlobReadDenied":%d}}]}`, blobID)
		ownerID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

		// 403 on read the BLOB from read-denied field
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "BlobReadDenied", ownerID,
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403(),
		)
	})
}

func TestBasicUsage_Temporary(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	// [~server.apiv2.tblobs/it.TestTBlobsCreate~impl]
	blobSUUID := vit.UploadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB, iblobstorage.DurationType_1Day,
		coreutils.WithAuthorizeBy(ws.Owner.Token))
	log.Println(blobSUUID)

	// read
	// [~server.apiv2.blobs/it.TestTBlobsRead~impl]
	blobReader := vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, blobSUUID, coreutils.WithAuthorizeBy(ws.Owner.Token))
	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	t.Run("expiration", func(t *testing.T) {

		// make the temp blob almost expired
		vit.TimeAdd(time.Duration(iblobstorage.DurationType_1Day.Seconds()-1) * time.Second)

		// re-take the token because it is expired
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

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
		coreutils.WithAuthorizeBy(ws.Owner.Token))

	t.Run("404 on not found", func(t *testing.T) {
		vit.ReadTempBLOB(istructs.AppQName_test1_app1, ws.WSID, "unknownSUUIDaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.Expect404())
	})
}

func TestAPIv1v2BackwardCompatibility(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write BLOB using APIv1
	httpClient, cleanup := coreutils.NewIHTTPClient()
	defer cleanup()
	uploadBLOBURL := fmt.Sprintf("%s/blob/test1/app1/%d?name=%s&mimeType=%s", vit.IFederation.URLStr(), ws.WSID,
		url.QueryEscape(coreutils.BlobName), url.QueryEscape(coreutils.ContentType_ApplicationXBinary))
	resp, err := httpClient.ReqReader(uploadBLOBURL, io.NopCloser(bytes.NewReader(expBLOB)),
		coreutils.WithMethod(http.MethodPost),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	require.NoError(err)
	blobID, err := strconv.ParseUint(resp.Body, utils.DecimalBase, utils.BitSize64)
	require.NoError(err)

	// put the blob into the owner field
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.DocWithBLOB","Blob":%d}}]}`, blobID)
	ownerID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	// the BLOB written via APIv1, should be ok to read via APIv2
	blobReader := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameDocWithBLOB, "Blob", ownerID,
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal(coreutils.BlobName, blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)
}

func TestODocWithBLOB(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write BLOB
	blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ContentType_ApplicationXBinary, expBLOB,
		it.QNameDocWithBLOB, it.Field_Blob, coreutils.WithAuthorizeBy(ws.Owner.Token))
	log.Println(blobID)

	// set to ODoc
	body := fmt.Sprintf(`{"args":{"sys.ID": 1,"Blob":%d}}`, blobID)
	ownerID := vit.PostWS(ws, "c.app1pkg.CmdODocWithBLOB", body).NewID()

	// read from ODoc
	blobReader := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, it.QNameODocWithBLOB, "Blob", ownerID, coreutils.WithAuthorizeBy(ws.Owner.Token))
	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ContentType_ApplicationXBinary, blobReader.ContentType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)
}
