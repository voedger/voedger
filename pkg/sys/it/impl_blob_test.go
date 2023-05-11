/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

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
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_BLOBProcessors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	expBLOB := []byte{1, 2, 3, 4, 5}

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// write
	resp := vit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), string(expBLOB),
		coreutils.WithAuthorizeBy(systemPrincipal),
		coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
	)
	blobID, err := strconv.Atoi(resp.Body)
	require.NoError(err)
	log.Println(blobID)

	// read, authorize over headers
	resp = vit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		coreutils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB := []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over unescaped cookies
	resp = vit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		coreutils.WithCookies(coreutils.Authorization, "Bearer "+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, authorize over escaped cookies
	resp = vit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID),
		coreutils.WithCookies(coreutils.Authorization, "Bearer%20"+systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

	// read, POST
	resp = vit.Post(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, blobID), "",
		coreutils.WithAuthorizeBy(systemPrincipal),
	)
	actBLOB = []byte(resp.Body)
	require.Equal("application/x-binary", resp.HTTPResp.Header["Content-Type"][0])
	require.Equal(`attachment;filename="test"`, resp.HTTPResp.Header["Content-Disposition"][0])
	require.Equal(expBLOB, actBLOB)

}

func TestBlobberErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	t.Run("401 unauthorized on no authorization token in neither headers nor cookies", func(t *testing.T) {
		vit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "",
			coreutils.Expect401(),
		).Println()
	})

	t.Run("403 forbidden on blob size quota exceeded", func(t *testing.T) {
		bigBLOB := make([]byte, 150)
		vit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), string(bigBLOB),
			coreutils.WithAuthorizeBy(systemPrincipal),
			coreutils.Expect403(),
		).Println()
	})

	t.Run("404 not found on querying an unexsting blob", func(t *testing.T) {
		vit.Get(fmt.Sprintf(`blob/test1/app1/%d/%d`, ws.WSID, 1),
			coreutils.WithAuthorizeBy(systemPrincipal),
			coreutils.Expect404(),
		).Println()
	})

	t.Run("400 on wrong Content-Type and name+mimeType query params", func(t *testing.T) {
		t.Run("neither Content-Type nor name+mimeType query params are not provided", func(t *testing.T) {
			vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("no name+mimeType query params and non-(mutipart/form-data) Content-Type", func(t *testing.T) {
			vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("both name+mimeType query params and Conten-Type are specified", func(t *testing.T) {
			vit.Post(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("boundary of multipart/form-data is not specified", func(t *testing.T) {
			vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
	})
}

func TestBlobMultipartUpload(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	expBLOB1 := []byte{1, 2, 3, 4, 5}
	expBLOB2 := []byte{6, 7, 8, 9, 10}
	BLOBs := [][]byte{expBLOB1, expBLOB2}
	as, err := vit.AppStructs(istructs.AppQName_test1_app1)
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
	blobIDsStr := vit.Post(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), body.String(),
		coreutils.WithAuthorizeBy(systemPrincipalToken),
		coreutils.WithHeaders("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary)),
	).Body
	log.Println(blobIDsStr)

	blobIDsStrs := strings.Split(string(blobIDsStr), ",")
	blob1ID, err := strconv.Atoi(blobIDsStrs[0])
	require.NoError(err)
	blob2ID, err := strconv.Atoi(blobIDsStrs[1])
	require.NoError(err)

	// read blob1
	blob := vit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, int64(blob1ID), systemPrincipalToken)
	require.Equal("application/x-binary", blob.MimeType)
	require.Equal(`blob1`, blob.Name)
	require.Equal(expBLOB1, blob.Content)

	// read blob2
	blob = vit.GetBLOB(istructs.AppQName_test1_app1, ws.WSID, int64(blob2ID), systemPrincipalToken)
	require.Equal("application/x-binary", blob.MimeType)
	require.Equal(`blob2`, blob.Name)
	require.Equal(expBLOB2, blob.Content)
}
