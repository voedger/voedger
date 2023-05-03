/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"bytes"
	"fmt"
	"log"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_BLOBProcessors(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	as, err := hit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	resp := hit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), string(expBLOB),
		utils.WithAuthorizeBy(systemPrincipal),
		utils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)
	blobID, err := strconv.Atoi(resp.Body)
	require.NoError(err)
	log.Println(blobID)

	// read, authorize over headers
	resp = hit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		utils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB := []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over unescaped cookies
	resp = hit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		utils.WithCookies(coreutils.Authorization, "Bearer "+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over escaped cookies
	resp = hit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		utils.WithCookies(coreutils.Authorization, "Bearer%20"+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, POST
	resp = hit.Post(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID), "",
		utils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

}

func TestBlobberErrors(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	as, err := hit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	t.Run("401 unauthorized on no authorization token in neither headers nor cookies", func(t *testing.T) {
		hit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "",
			utils.Expect401(),
		).Println()
	})

	t.Run("403 forbidden on blob size quota exceeded", func(t *testing.T) {
		bigBLOB := make([]byte, 150)
		hit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), string(bigBLOB),
			utils.WithAuthorizeBy(systemPrincipal),
			utils.Expect403(),
		).Println()
	})

	t.Run("404 not found on querying an unexsting blob", func(t *testing.T) {
		hit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, 1),
			utils.WithAuthorizeBy(systemPrincipal),
			utils.Expect404(),
		).Println()
	})

	t.Run("400 on wrong Content-Type and name+mimeType query params", func(t *testing.T) {
		t.Run("neither Content-Type nor name+mimeType query params are not provided", func(t *testing.T) {
			hit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				utils.WithAuthorizeBy(systemPrincipal),
				utils.Expect400(),
			).Println()
		})
		t.Run("no name+mimeType query params and non-(mutipart/form-data) Content-Type", func(t *testing.T) {
			hit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				utils.WithAuthorizeBy(systemPrincipal),
				utils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"),
				utils.Expect400(),
			).Println()
		})
		t.Run("both name+mimeType query params and Conten-Type are specified", func(t *testing.T) {
			hit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "blobContent",
				utils.WithAuthorizeBy(systemPrincipal),
				utils.WithHeaders("Content-Type", "multipart/form-data"),
				utils.Expect400(),
			).Println()
		})
		t.Run("boundary of multipart/form-data is not specified", func(t *testing.T) {
			hit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				utils.WithAuthorizeBy(systemPrincipal),
				utils.WithHeaders("Content-Type", "multipart/form-data"),
				utils.Expect400(),
			).Println()
		})
	})
}

func TestBlobMultipartUpload(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	expBLOB1 := []byte{1, 2, 3, 4, 5}
	expBLOB2 := []byte{6, 7, 8, 9, 10}
	BLOBs := [][]byte{expBLOB1, expBLOB2}
	as, err := hit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipalToken, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	// compose body for blobs write request
	body := bytes.NewBuffer(nil)
	w := multipart.NewWriter(body)
	boundary := "----------------"
	w.SetBoundary(boundary)
	for i, blob := range BLOBs {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="blob%d"`, i+1))
		h.Set("Content-Type", "application/x-binary")
		part, err := w.CreatePart(h)
		require.NoError(err)
		_, err = part.Write(blob)
		require.NoError(err)
	}
	require.Nil(w.Close())
	log.Println(body.String())

	// write blobs
	blobIDsStr := hit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), body.String(),
		utils.WithAuthorizeBy(systemPrincipalToken),
		utils.WithHeaders("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary)),
	).Body
	log.Println(blobIDsStr)

	blobIDsStrs := strings.Split(string(blobIDsStr), ",")
	blob1ID, err := strconv.Atoi(blobIDsStrs[0])
	require.NoError(err)
	blob2ID, err := strconv.Atoi(blobIDsStrs[1])
	require.NoError(err)

	// read blob1
	blob := hit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, int64(blob1ID), systemPrincipalToken)
	require.Equal("application/x-binary", blob.MimeType)
	require.Equal(`blob1`, blob.Name)
	require.Equal(expBLOB1, blob.Content)

	// read blob2
	blob = hit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, int64(blob2ID), systemPrincipalToken)
	require.Equal("application/x-binary", blob.MimeType)
	require.Equal(`blob2`, blob.Name)
	require.Equal(expBLOB2, blob.Content)
}
