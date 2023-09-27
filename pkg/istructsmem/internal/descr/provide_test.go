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
		SetFieldComment("f1", "field comment").
		AddStringField("f2", false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`)).
		AddRefField("mainChild", false, recName).(appdef.ICDocBuilder).
		AddContainer("rec", recName, 0, 100, "container comment").(appdef.ICDocBuilder).
		AddUnique("", []string{"f1", "f2"})
	doc.SetComment(`comment 1`, `comment 2`)

	rec := appDef.AddCRecord(recName)
	rec.
		AddField("f1", appdef.DataKind_int64, true).
		AddStringField("f2", false).
		AddStringField("phone", true, appdef.MinLen(1), appdef.MaxLen(25)).
		SetFieldVerify("phone", appdef.VerificationKind_Any...).(appdef.ICRecordBuilder).
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
	require.Greater(len(json), 1)

	require.Regexp(`^{`, string(json))
	require.Regexp(`}$`, string(json))

	require.Regexp(`("Name")(\s*:\s*)("test1/app1")`, string(json), "app name expected")

	require.Regexp(
		`("test\.doc")(\s*:\s*{\s*)`+
			`("Comment")(\s*:\s*)("comment 1\\ncomment 2")(\s*,\s*)`+
			`("Name")(\s*:\s*)("test\.doc")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("TypeKind_CDoc")`,
		string(json), "doc «test.doc» expected")

	require.Regexp(
		`("Name")(\s*:\s*)("sys\.QName")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("DataKind_QName")`,
		string(json),
		"system field «sys.QName» expected")

	require.Regexp(
		`("Comment")(\s*:\s*)("field comment")(\s*,\s*)`+
			`("Name")(\s*:\s*)("f1")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("DataKind_int64")(\s*,\s*)`+
			`("Required")(\s*:\s*)(true)`,
		string(json),
		"int64 field «f1» expected")

	require.Regexp(
		`("Name")(\s*:\s*)("f2")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("DataKind_string")(\s*,\s*)`+
			`("Restricts")(\s*:\s*{\s*)`+
			`("MinLen")(\s*:\s*)(4)(\s*,\s*)`+
			`("MaxLen")(\s*:\s*)(4)(\s*,\s*)`+
			`("Pattern")(\s*:\s*)("\^\\\\w\+\$)`,
		string(json),
		"string field «f2» expected")

	require.Regexp(
		`("Name")(\s*:\s*)("mainChild")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("DataKind_RecordID")(\s*,\s*)`+
			`("Refs")(\s*:\s*)(\[\s*"test\.rec"\s*\])`,
		string(json),
		"ref field «mainChild» expected")

	require.Regexp(
		`("Uniques")(\s*:\s*\[\s*{\s*)`+
			`("Name")(\s*:\s*)("\w+")(\s*,\s*)`+
			`("Fields")(\s*:\s*\[\s*)`+
			`("f1")(\s*,\s*)`+
			`("f2")(\s*\]\s*)`,
		string(json),
		"unique expected")

	require.Regexp(
		`("Containers")(\s*:\s*\[\s*{\s*)`+
			`("Comment")(\s*:\s*)("container comment")(\s*,\s*)`+
			`("Name")(\s*:\s*)("rec")(\s*,\s*)`+
			`("Type")(\s*:\s*)("test\.rec")(\s*,\s*)`+
			`("MinOccurs")(\s*:\s*)(0)(\s*,\s*)`+
			`("MaxOccurs")(\s*:\s*)(100)`,
		string(json),
		"container expected")

	require.Regexp(`("Name")(\s*:\s*)("test\.rec")`, string(json), "record «test.rec» expected")

	require.Regexp(
		`("Name")(\s*:\s*)("phone")(\s*,\s*)`+
			`("Kind")(\s*:\s*)("DataKind_string")(\s*,\s*)`+
			`("Required")(\s*:\s*)(true)(\s*,\s*)`+
			`("Verifiable")(\s*:\s*)(true)`+
			``,
		string(json),
		"verified field «phone» expected")
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
