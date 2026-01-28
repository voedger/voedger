/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/teststore"
)

type (
	testEnvironment struct {
		appName          appdef.AppQName
		pkgName, pkgPath string

		wsName appdef.QName

		AppConfigs AppConfigsType
		AppCfg     *AppConfigType
		AppDef     appdef.IAppDef

		StorageProvider istorage.IAppStorageProvider
		Storage         *teststore.TestMemStorage

		AppStructsProvider istructs.IAppStructsProvider
		AppStructs         istructs.IAppStructs

		// common event entities
		eventRawBytes      []byte
		partition          istructs.PartitionID
		plogOfs            istructs.Offset
		workspace          istructs.WSID
		wlogOfs            istructs.Offset
		saleCmdName        appdef.QName
		saleCmdDocName     appdef.QName
		saleSecureParsName appdef.QName
		registeredTime     istructs.UnixMilli
		deviceIdent        string
		device             istructs.ConnectedDeviceID
		syncTime           istructs.UnixMilli

		// event command tree entities
		buyerIdent     appdef.FieldName
		buyerValue     string
		ageIdent       appdef.FieldName
		ageValue       int8
		heightIdent    appdef.FieldName
		heightValue    float32
		humanIdent     appdef.FieldName
		humanValue     bool
		photoIdent     appdef.FieldName
		photoValue     []byte
		remarkIdent    appdef.FieldName
		remarkValue    string
		emptinessIdent appdef.FieldName
		emptinessValue string
		saleIdent      appdef.FieldName
		basketIdent    appdef.FieldName
		goodIdent      appdef.FieldName
		nameIdent      appdef.FieldName
		codeIdent      appdef.FieldName
		weightIdent    appdef.FieldName
		goodCount      int
		goodNames      []string
		goodCodes      []int64
		goodWeights    []float64

		passwordIdent string

		tempSaleID   istructs.RecordID
		tempBasketID istructs.RecordID
		tempGoodsID  []istructs.RecordID

		// tested data types
		dataIdent appdef.QName
		dataPhoto appdef.QName

		// event CUDs entities
		tablePhotos    appdef.QName
		tempPhotoID    istructs.RecordID
		tablePhotoRems appdef.QName
		tempRemarkID   istructs.RecordID

		// tested resources
		changeCmdName appdef.QName

		queryPhotoFunctionName       appdef.QName
		queryPhotoFunctionParamsName appdef.QName
		photoRawIdent                appdef.FieldName
		photoRawValue                []byte

		// tested rows
		abstractCDoc          appdef.QName
		testRow               appdef.QName
		testRowUserFieldCount int
		testObj               appdef.QName

		// tested records
		testCDoc appdef.QName
		testCRec appdef.QName

		// tested viewRecords
		testViewRecord testViewRecordType
	}

	testViewRecordType struct {
		name        appdef.QName
		partFields  testViewRecordPartKeyFieldsType
		ccolsFields testViewRecordClustKeyFieldsType
		valueFields testViewRecordValueFieldsType
	}

	testViewRecordPartKeyFieldsType struct {
		partition appdef.FieldName
		workspace appdef.FieldName
	}

	testViewRecordClustKeyFieldsType struct {
		device appdef.FieldName
		sorter appdef.FieldName
	}

	testViewRecordValueFieldsType struct {
		buyer   appdef.FieldName
		age     appdef.FieldName
		heights appdef.FieldName
		human   appdef.FieldName
		photo   appdef.FieldName
		record  appdef.FieldName
		event   appdef.FieldName
	}
)

var defTestEnv = testEnvironment{
	appName: istructs.AppQName_test1_app1,
	pkgName: "test",
	pkgPath: "test.com/test",

	wsName: appdef.NewQName("test", "workspace"),

	eventRawBytes:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	partition:          55,
	plogOfs:            10000,
	workspace:          1234,
	wlogOfs:            1000,
	saleCmdName:        appdef.NewQName("test", "sales"),
	saleCmdDocName:     appdef.NewQName("test", "saleArgs"),
	saleSecureParsName: appdef.NewQName("test", "saleSecureArgs"),
	registeredTime:     100500,
	deviceIdent:        "Device",
	device:             762,
	syncTime:           1005001,

	buyerIdent:     "Buyer",
	buyerValue:     "Carlson 哇\"呀呀", // to test unicode issues
	ageIdent:       "Age",
	ageValue:       33,
	heightIdent:    "Height",
	heightValue:    1.75,
	humanIdent:     "isHuman",
	humanValue:     true,
	photoIdent:     "Photo",
	photoValue:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 4, 4, 3, 2, 1, 0},
	remarkIdent:    "Remark",
	remarkValue:    "remark text",
	emptinessIdent: "Emptiness",
	emptinessValue: "to be emptied",

	saleIdent:   "Sale",
	basketIdent: "Basket",
	goodIdent:   "Good",
	nameIdent:   "Name",
	codeIdent:   "Code",
	weightIdent: "Weight",
	goodCount:   2,
	goodNames:   []string{"Biscuits", "Jam"},
	goodCodes:   []int64{7070, 8080},
	goodWeights: []float64{1.1, 2.02},

	passwordIdent: "password",

	tempSaleID:   555,
	tempBasketID: 556,
	tempGoodsID:  []istructs.RecordID{557, 558},

	dataIdent: appdef.NewQName("test", "identString"),
	dataPhoto: appdef.NewQName("test", "KByte"),

	tablePhotos:    appdef.NewQName("test", "photos"),
	tempPhotoID:    1,
	tablePhotoRems: appdef.NewQName("test", "photoRems"),
	tempRemarkID:   11,

	changeCmdName: appdef.NewQName("test", "change"),

	queryPhotoFunctionName:       appdef.NewQName("test", "QueryPhoto"),
	queryPhotoFunctionParamsName: appdef.NewQName("test", "QueryPhotoParams"),
	photoRawIdent:                "rawPhoto",
	photoRawValue:                bytes.Repeat([]byte{1, 2, 3, 4}, 1024), // 4Kb

	abstractCDoc:          appdef.NewQName("test", "abstract"),
	testRow:               appdef.NewQName("test", "Row"),
	testRowUserFieldCount: 13,
	testObj:               appdef.NewQName("test", "Obj"),
	testCDoc:              appdef.NewQName("test", "CDoc"),
	testCRec:              appdef.NewQName("test", "Record"),

	testViewRecord: testViewRecordType{
		name: appdef.NewQName("test", "ViewPhotos"),
		partFields: testViewRecordPartKeyFieldsType{
			partition: "partition",
			workspace: "workspace",
		},
		ccolsFields: testViewRecordClustKeyFieldsType{
			device: "device",
			sorter: "sorter",
		},
		valueFields: testViewRecordValueFieldsType{
			buyer:   "buyer",
			age:     "age",
			heights: "heights",
			human:   "human",
			photo:   "photo",
			record:  "rec",
			event:   "ev",
		},
	},
}

var qNameMyQName = appdef.NewQName("test", "MyQName")

func newTest() *testEnvironment {

	test := defTestEnv

	prepareAppDef := func() appdef.IAppDefBuilder {
		adb := builder.New()
		adb.AddPackage(test.pkgName, test.pkgPath)

		{
			sysWS := adb.AlterWorkspace(appdef.SysWorkspaceQName)

			// for records registry: sys.RecordsRegistry
			v := sysWS.AddView(sys.RecordsRegistryView.Name)
			v.Key().PartKey().AddField(sys.RecordsRegistryView.Fields.IDHi, appdef.DataKind_int64)
			v.Key().ClustCols().AddField(sys.RecordsRegistryView.Fields.ID, appdef.DataKind_int64)
			v.Value().
				AddField(sys.RecordsRegistryView.Fields.WLogOffset, appdef.DataKind_int64, true).
				AddField(sys.RecordsRegistryView.Fields.QName, appdef.DataKind_QName, true).
				AddField(sys.RecordsRegistryView.Fields.IsActive, appdef.DataKind_bool, false)
		}

		wsb := adb.AddWorkspace(test.wsName)

		{
			wsDescQName := appdef.NewQName("test", "WSDesc")
			wsb.AddCDoc(wsDescQName)
			wsb.SetDescriptor(wsDescQName)

			identData := wsb.AddData(test.dataIdent, appdef.DataKind_string, appdef.NullQName)
			identData.AddConstraints(constraints.MinLen(1), constraints.MaxLen(50)).SetComment("string from 1 to 50 runes")

			photoData := wsb.AddData(test.dataPhoto, appdef.DataKind_bytes, appdef.NullQName)
			photoData.AddConstraints(constraints.MaxLen(1024)).SetComment("up to 1Kb")

			saleParams := wsb.AddODoc(test.saleCmdDocName)
			saleParams.
				AddDataField(test.buyerIdent, test.dataIdent, true).
				AddField(test.ageIdent, appdef.DataKind_int8, false).
				AddField(test.heightIdent, appdef.DataKind_float32, false).
				AddField(test.humanIdent, appdef.DataKind_bool, false).
				AddDataField(test.photoIdent, test.dataPhoto, false)
			saleParams.
				AddContainer(test.basketIdent, appdef.NewQName(test.pkgName, test.basketIdent), 1, 1)

			basket := wsb.AddORecord(appdef.NewQName(test.pkgName, test.basketIdent))
			basket.
				AddContainer(test.goodIdent, appdef.NewQName(test.pkgName, test.goodIdent), 0, appdef.Occurs_Unbounded)

			good := wsb.AddORecord(appdef.NewQName(test.pkgName, test.goodIdent))
			good.
				AddField(test.saleIdent, appdef.DataKind_RecordID, true).
				AddField(test.nameIdent, appdef.DataKind_string, true, constraints.MinLen(1)).
				AddField(test.codeIdent, appdef.DataKind_int64, true).
				AddField(test.weightIdent, appdef.DataKind_float64, false)

			saleSecureParams := wsb.AddObject(test.saleSecureParsName)
			saleSecureParams.
				AddField(test.passwordIdent, appdef.DataKind_string, true)

			photoParams := wsb.AddObject(test.queryPhotoFunctionParamsName)
			photoParams.
				AddField(test.buyerIdent, appdef.DataKind_string, true, constraints.MinLen(1), constraints.MaxLen(50)).
				AddField(test.photoRawIdent, appdef.DataKind_bytes, false, constraints.MaxLen(appdef.MaxFieldLength))
		}

		{
			rec := wsb.AddCDoc(test.tablePhotos)
			rec.
				AddDataField(test.buyerIdent, test.dataIdent, true).
				AddField(test.ageIdent, appdef.DataKind_int8, false).
				AddField(test.heightIdent, appdef.DataKind_float32, false).
				AddField(test.humanIdent, appdef.DataKind_bool, false).
				AddDataField(test.photoIdent, test.dataPhoto, false)
			rec.
				AddUnique(appdef.NewQName("test", "photos$uniques$buyerIdent"), []string{test.buyerIdent})
			rec.
				AddContainer(test.remarkIdent, test.tablePhotoRems, 0, appdef.Occurs_Unbounded)

			recChild := wsb.AddCRecord(test.tablePhotoRems)
			recChild.
				AddField(test.photoIdent, appdef.DataKind_RecordID, true).
				AddField(test.remarkIdent, appdef.DataKind_string, true).
				AddField(test.emptinessIdent, appdef.DataKind_string, false)
		}

		{
			abstractDoc := wsb.AddCDoc(test.abstractCDoc)
			abstractDoc.SetComment("abstract test cdoc")
			abstractDoc.SetAbstract()
			abstractDoc.
				AddField("int32", appdef.DataKind_int32, false)
		}

		{
			row := wsb.AddObject(test.testRow)
			row.
				AddField("int8", appdef.DataKind_int8, false).
				AddField("int16", appdef.DataKind_int16, false).
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, constraints.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
		}

		{
			obj := wsb.AddObject(test.testObj)
			obj.
				AddField("int8", appdef.DataKind_int8, false).
				AddField("int16", appdef.DataKind_int16, false).
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, constraints.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
			obj.AddContainer("child", test.testObj, 0, appdef.Occurs_Unbounded)
		}

		{
			cDoc := wsb.AddCDoc(test.testCDoc)
			cDoc.
				AddField("int8", appdef.DataKind_int8, false).
				AddField("int16", appdef.DataKind_int16, false).
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, constraints.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
			cDoc.
				AddContainer("record", test.testCRec, 0, appdef.Occurs_Unbounded)

			cRec := wsb.AddCRecord(test.testCRec)
			cRec.
				AddField("int8", appdef.DataKind_int8, false).
				AddField("int16", appdef.DataKind_int16, false).
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, constraints.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
		}

		{
			view := wsb.AddView(test.testViewRecord.name)
			view.Key().PartKey().
				AddField(test.testViewRecord.partFields.partition, appdef.DataKind_int32).
				AddField(test.testViewRecord.partFields.workspace, appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField(test.testViewRecord.ccolsFields.device, appdef.DataKind_int32).
				AddField(test.testViewRecord.ccolsFields.sorter, appdef.DataKind_string, constraints.MaxLen(100))
			view.Value().
				AddField(test.testViewRecord.valueFields.buyer, appdef.DataKind_string, true).
				AddField(test.testViewRecord.valueFields.age, appdef.DataKind_int8, false).
				AddField(test.testViewRecord.valueFields.heights, appdef.DataKind_float32, false).
				AddField(test.testViewRecord.valueFields.human, appdef.DataKind_bool, false).
				AddDataField(test.testViewRecord.valueFields.photo, test.dataPhoto, false).
				AddField(test.testViewRecord.valueFields.record, appdef.DataKind_Record, false).
				AddField(test.testViewRecord.valueFields.event, appdef.DataKind_Event, false)
		}

		{
			wsb.AddCommand(test.saleCmdName).SetUnloggedParam(test.saleSecureParsName).SetParam(test.saleCmdDocName)
			wsb.AddCommand(test.changeCmdName)
			wsb.AddQuery(test.queryPhotoFunctionName).SetParam(test.queryPhotoFunctionParamsName)
		}

		wsb.AddCDoc(qNameMyQName)

		return adb
	}

	test.AppConfigs = make(AppConfigsType, 1)
	test.AppCfg = test.AppConfigs.AddBuiltInAppConfig(test.appName, prepareAppDef())
	test.AppCfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	test.AppDef = test.AppCfg.AppDef

	test.AppCfg.Resources.Add(NewCommandFunction(test.saleCmdName, NullCommandExec))
	test.AppCfg.Resources.Add(NewCommandFunction(test.changeCmdName, NullCommandExec))
	test.AppCfg.Resources.Add(NewQueryFunction(test.queryPhotoFunctionName, NullQueryExec))

	var err error

	test.Storage = teststore.NewStorage(test.appName)
	test.StorageProvider = teststore.NewStorageProvider(test.Storage)

	test.AppStructsProvider = Provide(test.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), test.StorageProvider, isequencer.SequencesTrustLevel_0, nil)
	test.AppStructs, err = test.AppStructsProvider.BuiltIn(test.appName)
	if err != nil {
		panic(err)
	}

	return &test
}

func (test *testEnvironment) newEmptyTestRow() (row *rowType) {
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)
	return r
}

func (test *testEnvironment) newTestRow() (row *rowType) {
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)

	test.fillTestRow(r)
	return r
}

func (test *testEnvironment) fillTestRow(row *rowType) {
	row.PutInt8("int8", -2)
	row.PutInt16("int16", -1)
	row.PutInt32("int32", 1)
	row.PutInt64("int64", 2)
	row.PutFloat32("float32", 3)
	row.PutFloat64("float64", 4)
	row.PutBytes("bytes", []byte{1, 2, 3, 4, 5})
	row.PutString("string", "test string") // for unicode test
	row.PutBytes("raw", test.photoRawValue)
	row.PutQName("QName", test.tablePhotos)
	row.PutBool("bool", true)
	row.PutRecordID("RecordID", 7777777)
	row.PutRecordID("RecordID_2", 8888888)

	if err := row.build(); err != nil {
		panic(err)
	}
}

func testRowsIsEqual(t *testing.T, r1, r2 istructs.IRowReader) {
	require := require.New(t)

	row1 := r1.(*rowType)
	row2 := r2.(*rowType)

	require.Equal(row1.QName(), row2.QName())

	require.Equal(row1.ID(), row2.ID())
	require.Equal(row1.Parent(), row2.Parent())
	require.Equal(row1.Container(), row2.Container())
	require.Equal(row1.IsActive(), row2.IsActive())

	row1.dyB.IterateFields(nil, func(name string, val1 any) bool {
		require.True(row2.HasValue(name), name)
		val2 := row2.dyB.Get(name)
		require.Equal(val1, val2, name)
		return true
	})
	row2.dyB.IterateFields(nil, func(name string, _ any) bool {
		require.True(row1.HasValue(name), name)
		return true
	})
}

func rowsIsEqual(r1, r2 istructs.IRowReader) (ok bool, err error) {
	row1 := r1.(*rowType)
	row2 := r2.(*rowType)

	if row1.QName() != row2.QName() {
		return false, fmt.Errorf("row1.QName(): «%v» != row2.QName(): «%v»", row1.QName(), row2.QName())
	}

	row1.dyB.IterateFields(nil, func(name string, val1 any) bool {
		if !row2.HasValue(name) {
			err = fmt.Errorf("row1 has cell «%s», but row2 has't", name)
			return false
		}
		val2 := row2.dyB.Get(name)
		if !assert.ObjectsAreEqual(val1, val2) {
			err = fmt.Errorf("cell «%s» in row1 has value «%v», but in row2 «%v»", name, val1, val2)
			return false
		}
		return true
	})
	if err != nil {
		return false, err
	}

	row2.dyB.IterateFields(nil, func(name string, val2 any) bool {
		if !row1.HasValue(name) {
			err = fmt.Errorf("row2 has cell «%s», but row1 has't", name)
			return false
		}
		return true
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (test *testEnvironment) testTestRow(t *testing.T, row istructs.IRowReader) {
	require := require.New(t)

	require.Equal(int8(-2), row.AsInt8("int8"))
	require.Equal(int16(-1), row.AsInt16("int16"))
	require.Equal(int32(1), row.AsInt32("int32"))
	require.Equal(int64(2), row.AsInt64("int64"))
	require.Equal(float32(3), row.AsFloat32("float32"))
	require.Equal(float64(4), row.AsFloat64("float64"))
	require.Equal([]byte{1, 2, 3, 4, 5}, row.AsBytes("bytes"))
	require.Equal("test string", row.AsString("string"))
	require.Equal(test.photoRawValue, row.AsBytes("raw"))

	require.Equal(test.tablePhotos, row.AsQName("QName"))
	require.True(row.AsBool("bool"))
	require.Equal(istructs.RecordID(7777777), row.AsRecordID("RecordID"))
}

func (test *testEnvironment) newTestCRecord(id istructs.RecordID) *recordType {
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	test.fillTestCRecord(rec, id)
	return rec
}

func (test *testEnvironment) newEmptyTestCRecord() *recordType {
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	return rec
}

func (test *testEnvironment) fillTestCRecord(rec *recordType, id istructs.RecordID) {
	rec.setID(id)
	test.fillTestRow(&rec.rowType)
}

func (test *testEnvironment) testTestCRec(t *testing.T, rec istructs.IRecord, id istructs.RecordID) {
	test.testTestRow(t, rec)

	require := require.New(t)
	require.Equal(id, rec.ID())
}

func (test *testEnvironment) newTestCDoc(id istructs.RecordID) *recordType {
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCDoc)
	test.fillTestCDoc(rec, id)
	return rec
}

func (test *testEnvironment) newEmptyTestCDoc() *recordType {
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCDoc)
	return rec
}

func (test *testEnvironment) fillTestCDoc(doc *recordType, id istructs.RecordID) {
	doc.setID(id)
	test.fillTestRow(&doc.rowType)
}

func (test *testEnvironment) testTestCDoc(t *testing.T, doc istructs.IRecord, id istructs.RecordID) {
	test.testTestRow(t, doc)

	require := require.New(t)
	require.Equal(id, doc.ID())
}

func testRecsIsEqual(t *testing.T, record1, record2 istructs.IRecord) {
	require := require.New(t)

	require.Equal(record1.ID(), record2.ID())
	require.Equal(record1.QName(), record2.QName())

	rec1 := record1.(*recordType)
	rec2 := record2.(*recordType)

	testRowsIsEqual(t, &rec1.rowType, &rec2.rowType)
}

func recsIsEqual(record1, record2 istructs.IRecord) (ok bool, err error) {
	if record1.ID() != record2.ID() {
		return false, fmt.Errorf("record1.ID(): «%d» != record2.ID(): «%d»", record1.ID(), record2.ID())
	}
	if record1.QName() != record2.QName() {
		return false, fmt.Errorf("record1.QName(): «%v» != record2.QName(): «%v»", record1.QName(), record2.QName())
	}

	rec1 := record1.(*recordType)
	rec2 := record2.(*recordType)

	return rowsIsEqual(&rec1.rowType, &rec2.rowType)
}

func (test *testEnvironment) fillTestObject(obj istructs.IObjectBuilder) {
	obj.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
	obj.PutString(test.buyerIdent, test.buyerValue)
	obj.PutInt8(test.ageIdent, test.ageValue)
	obj.PutFloat32(test.heightIdent, test.heightValue)
	obj.PutBool(test.humanIdent, test.humanValue)
	obj.PutBytes(test.photoIdent, test.photoValue)

	basket := obj.ChildBuilder(test.basketIdent)
	basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

	for i := range test.goodCount {
		good := basket.ChildBuilder(test.goodIdent)
		good.PutRecordID(appdef.SystemField_ID, test.tempGoodsID[i])
		good.PutRecordID(test.saleIdent, test.tempSaleID)
		good.PutString(test.nameIdent, test.goodNames[i])
		good.PutInt64(test.codeIdent, test.goodCodes[i])
		good.PutFloat64(test.weightIdent, test.goodWeights[i])
	}

	_, err := obj.Build()
	if err != nil {
		panic(err)
	}
}

func (test *testEnvironment) testTestObject(t *testing.T, value istructs.IObject) {
	require := require.New(t)

	require.Equal(test.buyerValue, value.AsString(test.buyerIdent))
	require.Equal(test.ageValue, value.AsInt8(test.ageIdent))
	require.Equal(test.heightValue, value.AsFloat32(test.heightIdent))
	require.Equal(test.humanValue, value.AsBool(test.humanIdent))
	require.Equal(test.photoValue, value.AsBytes(test.photoIdent))

	var basket istructs.IObject
	for c := range value.Children(test.basketIdent) {
		basket = c
		break
	}
	require.NotNil(basket)

	var cnt int
	for c := range basket.Children(test.goodIdent) {
		require.NotEqual(istructs.NullRecordID, c.AsRecordID(test.saleIdent))
		require.Equal(test.goodNames[cnt], c.AsString(test.nameIdent))
		require.Equal(test.goodCodes[cnt], c.AsInt64(test.codeIdent))
		require.Equal(test.goodWeights[cnt], c.AsFloat64(test.weightIdent))
		cnt++
	}

	require.Equal(test.goodCount, cnt)
}

func (test *testEnvironment) fillTestSecureObject(obj istructs.IObjectBuilder) {
	obj.PutString(test.passwordIdent, "12345")

	_, err := obj.Build()
	if err != nil {
		panic(err)
	}
}

func (test *testEnvironment) testTestSecureObject(t *testing.T, obj istructs.IRowReader) {
	require.New(t).Equal(maskString, obj.AsString(test.passwordIdent))
}

func fillTestCUD(test *testEnvironment, cud *cudType) {
	rec := cud.Create(test.tablePhotos)
	rec.PutRecordID(appdef.SystemField_ID, test.tempPhotoID)
	rec.PutString(test.buyerIdent, test.buyerValue)
	rec.PutInt8(test.ageIdent, test.ageValue)
	rec.PutFloat32(test.heightIdent, test.heightValue)
	rec.PutBool(test.humanIdent, true)
	rec.PutBytes(test.photoIdent, test.photoValue)

	recRem := cud.Create(test.tablePhotoRems)
	recRem.PutRecordID(appdef.SystemField_ID, test.tempRemarkID)
	recRem.PutRecordID(appdef.SystemField_ParentID, test.tempPhotoID)
	recRem.PutString(appdef.SystemField_Container, test.remarkIdent)
	recRem.PutRecordID(test.photoIdent, test.tempPhotoID)
	recRem.PutString(test.remarkIdent, test.remarkValue)
}

func (test *testEnvironment) newTestEvent(pLogOffs, wLogOffs istructs.Offset) *eventType {
	ev := newEvent(test.AppCfg)

	ev.pLogOffs = pLogOffs
	ev.wLogOffs = wLogOffs

	test.fillTestEvent(ev)

	return ev
}

func (test *testEnvironment) fillTestEvent(ev *eventType) {
	ev.setName(test.saleCmdName)

	ev.rawBytes = test.eventRawBytes
	ev.partition = test.partition
	ev.ws = test.workspace
	ev.regTime = test.registeredTime
	ev.sync = true
	ev.device = test.device
	ev.syncTime = test.syncTime

	test.fillTestObject(&ev.argObject)
	test.fillTestSecureObject(&ev.argUnlObj)
	fillTestCUD(test, &ev.cud)

	err := ev.build()
	if err != nil {
		panic(err)
	}
}

func (test *testEnvironment) testTestEvent(t *testing.T, value istructs.IDbEvent, pLogOffs, wLogOffs istructs.Offset, secure bool) {
	require := require.New(t)

	event := value.(*eventType)

	require.Equal(pLogOffs, event.pLogOffs)
	require.Equal(wLogOffs, event.wLogOffs)

	test.testTestObject(t, value.ArgumentObject())
	if secure {
		test.testTestSecureObject(t, &event.argUnlObj)
	}

	var cnt int
	for rec := range value.CUDs {
		require.True(rec.IsNew())
		if rec.QName() == test.tablePhotos {
			test.testPhotoRow(t, rec)
		}
		if rec.QName() == test.tablePhotoRems {
			require.Equal(rec.AsRecordID(appdef.SystemField_ParentID), rec.AsRecordID(test.photoIdent))
			require.Equal(test.remarkValue, rec.AsString(test.remarkIdent))
		}
		cnt++
	}
	require.Equal(2, cnt)
}

func (test *testEnvironment) newEmptyTestEvent() *eventType {
	ev := newEvent(test.AppCfg)
	ev.name = appdef.NullQName
	return ev
}

func (test *testEnvironment) newEmptyTestViewValue() *valueType {
	return newValue(test.AppCfg, test.testViewRecord.name)
}

func (test *testEnvironment) newTestViewValue() *valueType {
	v := test.newEmptyTestViewValue()

	test.fillTestViewValue(v)

	return v
}

func (test *testEnvironment) fillTestViewValue(value *valueType) {
	value.PutString(test.testViewRecord.valueFields.buyer, test.buyerValue)
	value.PutInt8(test.testViewRecord.valueFields.age, test.ageValue)
	value.PutFloat32(test.testViewRecord.valueFields.heights, test.heightValue)
	value.PutBool(test.testViewRecord.valueFields.human, true)
	value.PutBytes(test.testViewRecord.valueFields.photo, test.photoValue)

	r := test.newTestCDoc(100888)
	value.PutRecord(test.testViewRecord.valueFields.record, r)

	e := test.newTestEvent(100500, 1050)
	e.argUnlObj.maskValues()
	value.PutEvent(test.testViewRecord.valueFields.event, e)

	if err := value.build(); err != nil {
		panic(err)
	}
}

func (test *testEnvironment) testTestViewValue(t *testing.T, value istructs.IValue) {
	require := require.New(t)

	require.Equal(test.buyerValue, value.AsString(test.testViewRecord.valueFields.buyer))
	require.Equal(test.ageValue, value.AsInt8(test.testViewRecord.valueFields.age))
	require.Equal(test.heightValue, value.AsFloat32(test.testViewRecord.valueFields.heights))
	require.True(value.AsBool(test.testViewRecord.valueFields.human))
	require.Equal(test.photoValue, value.AsBytes(test.testViewRecord.valueFields.photo))

	r := value.AsRecord(test.testViewRecord.valueFields.record)
	test.testTestCDoc(t, r, 100888)

	e := value.AsEvent(test.testViewRecord.valueFields.event)
	test.testTestEvent(t, e, 100500, 1050, true)
}

func (test *testEnvironment) testCommandsTree(t *testing.T, cmd istructs.IObject) {
	require := require.New(t)

	t.Run("test command", func(t *testing.T) {
		require.NotNil(cmd)

		require.Equal(test.buyerValue, cmd.AsString(test.buyerIdent))
		require.Equal(test.ageValue, cmd.AsInt8(test.ageIdent))
		require.Equal(test.heightValue, cmd.AsFloat32(test.heightIdent))
		require.Equal(test.photoValue, cmd.AsBytes(test.photoIdent))

		require.Equal(test.humanValue, cmd.AsBool(test.humanIdent))
	})

	var basket istructs.IObject

	t.Run("test basket", func(t *testing.T) {
		var names []string
		for name := range cmd.Containers {
			names = append(names, name)
		}
		require.Len(names, 1)
		require.Equal(test.basketIdent, names[0])

		for c := range cmd.Children(test.basketIdent) {
			basket = c
			break
		}
		require.NotNil(basket)

		require.Equal(cmd.AsRecord().ID(), basket.AsRecord().Parent())
	})

	t.Run("test goods", func(t *testing.T) {
		var names []string
		for name := range basket.Containers {
			names = append(names, name)
		}
		require.Len(names, 1)
		require.Equal(test.goodIdent, names[0])

		var goods []istructs.IObject
		for g := range basket.Children(test.goodIdent) {
			goods = append(goods, g)
		}
		require.Len(goods, test.goodCount)

		for i := range test.goodCount {
			good := goods[i]
			require.Equal(basket.AsRecord().ID(), good.AsRecord().Parent())
			require.Equal(cmd.AsRecord().ID(), good.AsRecordID(test.saleIdent))
			require.Equal(test.goodNames[i], good.AsString(test.nameIdent))
			require.Equal(test.goodCodes[i], good.AsInt64(test.codeIdent))
			require.Equal(test.goodWeights[i], good.AsFloat64(test.weightIdent))
		}
	})
}

func (test *testEnvironment) testUnloggedObject(t *testing.T, cmd istructs.IObject) {
	require := require.New(t)

	hasPassword := false
	cmd.Fields(func(iField appdef.IField) bool {
		if hasPassword = iField.Name() == test.passwordIdent; hasPassword {
			return false
		}
		return true
	})
	require.True(hasPassword)

	require.Equal(maskString, cmd.AsString(test.passwordIdent))
}

func (test *testEnvironment) testPhotoRow(t *testing.T, photo istructs.IRowReader) {
	require := require.New(t)
	require.Equal(test.buyerValue, photo.AsString(test.buyerIdent))
	require.Equal(test.ageValue, photo.AsInt8(test.ageIdent))
	require.Equal(test.heightValue, photo.AsFloat32(test.heightIdent))
	require.Equal(test.photoValue, photo.AsBytes(test.photoIdent))
}

func (test *testEnvironment) testDBEvent(t *testing.T, event istructs.IDbEvent) {
	require := require.New(t)

	// test DBEvent event general entities
	require.Equal(test.saleCmdName, event.QName())
	require.Equal(test.registeredTime, event.RegisteredAt())
	require.True(event.Synced())
	require.Equal(test.device, event.DeviceID())
	require.Equal(test.syncTime, event.SyncedAt())

	// test DBEvent commands tree
	test.testCommandsTree(t, event.ArgumentObject())

	t.Run("test DBEvent CUDs", func(t *testing.T) {
		var cuds []istructs.IRowReader
		cnt := 0
		for row := range event.CUDs {
			cuds = append(cuds, row)
			if cnt == 0 {
				require.True(row.IsNew())
				require.Equal(test.tablePhotos, row.QName())
			}
			cnt++
		}
		require.Equal(2, cnt)
		require.Len(cuds, 2)
		test.testPhotoRow(t, cuds[0])
		require.Equal(cuds[0].AsRecordID(appdef.SystemField_ID), cuds[1].AsRecordID(test.photoIdent))
		require.Equal(test.remarkValue, cuds[1].AsString(test.remarkIdent))
	})
}
