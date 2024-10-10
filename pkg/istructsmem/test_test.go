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
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

type (
	testDataType struct {
		appName          appdef.AppQName
		pkgName, pkgPath string

		AppConfigs AppConfigsType
		AppCfg     *AppConfigType
		AppDef     appdef.IAppDef

		StorageProvider istorage.IAppStorageProvider
		Storage         istorage.IAppStorage

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
		ageValue       int32
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
		abstractCDoc appdef.QName
		testRow      appdef.QName
		testObj      appdef.QName

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

var testData = testDataType{
	appName: istructs.AppQName_test1_app1,
	pkgName: "test",
	pkgPath: "test.com/test",

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
	tempRemarkID:   2,

	changeCmdName: appdef.NewQName("test", "change"),

	queryPhotoFunctionName:       appdef.NewQName("test", "QueryPhoto"),
	queryPhotoFunctionParamsName: appdef.NewQName("test", "QueryPhotoParams"),
	photoRawIdent:                "rawPhoto",
	photoRawValue:                bytes.Repeat([]byte{1, 2, 3, 4}, 1024), // 4Kb

	abstractCDoc: appdef.NewQName("test", "abstract"),
	testRow:      appdef.NewQName("test", "Row"),
	testObj:      appdef.NewQName("test", "Obj"),
	testCDoc:     appdef.NewQName("test", "CDoc"),
	testCRec:     appdef.NewQName("test", "Record"),

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

func test() *testDataType {

	prepareAppDef := func() appdef.IAppDefBuilder {
		adb := appdef.New()
		adb.AddPackage(testData.pkgName, testData.pkgPath)

		{
			identData := adb.AddData(testData.dataIdent, appdef.DataKind_string, appdef.NullQName)
			identData.AddConstraints(appdef.MinLen(1), appdef.MaxLen(50)).SetComment("string from 1 to 50 runes")

			photoData := adb.AddData(testData.dataPhoto, appdef.DataKind_bytes, appdef.NullQName)
			photoData.AddConstraints(appdef.MaxLen(1024)).SetComment("up to 1Kb")

			saleParams := adb.AddODoc(testData.saleCmdDocName)
			saleParams.
				AddDataField(testData.buyerIdent, testData.dataIdent, true).
				AddField(testData.ageIdent, appdef.DataKind_int32, false).
				AddField(testData.heightIdent, appdef.DataKind_float32, false).
				AddField(testData.humanIdent, appdef.DataKind_bool, false).
				AddDataField(testData.photoIdent, testData.dataPhoto, false)
			saleParams.
				AddContainer(testData.basketIdent, appdef.NewQName(testData.pkgName, testData.basketIdent), 1, 1)

			basket := adb.AddORecord(appdef.NewQName(testData.pkgName, testData.basketIdent))
			basket.
				AddContainer(testData.goodIdent, appdef.NewQName(testData.pkgName, testData.goodIdent), 0, appdef.Occurs_Unbounded)

			good := adb.AddORecord(appdef.NewQName(testData.pkgName, testData.goodIdent))
			good.
				AddField(testData.saleIdent, appdef.DataKind_RecordID, true).
				AddField(testData.nameIdent, appdef.DataKind_string, true, appdef.MinLen(1)).
				AddField(testData.codeIdent, appdef.DataKind_int64, true).
				AddField(testData.weightIdent, appdef.DataKind_float64, false)

			saleSecureParams := adb.AddObject(testData.saleSecureParsName)
			saleSecureParams.
				AddField(testData.passwordIdent, appdef.DataKind_string, true)

			photoParams := adb.AddObject(testData.queryPhotoFunctionParamsName)
			photoParams.
				AddField(testData.buyerIdent, appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(50)).
				AddField(testData.photoRawIdent, appdef.DataKind_bytes, false, appdef.MaxLen(appdef.MaxFieldLength))
		}

		{
			rec := adb.AddCDoc(testData.tablePhotos)
			rec.
				AddDataField(testData.buyerIdent, testData.dataIdent, true).
				AddField(testData.ageIdent, appdef.DataKind_int32, false).
				AddField(testData.heightIdent, appdef.DataKind_float32, false).
				AddField(testData.humanIdent, appdef.DataKind_bool, false).
				AddDataField(testData.photoIdent, testData.dataPhoto, false)
			rec.
				AddUnique(appdef.NewQName("test", "photos$uniques$buyerIdent"), []string{testData.buyerIdent})
			rec.
				AddContainer(testData.remarkIdent, testData.tablePhotoRems, 0, appdef.Occurs_Unbounded)

			recChild := adb.AddCRecord(testData.tablePhotoRems)
			recChild.
				AddField(testData.photoIdent, appdef.DataKind_RecordID, true).
				AddField(testData.remarkIdent, appdef.DataKind_string, true).
				AddField(testData.emptinessIdent, appdef.DataKind_string, false)
		}

		{
			abstractDoc := adb.AddCDoc(testData.abstractCDoc)
			abstractDoc.SetComment("abstract test cdoc")
			abstractDoc.SetAbstract()
			abstractDoc.
				AddField("int32", appdef.DataKind_int32, false)
		}

		{
			row := adb.AddObject(testData.testRow)
			row.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, appdef.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
		}

		{
			obj := adb.AddObject(testData.testObj)
			obj.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, appdef.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false)
			obj.AddContainer("child", testData.testObj, 0, appdef.Occurs_Unbounded)
		}

		{
			cDoc := adb.AddCDoc(testData.testCDoc)
			cDoc.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, appdef.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false)
			cDoc.
				AddContainer("record", testData.testCRec, 0, appdef.Occurs_Unbounded)

			cRec := adb.AddCRecord(testData.testCRec)
			cRec.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("raw", appdef.DataKind_bytes, false, appdef.MaxLen(appdef.MaxFieldLength)).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false)
		}

		{
			view := adb.AddView(testData.testViewRecord.name)
			view.Key().PartKey().
				AddField(testData.testViewRecord.partFields.partition, appdef.DataKind_int32).
				AddField(testData.testViewRecord.partFields.workspace, appdef.DataKind_int64)
			view.Key().ClustCols().
				AddField(testData.testViewRecord.ccolsFields.device, appdef.DataKind_int32).
				AddField(testData.testViewRecord.ccolsFields.sorter, appdef.DataKind_string, appdef.MaxLen(100))
			view.Value().
				AddField(testData.testViewRecord.valueFields.buyer, appdef.DataKind_string, true).
				AddField(testData.testViewRecord.valueFields.age, appdef.DataKind_int32, false).
				AddField(testData.testViewRecord.valueFields.heights, appdef.DataKind_float32, false).
				AddField(testData.testViewRecord.valueFields.human, appdef.DataKind_bool, false).
				AddDataField(testData.testViewRecord.valueFields.photo, testData.dataPhoto, false).
				AddField(testData.testViewRecord.valueFields.record, appdef.DataKind_Record, false).
				AddField(testData.testViewRecord.valueFields.event, appdef.DataKind_Event, false)
		}

		{
			adb.AddCommand(testData.saleCmdName).SetUnloggedParam(testData.saleSecureParsName).SetParam(testData.saleCmdDocName)
			adb.AddCommand(testData.changeCmdName)
			adb.AddQuery(testData.queryPhotoFunctionName).SetParam(testData.queryPhotoFunctionParamsName)
		}

		return adb
	}

	if testData.AppConfigs == nil {
		testData.AppConfigs = make(AppConfigsType, 1)
		testData.AppCfg = testData.AppConfigs.AddBuiltInAppConfig(testData.appName, prepareAppDef())
		testData.AppCfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
		testData.AppDef = testData.AppCfg.AppDef

		testData.AppCfg.Resources.Add(NewCommandFunction(testData.saleCmdName, NullCommandExec))
		testData.AppCfg.Resources.Add(NewCommandFunction(testData.changeCmdName, NullCommandExec))
		testData.AppCfg.Resources.Add(NewQueryFunction(testData.queryPhotoFunctionName, NullQueryExec))

		var err error

		testData.StorageProvider = istorageimpl.Provide(mem.Provide())

		testData.AppStructsProvider = Provide(testData.AppConfigs, iratesce.TestBucketsFactory, testTokensFactory(), testData.StorageProvider)
		testData.AppStructs, err = testData.AppStructsProvider.BuiltIn(testData.appName)
		if err != nil {
			panic(err)
		}
	}

	return &testData
}

func newEmptyTestRow() (row *rowType) {
	test := test()
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)
	return r
}

func newTestRow() (row *rowType) {
	test := test()
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)

	fillTestRow(r)
	return r
}

func fillTestRow(row *rowType) {
	test := test()

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

	row1.dyB.IterateFields(nil, func(name string, val1 interface{}) bool {
		require.True(row2.HasValue(name), name)
		val2 := row2.dyB.Get(name)
		require.Equal(val1, val2, name)
		return true
	})
	row2.dyB.IterateFields(nil, func(name string, _ interface{}) bool {
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

	row1.dyB.IterateFields(nil, func(name string, val1 interface{}) bool {
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

	row2.dyB.IterateFields(nil, func(name string, val2 interface{}) bool {
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

func testTestRow(t *testing.T, row istructs.IRowReader) {
	require := require.New(t)
	test := test()

	require.Equal(int32(1), row.AsInt32("int32"))
	require.Equal(int64(2), row.AsInt64("int64"))
	require.Equal(float32(3), row.AsFloat32("float32"))
	require.Equal(float64(4), row.AsFloat64("float64"))
	require.Equal([]byte{1, 2, 3, 4, 5}, row.AsBytes("bytes"))
	require.Equal("test string", row.AsString("string"))
	require.EqualValues(test.photoRawValue, row.AsBytes("raw"))

	require.Equal(test.tablePhotos, row.AsQName("QName"))
	require.True(row.AsBool("bool"))
	require.Equal(istructs.RecordID(7777777), row.AsRecordID("RecordID"))
}

func newTestCRecord(id istructs.RecordID) *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	fillTestCRecord(rec, id)
	return rec
}

func newEmptyTestCRecord() *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	return rec
}

func fillTestCRecord(rec *recordType, id istructs.RecordID) {
	rec.setID(id)
	fillTestRow(&rec.rowType)
}

func testTestCRec(t *testing.T, rec istructs.IRecord, id istructs.RecordID) {
	testTestRow(t, rec)

	require := require.New(t)
	require.Equal(id, rec.ID())
}

func newTestCDoc(id istructs.RecordID) *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCDoc)
	fillTestCDoc(rec, id)
	return rec
}

func newEmptyTestCDoc() *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCDoc)
	return rec
}

func fillTestCDoc(doc *recordType, id istructs.RecordID) {
	doc.setID(id)
	fillTestRow(&doc.rowType)
}

func testTestCDoc(t *testing.T, doc istructs.IRecord, id istructs.RecordID) {
	testTestRow(t, doc)

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

func fillTestObject(obj *objectType) {
	test := test()
	obj.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
	obj.PutString(test.buyerIdent, test.buyerValue)
	obj.PutInt32(test.ageIdent, test.ageValue)
	obj.PutFloat32(test.heightIdent, test.heightValue)
	obj.PutBool(test.humanIdent, test.humanValue)
	obj.PutBytes(test.photoIdent, test.photoValue)

	basket := obj.ChildBuilder(test.basketIdent)
	basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

	for i := 0; i < test.goodCount; i++ {
		good := basket.ChildBuilder(test.goodIdent)
		good.PutRecordID(appdef.SystemField_ID, test.tempGoodsID[i])
		good.PutRecordID(test.saleIdent, test.tempSaleID)
		good.PutString(test.nameIdent, test.goodNames[i])
		good.PutInt64(test.codeIdent, test.goodCodes[i])
		good.PutFloat64(test.weightIdent, test.goodWeights[i])
	}

	err := obj.build()
	if err != nil {
		panic(err)
	}
}

func testTestObject(t *testing.T, value istructs.IObject) {
	require := require.New(t)
	test := test()

	require.Equal(test.buyerValue, value.AsString(test.buyerIdent))
	require.Equal(test.ageValue, value.AsInt32(test.ageIdent))
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

func fillTestSecureObject(obj *objectType) {
	test := test()
	obj.PutString(test.passwordIdent, "12345")

	err := obj.build()
	if err != nil {
		panic(err)
	}
}

func testTestSecureObject(t *testing.T, obj *objectType) {
	require := require.New(t)
	test := test()

	require.Equal(maskString, obj.AsString(test.passwordIdent))
}

func fillTestCUD(cud *cudType) {
	test := test()

	rec := cud.Create(test.tablePhotos)
	rec.PutRecordID(appdef.SystemField_ID, test.tempPhotoID)
	rec.PutString(test.buyerIdent, test.buyerValue)
	rec.PutInt32(test.ageIdent, test.ageValue)
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

func newTestEvent(pLogOffs, wLogOffs istructs.Offset) *eventType {
	test := test()
	ev := newEvent(test.AppCfg)

	ev.pLogOffs = pLogOffs
	ev.wLogOffs = wLogOffs

	fillTestEvent(ev)

	return ev
}

func fillTestEvent(ev *eventType) {
	test := test()
	ev.setName(test.saleCmdName)

	ev.rawBytes = test.eventRawBytes
	ev.partition = test.partition
	ev.ws = test.workspace
	ev.regTime = test.registeredTime
	ev.sync = true
	ev.device = test.device
	ev.syncTime = test.syncTime

	fillTestObject(&ev.argObject)
	fillTestSecureObject(&ev.argUnlObj)
	fillTestCUD(&ev.cud)

	err := ev.build()
	if err != nil {
		panic(err)
	}
}

func testTestEvent(t *testing.T, value istructs.IDbEvent, pLogOffs, wLogOffs istructs.Offset, secure bool) {
	require := require.New(t)
	test := test()

	event := value.(*eventType)

	require.Equal(pLogOffs, event.pLogOffs)
	require.Equal(wLogOffs, event.wLogOffs)

	testTestObject(t, value.ArgumentObject())
	if secure {
		testTestSecureObject(t, &event.argUnlObj)
	}

	var cnt int
	for rec := range value.CUDs {
		require.True(rec.IsNew())
		if rec.QName() == test.tablePhotos {
			testPhotoRow(t, rec)
		}
		if rec.QName() == test.tablePhotoRems {
			require.Equal(rec.AsRecordID(appdef.SystemField_ParentID), rec.AsRecordID(test.photoIdent))
			require.Equal(test.remarkValue, rec.AsString(test.remarkIdent))
		}
		cnt++
	}
	require.Equal(2, cnt)
}

func newEmptyTestEvent() *eventType {
	test := test()
	ev := newEvent(test.AppCfg)
	ev.name = appdef.NullQName
	return ev
}

func newEmptyTestViewValue() *valueType {
	test := test()
	return newValue(test.AppCfg, test.testViewRecord.name)
}

func newTestViewValue() *valueType {
	v := newEmptyTestViewValue()

	fillTestViewValue(v)

	return v
}

func fillTestViewValue(value *valueType) {
	test := test()

	value.PutString(test.testViewRecord.valueFields.buyer, test.buyerValue)
	value.PutInt32(test.testViewRecord.valueFields.age, test.ageValue)
	value.PutFloat32(test.testViewRecord.valueFields.heights, test.heightValue)
	value.PutBool(test.testViewRecord.valueFields.human, true)
	value.PutBytes(test.testViewRecord.valueFields.photo, test.photoValue)

	r := newTestCDoc(100888)
	value.PutRecord(test.testViewRecord.valueFields.record, r)

	e := newTestEvent(100500, 1050)
	e.argUnlObj.maskValues()
	value.PutEvent(test.testViewRecord.valueFields.event, e)

	if err := value.build(); err != nil {
		panic(err)
	}
}

func testTestViewValue(t *testing.T, value istructs.IValue) {
	require := require.New(t)
	test := test()

	require.Equal(test.buyerValue, value.AsString(test.testViewRecord.valueFields.buyer))
	require.Equal(test.ageValue, value.AsInt32(test.testViewRecord.valueFields.age))
	require.Equal(test.heightValue, value.AsFloat32(test.testViewRecord.valueFields.heights))
	require.True(value.AsBool(test.testViewRecord.valueFields.human))
	require.Equal(test.photoValue, value.AsBytes(test.testViewRecord.valueFields.photo))

	r := value.AsRecord(test.testViewRecord.valueFields.record)
	testTestCDoc(t, r, 100888)

	e := value.AsEvent(test.testViewRecord.valueFields.event)
	testTestEvent(t, e, 100500, 1050, true)
}
