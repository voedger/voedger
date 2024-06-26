/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package projectors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitions struct {
	mock.Mock
	appparts.IAppPartitions
}

func (s *appPartitions) AppDef(n appdef.AppQName) (appdef.IAppDef, error) {
	def := s.Called(n).Get(0).(appdef.IAppDef)
	err := s.Called(n).Get(1)
	if err == nil {
		return def, nil
	}
	return def, err.(error)
}

func Test_actualizers_DeployPartition(t *testing.T) {
	t.Skip("This test is not ready yet")

	cmd := appdef.NewQName("test", "cmd")
	prj := appdef.NewQName("test", "projector")

	def := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddCommand(cmd)
		adb.AddProjector(prj).
			SetSync(false).
			Events().Add(cmd)
		return adb.MustBuild()
	}()

	parts := new(appPartitions)
	parts.On("AppDef", istructs.AppQName_test1_app1).Return(def, nil)

	vvmCtx, vvmCancel := context.WithCancel(context.Background())

	actualizers := newActualizers(BasicAsyncActualizerConfig{
		VvmName:       "test",
		Ctx:           vvmCtx,
		AppPartitions: parts,
		SecretReader:  nil,
		Tokens:        nil,
		Metrics:       nil,
		Broker:        nil,
		Federation:    nil,
	})

	actualizers.DeployPartition(istructs.AppQName_test1_app1, istructs.PartitionID(1))

	require := require.New(t)

	require.Equal(1, 1)

	vvmCancel()
}
