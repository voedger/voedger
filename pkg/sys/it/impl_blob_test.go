/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_BLOBProcessors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	as, err := vit.BuiltIn(istructs.AppQName_test1_app1)
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
	blobReader := vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithAuthorizeBy(systemPrincipal),
	)

	actualBLOBContent, err := io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// read, authorize over unescaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer "+systemPrincipal),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)

	// read, authorize over escaped cookies
	blobReader = vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID,
		coreutils.WithCookies(coreutils.Authorization, "Bearer%20"+systemPrincipal),
	)
	actualBLOBContent, err = io.ReadAll(blobReader)
	require.NoError(err)
	require.Equal(coreutils.ApplicationXBinary, blobReader.MimeType)
	require.Equal("test", blobReader.Name)
	require.Equal(expBLOB, actualBLOBContent)
}

func TestBlobberErrors(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	as, err := vit.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	systemPrincipal, err := payloads.GetSystemPrincipalTokenApp(as.AppTokens())
	require.NoError(err)

	t.Run("401 unauthorized on write without token", func(t *testing.T) {
		vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ApplicationXBinary, []byte{},
			coreutils.Expect401(),
		)
	})

	t.Run("401 unauthorized on read without token", func(t *testing.T) {
		expBLOB := []byte{1, 2, 3, 4, 5}
		blobID := vit.UploadBLOB(istructs.AppQName_test1_app1, ws.WSID, "test", coreutils.ApplicationXBinary, expBLOB,
			coreutils.WithAuthorizeBy(systemPrincipal),
			coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"), // has name+mimeType query params -> any Content-Type except "multipart/form-data" is allowed
		)
		vit.ReadBLOB(istructs.AppQName_test1_app1, ws.WSID, blobID, coreutils.Expect401())
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
		)
	})

	t.Run("400 on wrong Content-Type and name+mimeType query params", func(t *testing.T) {
		t.Run("neither Content-Type nor name+mimeType query params are not provided", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("no name+mimeType query params and non-(mutipart/form-data) Content-Type", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "application/x-www-form-urlencoded"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("both name+mimeType query params and Conten-Type are specified", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d?name=test&mimeType=application/x-binary`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
		t.Run("boundary of multipart/form-data is not specified", func(t *testing.T) {
			vit.POST(fmt.Sprintf(`blob/test1/app1/%d`, ws.WSID), "blobContent",
				coreutils.WithAuthorizeBy(systemPrincipal),
				coreutils.WithHeaders("Content-Type", "multipart/form-data"),
				coreutils.Expect400(),
			).Println()
		})
	})
}
