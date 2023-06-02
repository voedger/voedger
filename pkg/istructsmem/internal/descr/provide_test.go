/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBasicUsage(t *testing.T) {
	appDef := appdef.New()

	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	doc := appDef.AddSingleton(docName)
	doc.
		AddField("f1", appdef.DataKind_int64, true).
		AddField("f2", appdef.DataKind_string, false).
		AddRefField("mainChild", false, recName).(appdef.ICDocBuilder).
		AddContainer("rec", recName, 0, 100).(appdef.ICDocBuilder).
		AddUnique("", []string{"f1", "f2"})

	rec := appDef.AddCRecord(recName)
	rec.
		AddField("f1", appdef.DataKind_int64, true).
		AddField("f2", appdef.DataKind_string, false).
		AddVerifiedField("phone", appdef.DataKind_string, true, appdef.VerificationKind_Any...).(appdef.ICRecordBuilder).
		SetUniqueField("phone")

	res := &mockResources{}
	res.
		On("Resources", mock.AnythingOfType("func(appdef.QName)")).Run(func(args mock.Arguments) {})

	appStr := &mockedAppStructs{}
	appStr.
		On("AppQName").Return(istructs.AppQName_test1_app1).
		On("AppDef").Return(appDef).
		On("Resources").Return(res)

	appLimits := map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit{}

	app := Provide(appStr, appLimits)

	json, err := json.Marshal(app)

	require := require.New(t)
	require.NoError(err)
	require.Contains(string(json), "{")
}

type mockedAppStructs struct {
	istructs.IAppStructs
	mock.Mock
}

func (s *mockedAppStructs) AppDef() appdef.IAppDef {
	return s.Called().Get(0).(appdef.IAppDef)
}

func (s *mockedAppStructs) AppQName() istructs.AppQName {
	return s.Called().Get(0).(istructs.AppQName)
}

func (s *mockedAppStructs) Resources() istructs.IResources {
	return s.Called().Get(0).(istructs.IResources)
}

type mockResources struct {
	istructs.IResources
	mock.Mock
}

func (r *mockResources) Resources(cb func(appdef.QName)) {
	r.Called(cb)
}
