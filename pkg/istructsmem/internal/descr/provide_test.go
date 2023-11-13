/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

//go:embed provide_test.json
var expectedJson string

func TestBasicUsage(t *testing.T) {
	appDef := appdef.New()

	numName := appdef.NewQName("test", "number")
	strName := appdef.NewQName("test", "string")

	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	n := appDef.AddData(numName, appdef.DataKind_int64, appdef.NullQName, appdef.MinIncl(1))
	n.SetComment("natural (positive) integer")

	s := appDef.AddData(strName, appdef.DataKind_string, appdef.NullQName)
	s.AddConstraints(appdef.MinLen(1), appdef.MaxLen(100), appdef.Pattern(`^\w+$`, "only word characters allowed"))

	doc := appDef.AddSingleton(docName)
	doc.
		AddField("f1", appdef.DataKind_int64, true).
		SetFieldComment("f1", "field comment").
		AddField("f2", appdef.DataKind_string, false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`)).
		AddDataField("numField", numName, false).
		AddRefField("mainChild", false, recName).(appdef.ICDocBuilder).
		AddContainer("rec", recName, 0, 100, "container comment").(appdef.ICDocBuilder).
		AddUnique("", []string{"f1", "f2"})
	doc.SetComment(`comment 1`, `comment 2`)

	rec := appDef.AddCRecord(recName)
	rec.
		AddField("f1", appdef.DataKind_int64, true).
		AddField("f2", appdef.DataKind_string, false).
		AddField("phone", appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(25)).
		SetFieldVerify("phone", appdef.VerificationKind_Any...).(appdef.ICRecordBuilder).
		SetUniqueField("phone")

	viewName := appdef.NewQName("test", "view")
	view := appDef.AddView(viewName)
	view.KeyBuilder().PartKeyBuilder().
		AddField("pk_1", appdef.DataKind_int64)
	view.KeyBuilder().ClustColsBuilder().
		AddField("cc_1", appdef.DataKind_string, appdef.MaxLen(100))
	view.ValueBuilder().
		AddDataField("vv_code", strName, true).
		AddRefField("vv_1", true, docName)

	objName := appdef.NewQName("test", "obj")
	obj := appDef.AddObject(objName)
	obj.AddField("f1", appdef.DataKind_string, true)

	appDef.AddCommand(appdef.NewQName("test", "cmd")).
		SetUnloggedParam(objName).
		SetParam(objName).
		SetExtension("cmd", appdef.ExtensionEngineKind_WASM)

	appDef.AddQuery(appdef.NewQName("test", "query")).
		SetParam(objName).
		SetResult(objName).
		SetExtension("cmd", appdef.ExtensionEngineKind_BuiltIn)

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
	require.Greater(len(json), 1)

	//ioutil.WriteFile("C://temp//provide_test.json", json, 0644)

	require.JSONEq(expectedJson, string(json))
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
