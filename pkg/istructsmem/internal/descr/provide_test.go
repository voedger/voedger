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

	appDef.AddPackage("test", "test/path")

	numName := appdef.NewQName("test", "number")
	strName := appdef.NewQName("test", "string")

	sysRecords := appdef.NewQName("sys", "records")
	sysViews := appdef.NewQName("sys", "views")

	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	n := appDef.AddData(numName, appdef.DataKind_int64, appdef.NullQName, appdef.MinIncl(1))
	n.SetComment("natural (positive) integer")

	s := appDef.AddData(strName, appdef.DataKind_string, appdef.NullQName)
	s.AddConstraints(appdef.MinLen(1), appdef.MaxLen(100), appdef.Pattern(`^\w+$`, "only word characters allowed"))

	doc := appDef.AddCDoc(docName)
	doc.SetSingleton()
	doc.
		AddField("f1", appdef.DataKind_int64, true).
		SetFieldComment("f1", "field comment").
		AddField("f2", appdef.DataKind_string, false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`)).
		AddDataField("numField", numName, false).
		AddRefField("mainChild", false, recName).(appdef.ICDocBuilder).
		AddContainer("rec", recName, 0, 100, "container comment").(appdef.ICDocBuilder).
		AddUnique(appdef.UniqueQName(doc.QName(), "unique1"), []string{"f1", "f2"})
	doc.SetComment(`comment 1`, `comment 2`)

	rec := appDef.AddCRecord(recName)
	rec.
		AddField("f1", appdef.DataKind_int64, true).
		AddField("f2", appdef.DataKind_string, false).
		AddField("phone", appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(25)).
		SetFieldVerify("phone", appdef.VerificationKind_Any...).(appdef.ICRecordBuilder).
		SetUniqueField("phone").
		AddUnique(appdef.UniqueQName(rec.QName(), "uniq1"), []string{"f1"})

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

	cmdName := appdef.NewQName("test", "cmd")
	appDef.AddCommand(cmdName).
		SetUnloggedParam(objName).
		SetParam(objName).
		SetEngine(appdef.ExtensionEngineKind_WASM)

	appDef.AddQuery(appdef.NewQName("test", "query")).
		SetParam(objName).
		SetResult(appdef.QNameANY)

	prj := appDef.AddProjector(appdef.NewQName("test", "projector"))
	prj.
		SetWantErrors().
		SetEngine(appdef.ExtensionEngineKind_WASM)
	prj.Events().
		Add(recName, appdef.ProjectorEventKind_AnyChanges...).SetComment(recName, "run projector every time when «test.rec» is changed").
		Add(cmdName).SetComment(cmdName, "run projector every time when «test.cmd» command is executed").
		Add(objName).SetComment(objName, "run projector every time when any command with «test.obj» argument is executed")
	prj.States().
		Add(sysRecords, docName, recName).SetComment(sysRecords, "needs to read «test.doc» and «test.rec» from «sys.records» storage")
	prj.Intents().
		Add(sysViews, viewName).SetComment(sysViews, "needs to update «test.view» from «sys.views» storage")

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
	require.NotEmpty(json)

	// ioutil.WriteFile("C://temp//provide_test.json", json, 0644)

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
