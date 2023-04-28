/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
)

type (
	testDataType struct {
		appName istructs.AppQName
		pkgName string

		AppConfigs AppConfigsType
		AppCfg     *AppConfigType

		// common event entites
		eventRawBytes     []byte
		partition         istructs.PartitionID
		plogOfs           istructs.Offset
		workspace         istructs.WSID
		wlogOfs           istructs.Offset
		saleCmdName       appdef.QName
		saleCmdDocName    appdef.QName
		saleSecurParsName appdef.QName
		registeredTime    istructs.UnixMilli
		deviceIdent       string
		device            istructs.ConnectedDeviceID
		syncTime          istructs.UnixMilli

		// event command tree entities
		buyerIdent     string
		buyerValue     string
		ageIdent       string
		ageValue       int32
		heightIdent    string
		heightValue    float32
		humanIdent     string
		humanValue     bool
		photoIdent     string
		photoValue     []byte
		remarkIdent    string
		remarkValue    string
		emptiableIdent string
		emptiableValue string
		saleIdent      string
		basketIdent    string
		goodIdent      string
		nameIdent      string
		codeIdent      string
		weightIdent    string
		goodCount      int
		goodNames      []string
		goodCodes      []int64
		goodWeights    []float64

		passwordIdent string

		tempSaleID   istructs.RecordID
		tempBasketID istructs.RecordID
		tempGoodsID  []istructs.RecordID

		// event cuids entities
		tablePhotos    appdef.QName
		tempPhotoID    istructs.RecordID
		tablePhotoRems appdef.QName
		tempRemarkID   istructs.RecordID

		// tested resources
		changeCmdName appdef.QName

		queryPhotoFunctionName         appdef.QName
		queryPhotoFunctionParamsSchema appdef.QName

		// tested rows
		testRow appdef.QName

		// tested records
		testCDoc appdef.QName
		testCRec appdef.QName

		// tested viewRecords
		testViewRecord testViewRecordType
	}

	testViewRecordType struct {
		name, valueName appdef.QName
		partFields      testViewRecordPartKeyFieldsType
		clustFields     testViewRecordClustKeyFieldsType
		valueFields     testViewRecordValueFieldsType
	}

	testViewRecordPartKeyFieldsType struct {
		partition string
		workspace string
	}

	testViewRecordClustKeyFieldsType struct {
		device string
		sorter string
	}

	testViewRecordValueFieldsType struct {
		buyer   string
		age     string
		heights string
		human   string
		photo   string
		record  string
		event   string
	}
)

var data = testDataType{
	appName: istructs.AppQName_test1_app1,
	pkgName: "test",

	eventRawBytes:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	partition:         55,
	plogOfs:           10000,
	workspace:         1234,
	wlogOfs:           1000,
	saleCmdName:       appdef.NewQName("test", "sales"),
	saleCmdDocName:    appdef.NewQName("test", "saleArgs"),
	saleSecurParsName: appdef.NewQName("test", "saleSecureArgs"),
	registeredTime:    100500,
	deviceIdent:       "Device",
	device:            762,
	syncTime:          1005001,

	buyerIdent:     "Buyer",
	buyerValue:     "Карлосон 哇\"呀呀", // to test unicode issues
	ageIdent:       "Age",
	ageValue:       33,
	heightIdent:    "Height",
	heightValue:    1.75,
	humanIdent:     "isHuman",
	humanValue:     true,
	photoIdent:     "Photo",
	photoValue:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 9, 8, 7, 6, 4, 4, 3, 2, 1, 0},
	remarkIdent:    "Remark",
	remarkValue:    "remark text Примечание",
	emptiableIdent: "Emptiable",
	emptiableValue: "to be emptied",

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

	tablePhotos:    appdef.NewQName("test", "photos"),
	tempPhotoID:    1,
	tablePhotoRems: appdef.NewQName("test", "photoRems"),
	tempRemarkID:   2,

	changeCmdName: appdef.NewQName("test", "change"),

	queryPhotoFunctionName:         appdef.NewQName("test", "QueryPhoto"),
	queryPhotoFunctionParamsSchema: appdef.NewQName("test", "QueryPhotoParams"),

	testRow:  appdef.NewQName("test", "Row"),
	testCDoc: appdef.NewQName("test", "CDoc"),
	testCRec: appdef.NewQName("test", "Record"),

	testViewRecord: testViewRecordType{
		name: appdef.NewQName("test", "ViewPhotos"),
		partFields: testViewRecordPartKeyFieldsType{
			partition: "partition",
			workspace: "workspace",
		},
		clustFields: testViewRecordClustKeyFieldsType{
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

var test func() *testDataType = func() *testDataType {

	prepareAppDef := func() appdef.IAppDefBuilder {
		appDef := appdef.New()

		{
			saleParamsSchema := appDef.Add(data.saleCmdDocName, appdef.DefKind_ODoc)
			saleParamsSchema.
				AddField(data.buyerIdent, appdef.DataKind_string, true).
				AddField(data.ageIdent, appdef.DataKind_int32, false).
				AddField(data.heightIdent, appdef.DataKind_float32, false).
				AddField(data.humanIdent, appdef.DataKind_bool, false).
				AddField(data.photoIdent, appdef.DataKind_bytes, false).
				AddContainer(data.basketIdent, appdef.NewQName(data.pkgName, data.basketIdent), 1, 1)

			basketSchema := appDef.Add(appdef.NewQName(data.pkgName, data.basketIdent), appdef.DefKind_ORecord)
			basketSchema.
				AddContainer(data.goodIdent, appdef.NewQName(data.pkgName, data.goodIdent), 0, appdef.Occurs_Unbounded)

			goodSchema := appDef.Add(appdef.NewQName(data.pkgName, data.goodIdent), appdef.DefKind_ORecord)
			goodSchema.
				AddField(data.saleIdent, appdef.DataKind_RecordID, true).
				AddField(data.nameIdent, appdef.DataKind_string, true).
				AddField(data.codeIdent, appdef.DataKind_int64, true).
				AddField(data.weightIdent, appdef.DataKind_float64, false)

			saleSecurParamsSchema := appDef.Add(data.saleSecurParsName, appdef.DefKind_Object)
			saleSecurParamsSchema.
				AddField(data.passwordIdent, appdef.DataKind_string, true)

			photoParamsSchema := appDef.Add(data.queryPhotoFunctionParamsSchema, appdef.DefKind_Object)
			photoParamsSchema.
				AddField(data.buyerIdent, appdef.DataKind_string, true)
		}

		{
			recSchema := appDef.Add(data.tablePhotos, appdef.DefKind_CDoc)
			recSchema.
				AddField(data.buyerIdent, appdef.DataKind_string, true).
				AddField(data.ageIdent, appdef.DataKind_int32, false).
				AddField(data.heightIdent, appdef.DataKind_float32, false).
				AddField(data.humanIdent, appdef.DataKind_bool, false).
				AddField(data.photoIdent, appdef.DataKind_bytes, false).
				AddContainer(data.remarkIdent, data.tablePhotoRems, 0, appdef.Occurs_Unbounded)

			recSchemaChild := appDef.Add(data.tablePhotoRems, appdef.DefKind_CRecord)
			recSchemaChild.
				AddField(data.photoIdent, appdef.DataKind_RecordID, true).
				AddField(data.remarkIdent, appdef.DataKind_string, true).
				AddField(data.emptiableIdent, appdef.DataKind_string, false)
		}

		{
			rowSchema := appDef.Add(data.testRow, appdef.DefKind_Element)
			rowSchema.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddField("RecordID_2", appdef.DataKind_RecordID, false)
		}

		{
			cDocSchema := appDef.Add(data.testCDoc, appdef.DefKind_CDoc)
			cDocSchema.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false).
				AddContainer("record", data.testCRec, 0, appdef.Occurs_Unbounded)

			cRecSchema := appDef.Add(data.testCRec, appdef.DefKind_CRecord)
			cRecSchema.
				AddField("int32", appdef.DataKind_int32, false).
				AddField("int64", appdef.DataKind_int64, false).
				AddField("float32", appdef.DataKind_float32, false).
				AddField("float64", appdef.DataKind_float64, false).
				AddField("bytes", appdef.DataKind_bytes, false).
				AddField("string", appdef.DataKind_string, false).
				AddField("QName", appdef.DataKind_QName, false).
				AddField("bool", appdef.DataKind_bool, false).
				AddField("RecordID", appdef.DataKind_RecordID, false)
		}

		{
			viewSchema := appDef.AddView(data.testViewRecord.name)
			viewSchema.
				AddPartField(data.testViewRecord.partFields.partition, appdef.DataKind_int32).
				AddPartField(data.testViewRecord.partFields.workspace, appdef.DataKind_int64).
				AddClustColumn(data.testViewRecord.clustFields.device, appdef.DataKind_int32).
				AddClustColumn(data.testViewRecord.clustFields.sorter, appdef.DataKind_string).
				AddValueField(data.testViewRecord.valueFields.buyer, appdef.DataKind_string, true).
				AddValueField(data.testViewRecord.valueFields.age, appdef.DataKind_int32, false).
				AddValueField(data.testViewRecord.valueFields.heights, appdef.DataKind_float32, false).
				AddValueField(data.testViewRecord.valueFields.human, appdef.DataKind_bool, false).
				AddValueField(data.testViewRecord.valueFields.photo, appdef.DataKind_bytes, false).
				AddValueField(data.testViewRecord.valueFields.record, appdef.DataKind_Record, false).
				AddValueField(data.testViewRecord.valueFields.event, appdef.DataKind_Event, false)
			data.testViewRecord.valueName = viewSchema.ValueSchema().QName()
		}

		return appDef
	}

	prepareConfig := func(cfg *AppConfigType) {

		sp := istorageimpl.Provide(istorage.ProvideMem())
		storage, err := sp.AppStorage(data.appName)
		if err != nil {
			panic(err)
		}

		cfg.Resources.Add(NewCommandFunction(data.saleCmdName, data.saleCmdDocName, data.saleSecurParsName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewCommandFunction(data.changeCmdName, appdef.NullQName, appdef.NullQName, appdef.NullQName, NullCommandExec))
		cfg.Resources.Add(NewQueryFunction(data.queryPhotoFunctionName, data.queryPhotoFunctionParamsSchema, appdef.NullQName, NullQueryExec))

		if err := cfg.prepare(iratesce.TestBucketsFactory(), storage); err != nil {
			panic(err)
		}
	}

	if data.AppConfigs == nil {
		data.AppConfigs = make(AppConfigsType, 1)
		data.AppCfg = data.AppConfigs.AddConfig(data.appName, prepareAppDef())
		prepareConfig(data.AppCfg)
	}

	return &data
}

func newEmptyTestRow() (row *rowType) {
	test := test()
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)
	return &r
}

func newTestRow() (row *rowType) {
	test := test()
	r := newRow(test.AppCfg)
	r.setQName(test.testRow)

	fillTestRow(&r)
	return &r
}

func fillTestRow(row *rowType) {
	test := test()

	row.PutInt32("int32", 1)
	row.PutInt64("int64", 2)
	row.PutFloat32("float32", 3)
	row.PutFloat64("float64", 4)
	row.PutBytes("bytes", []byte{1, 2, 3, 4, 5})
	row.PutString("string", "Строка") // for unicode test
	row.PutQName("QName", test.tablePhotos)
	row.PutBool("bool", true)
	row.PutRecordID("RecordID", 7777777)

	if _, err := row.build(); err != nil {
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
		require.True(row2.hasValue(name), name)
		val2 := row2.dyB.Get(name)
		require.Equal(val1, val2, name)
		return true
	})
	row2.dyB.IterateFields(nil, func(name string, _ interface{}) bool {
		require.True(row1.hasValue(name), name)
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
		if !row2.hasValue(name) {
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
		if !row1.hasValue(name) {
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
	require.Equal("Строка", row.AsString("string"))
	require.Equal(test.tablePhotos, row.AsQName("QName"))
	require.Equal(true, row.AsBool("bool"))
	require.Equal(istructs.RecordID(7777777), row.AsRecordID("RecordID"))
}

func newTestCRecord(id istructs.RecordID) *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	fillTestCRecord(&rec, id)
	return &rec
}

func newEmptyTestCRecord() *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCRec)
	return &rec
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
	fillTestCDoc(&rec, id)
	return &rec
}

func newEmptyTestCDoc() *recordType {
	test := test()
	rec := newRecord(test.AppCfg)
	rec.setQName(test.testCDoc)
	return &rec
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

func fillTestObject(obj *elementType) {
	test := test()
	obj.PutRecordID(appdef.SystemField_ID, test.tempSaleID)
	obj.PutString(test.buyerIdent, test.buyerValue)
	obj.PutInt32(test.ageIdent, test.ageValue)
	obj.PutFloat32(test.heightIdent, test.heightValue)
	obj.PutBool(test.humanIdent, test.humanValue)
	obj.PutBytes(test.photoIdent, test.photoValue)

	basket := obj.ElementBuilder(test.basketIdent)
	basket.PutRecordID(appdef.SystemField_ID, test.tempBasketID)

	for i := 0; i < test.goodCount; i++ {
		good := basket.ElementBuilder(test.goodIdent)
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

	var basket istructs.IElement
	value.Elements(test.basketIdent, func(el istructs.IElement) { basket = el })
	require.NotNil(basket)

	var cnt int
	basket.Elements(test.goodIdent, func(el istructs.IElement) {
		require.NotEqual(istructs.NullRecordID, el.AsRecordID(test.saleIdent))
		require.Equal(test.goodNames[cnt], el.AsString(test.nameIdent))
		require.Equal(test.goodCodes[cnt], el.AsInt64(test.codeIdent))
		require.Equal(test.goodWeights[cnt], el.AsFloat64(test.weightIdent))
		cnt++
	})

	require.Equal(test.goodCount, cnt)
}

func fillTestUnloggedObject(obj *elementType) {
	test := test()
	obj.PutString(test.passwordIdent, "12345")

	err := obj.build()
	if err != nil {
		panic(err)
	}
}

func testTestUnloggedObject(t *testing.T, obj *elementType) {
	require := require.New(t)
	test := test()

	require.Equal(obj.AsString(test.passwordIdent), maskString)
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

func newTestEvent(pLogOffs, wLogOffs istructs.Offset) *dbEventType {
	test := test()
	ev := newDbEvent(test.AppCfg)

	ev.pLogOffs = pLogOffs
	ev.wLogOffs = wLogOffs

	fillTestEvent(&ev)

	return &ev
}

func fillTestEvent(ev *dbEventType) {
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
	fillTestUnloggedObject(&ev.argUnlObj)
	fillTestCUD(&ev.cud)
	// fill_test_CUD(&ev.resCUD) TODO:

	err := ev.build()
	if err != nil {
		panic(err)
	}
}

func testTestEvent(t *testing.T, value istructs.IDbEvent, pLogOffs, wLogOffs istructs.Offset, unlogged bool) {
	require := require.New(t)
	test := test()

	event := value.(*dbEventType)

	require.Equal(pLogOffs, event.pLogOffs)
	require.Equal(wLogOffs, event.wLogOffs)

	testTestObject(t, value.ArgumentObject())
	if unlogged {
		testTestUnloggedObject(t, &event.argUnlObj)
	}

	var cnt int
	value.CUDs(func(rec istructs.ICUDRow) error {
		require.True(rec.IsNew())
		if rec.QName() == test.tablePhotos {
			testPhotoRow(t, rec)
		}
		if rec.QName() == test.tablePhotoRems {
			require.Equal(rec.AsRecordID(appdef.SystemField_ParentID), rec.AsRecordID(test.photoIdent))
			require.Equal(test.remarkValue, rec.AsString(test.remarkIdent))
		}
		cnt++
		return nil
	})
	require.Equal(2, cnt)
}

func newEmptyTestEvent() *dbEventType {
	test := test()
	ev := newDbEvent(test.AppCfg)
	ev.name = appdef.NullQName
	return &ev
}

func newEmptyViewValue() (val *rowType) {
	test := test()
	v := newRow(test.AppCfg)
	v.setQName(test.testViewRecord.valueName)
	return &v
}

func newTestViewValue() (val *rowType) {
	test := test()
	v := newRow(test.AppCfg)

	v.setQName(test.testViewRecord.valueName)
	fillTestViewValue(&v)

	return &v
}

func fillTestViewValue(value *rowType) {
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

	if _, err := value.build(); err != nil {
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
