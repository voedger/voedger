/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/blobber"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_BLOBProcessors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ApplicationXBinary, expBLOB,
		coreutils.WithAuthorizeBy(systemPrincipal),
		coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)
	log.Println(blobID)

	// read, authorize over headers
	resp := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB := []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over unescaped cookies
	resp = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer "+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over escaped cookies
	resp = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer%20"+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, POST
	resp = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

}

func TestBlobberErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	t.Run("401 unauthorized on no authorization token in neither headers nor cookies", func(t *testing.T) {
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ApplicationXBinary, []byte{},
			coreutils.Expect401(),
		)
	})

	t.Run("403 forbidden on blob size quota exceeded", func(t *testing.T) {
		bigBLOB := make([]byte, 150)
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ApplicationXBinary, bigBLOB,
			coreutils.WithAuthorizeBy(systemPrincipal),
			coreutils.Expect403(),
		)
	})

	t.Run("404 not found on querying an unexsting blob", func(t *testing.T) {
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, 1,
			coreutils.WithAuthorizeBy(systemPrincipal),
			coreutils.Expect404(),
		).Println()
	})

	// t.Run("400 on wrong Content-Type and name+mimeType query params", func(t *testing.T) {
	// 	t.Run("neither Content-Type nor name+mimeType query params are not provided", func(t *testing.T) {
	// 		vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
	// 			coreutils.WithAuthorizeBy(systemPrincipal),
	// 			coreutils.Expect400(),
	// 		).Println()
	// 	})
	// 	t.Run("no name+mimeType query params and non-(mutipart/form-data) Content-Type", func(t *testing.T) {
	// 		vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
	// 			coreutils.WithAuthorizeBy(systemPrincipal),
	// 			coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"),
	// 			coreutils.Expect400(),
	// 		).Println()
	// 	})
	// 	t.Run("both name+mimeType query params and Conten-Type are specified", func(t *testing.T) {
	// 		vit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "blobContent",
	// 			coreutils.WithAuthorizeBy(systemPrincipal),
	// 			coreutils.WithHeaders("Content-Type", "multipart/form-data"),
	// 			coreutils.Expect400(),
	// 		).Println()
	// 	})
	// 	t.Run("boundary of multipart/form-data is not specified", func(t *testing.T) {
	// 		vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
	// 			coreutils.WithAuthorizeBy(systemPrincipal),
	// 			coreutils.WithHeaders("Content-Type", "multipart/form-data"),
	// 			coreutils.Expect400(),
	// 		).Println()
	// 	})
	// })
}

func TestBlobMultipartUpload(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	blobs := []blobber.BLOB{
		{
			Content: []byte{1, 2, 3, 4, 5},
			Name:    "blob1",
		},
		{
			Content: []byte{6, 7, 8, 9, 10},
			Name:    "blob2",
		},
	}

	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipalToken, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)
	blobIDs := vit.UploadBLOBs(istructs.AppQName_test1_app1, ws.WSID, blobs,
		coreutils.WithAuthorizeBy(systemPrincipalToken))

	// read blob1
	actualBLOB1 := vit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, blobIDs[0], systemPrincipalToken)
	require.Equal("application/x-binary", actualBLOB1.MimeType)
	require.Equal(`blob1`, actualBLOB1.Name)
	require.Equal(blobs[0].Content, actualBLOB1.Content)

	// read blob2
	actualBLOB2 := vit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, blobIDs[1], systemPrincipalToken)
	require.Equal("application/x-binary", actualBLOB2.MimeType)
	require.Equal(`blob2`, actualBLOB2.Name)
	require.Equal(blobs[1].Content, actualBLOB2.Content)
}
