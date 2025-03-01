/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vvmshutdowntest

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vvm"
)

// this test must be the single or last in the entire test process, otherwise killerRoutine will kill the process if the rest of tests will last longer than 5 seconds
// so it is implemented in a separate package
// [~server.design.orch/VVM.test.Shutdown~impl]
func TestAutomaticShutdownOnLeadershipLoss(t *testing.T) {
	r := require.New(t)

	vvmCfg := vvm.GetTestVVMCfg(net.IPv4(192, 168, 0, 1))
	vvm1, err := vvm.Provide(vvmCfg)
	r.NoError(err)

	// Launch VVM1
	problemCtx := vvm1.Launch(vvm.DefaultLeadershipDurationSeconds, vvm.DefaultLeadershipAcquisitionDuration)
	r.NoError(problemCtx.Err())

	// Simulate leadership loss
	pKey := make([]byte, utils.Uint32Size)
	cCols := make([]byte, utils.Uint32Size)
	binary.BigEndian.PutUint32(pKey, 1)
	binary.BigEndian.PutUint32(cCols, uint32(1))

	vvmAppTTLStorage, err := vvm1.APIs.IAppStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(t, err)
	ok, err := vvmAppTTLStorage.CompareAndSwap(
		pKey,
		cCols,
		[]byte(vvmCfg.IP.String()),
		[]byte("another_value"),
		50,
	)
	r.NoError(err)
	r.True(ok)

	// Bump mock time
	coreutils.MockTime.Sleep(time.Duration(vvm.DefaultLeadershipDurationSeconds) * time.Second)

	// Check problem context
	<-problemCtx.Done()
	r.ErrorIs(vvm1.Shutdown(), vvm.ErrLeadershipLost)
}
