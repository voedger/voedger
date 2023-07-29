/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
)

var provider istructs.IAppStructsProvider
var cocaColaDocID istructs.RecordID
var collectionFuncResource istructs.IResource
var cdocFuncResource istructs.IResource
var stateFuncResource istructs.IResource
var testCfgs istructsmem.AppConfigsType

const maxPrepareQueries = 10

func appConfigs() (istructsmem.AppConfigsType, istorage.IAppStorageProvider) {
	cfgs := make(istructsmem.AppConfigsType, 1)
	asf := istorage.ProvideMem()
	appStorageProvider := istorageimpl.Provide(asf)
	// конфиг приложения airs-bp
	adf := appdef.New()
	cfg := cfgs.AddConfig(test.appQName, adf)
	{
		Provide(cfg, adf, false)
	}
	{ // "modify" function
		cfg.Resources.Add(istructsmem.NewCommandFunction(test.modifyCmdName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
	}
	{ // CDoc: articles
		articlesDef := adf.AddCDoc(test.tableArticles)
		articlesDef.
			AddField(test.articleNameIdent, appdef.DataKind_string, true).
			AddField(test.articleNumberIdent, appdef.DataKind_int32, false).
			AddField(test.articleDeptIdent, appdef.DataKind_RecordID, false)
		articlesDef.
			AddContainer(test.tableArticlePrices.Entity(), test.tableArticlePrices, appdef.Occurs(0), appdef.Occurs(100))
	}
	{ // CDoc: departments
		depDef := adf.AddCDoc(test.tableDepartments)
		depDef.
			AddField(test.depNameIdent, appdef.DataKind_string, true).
			AddField(test.depNumberIdent, appdef.DataKind_int32, false)
	}
	{ // CDoc: periods
		periodsDef := adf.AddCDoc(test.tablePeriods)
		periodsDef.
			AddField(test.periodNameIdent, appdef.DataKind_string, true).
			AddField(test.periodNumberIdent, appdef.DataKind_int32, false)
	}
	{ // CDoc: prices
		pricesDef := adf.AddCDoc(test.tablePrices)
		pricesDef.
			AddField(test.priceNameIdent, appdef.DataKind_string, true).
			AddField(test.priceNumberIdent, appdef.DataKind_int32, false)
	}

	{ // CDoc: article prices
		articlesPricesDef := adf.AddCRecord(test.tableArticlePrices)
		articlesPricesDef.
			AddField(test.articlePricesPriceIdIdent, appdef.DataKind_RecordID, true).
			AddField(test.articlePricesPriceIdent, appdef.DataKind_float32, true)
		articlesPricesDef.
			AddContainer(test.tableArticlePriceExceptions.Entity(), test.tableArticlePriceExceptions, appdef.Occurs(0), appdef.Occurs(100))
	}

	{ // CDoc: article price exceptions
		articlesPricesExceptionsDef := adf.AddCRecord(test.tableArticlePriceExceptions)
		articlesPricesExceptionsDef.
			AddField(test.articlePriceExceptionsPeriodIdIdent, appdef.DataKind_RecordID, true).
			AddField(test.articlePriceExceptionsPriceIdent, appdef.DataKind_float32, true)
	}

	// TODO: remove it after https://github.com/voedger/voedger/issues/56
	if _, err := adf.Build(); err != nil {
		panic(err)
	}

	collectionFuncResource = cfg.Resources.QueryResource(qNameQueryCollection)
	cdocFuncResource = cfg.Resources.QueryResource(qNameCDocFunc)
	stateFuncResource = cfg.Resources.QueryResource(qNameQueryState)

	testCfgs = cfgs
	return cfgs, appStorageProvider
}

// Test executes 3 operations with CUDs:
// - insert coca-cola with 2 prices
// - modify coca-cola and 1 price
// - insert fanta with 2 prices
// ...then launches Collection actualizer and waits until it reads all the log.
// Then projection values checked.

func TestBasicUsage_Collection(t *testing.T) {
	require := require.New(t)
	appConfigs, asp := appConfigs()
	provider := istructsmem.Provide(appConfigs, iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp)
	as, err := provider.AppStructs(test.appQName)
	require.NoError(err)

	// Command processor
	actualizer := provideSyncActualizer(context.Background(), as, istructs.PartitionID(1))
	processor := pipeline.NewSyncPipeline(context.Background(), "partition processor", pipeline.WireSyncOperator("actualizer", actualizer))
	defer actualizer.Close()

	// ID and Offset generators
	idGen := newIdsGenerator()

	normalPriceID, happyHourPriceID, _ := insertPrices(require, as, &idGen)
	coldDrinks, _ := insertDepartments(require, as, &idGen)

	{ // CUDs: Insert coca-cola
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.4)
			newArPriceCUD(event, 1, 3, happyHourPriceID, 1.8)
		}))
		err := processor.SendSync(event)
		require.NoError(err)
	}

	cocaColaDocID = idGen.idmap[1]
	cocaColaNormalPriceElementId := idGen.idmap[2]
	cocaColaHappyHourPriceElementId := idGen.idmap[3]

	{ // CUDs: modify coca-cola number and normal price
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			updateArticleCUD(event, as, cocaColaDocID, test.cocaColaNumber2, "Coca-cola")
			updateArPriceCUD(event, as, cocaColaNormalPriceElementId, normalPriceID, 2.2)
		}))
		require.Nil(processor.SendSync(event))
	}

	{ // CUDs: insert fanta
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 7, coldDrinks, test.fantaNumber, "Fanta")
			newArPriceCUD(event, 7, 8, normalPriceID, 2.1)
			newArPriceCUD(event, 7, 9, happyHourPriceID, 1.7)
		}))
		require.Nil(processor.SendSync(event))
	}
	fantaDocID := idGen.idmap[7]
	fantaNormalPriceElementId := idGen.idmap[8]
	fantaHappyHourPriceElementId := idGen.idmap[9]

	// Check expected projection values
	{ // coca-cola
		requireArticle(require, "Coca-cola", test.cocaColaNumber2, as, cocaColaDocID)
		requireArPrice(require, normalPriceID, 2.2, as, cocaColaDocID, cocaColaNormalPriceElementId)
		requireArPrice(require, happyHourPriceID, 1.8, as, cocaColaDocID, cocaColaHappyHourPriceElementId)
	}
	{ // fanta
		requireArticle(require, "Fanta", test.fantaNumber, as, fantaDocID)
		requireArPrice(require, normalPriceID, 2.1, as, fantaDocID, fantaNormalPriceElementId)
		requireArPrice(require, happyHourPriceID, 1.7, as, fantaDocID, fantaHappyHourPriceElementId)
	}

}

func Test_updateChildRecord(t *testing.T) {
	require := require.New(t)
	appConfigs, asp := appConfigs()
	provider := istructsmem.Provide(appConfigs, iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp)
	as, err := provider.AppStructs(test.appQName)
	require.NoError(err)

	// ID and Offset generators
	idGen := newIdsGenerator()

	normalPriceID, _, _ := insertPrices(require, as, &idGen)
	coldDrinks, _ := insertDepartments(require, as, &idGen)

	{ // CUDs: Insert coca-cola
		saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.4)
		}))
	}

	cocaColaNormalPriceElementId := idGen.idmap[2]

	{ // CUDs: modify normal price
		saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			updateArPriceCUD(event, as, cocaColaNormalPriceElementId, normalPriceID, 2.2)
		}))
	}

	rec, err := as.Records().Get(test.workspace, true, cocaColaNormalPriceElementId)
	require.NoError(err)
	require.NotNil(rec)
	require.Equal(float32(2.2), rec.AsFloat32(test.articlePricesPriceIdent))
}

/*
coca-cola

	normal 2.0
	happy_hour 1.5
		exception: holiday 1.0
		exception: new year 0.8

fanta

	normal 2.1
		exception: holiday 1.6
		exception: new year 1.2
	happy_hour 1.6
		exception: holiday 1.1

update coca-cola:

	+exception for normal:
		- exception: holiday 1.8
	update exception for happy_hour:
		- holiday: 0.9
*/
func Test_Collection_3levels(t *testing.T) {
	var err error
	require := require.New(t)

	appConfigs, asp := appConfigs()
	provider = istructsmem.Provide(appConfigs, iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp)
	as, err := provider.AppStructs(test.appQName)
	require.NoError(err)

	// ID and Offset generators
	idGen := newIdsGenerator()

	// Command processor
	actualizer := provideSyncActualizer(context.Background(), as, istructs.PartitionID(1))
	processor := pipeline.NewSyncPipeline(context.Background(), "partition processor", pipeline.WireSyncOperator("actualizer", actualizer))
	defer actualizer.Close()

	normalPriceID, happyHourPriceID, eventPrices := insertPrices(require, as, &idGen)
	coldDrinks, eventDepartments := insertDepartments(require, as, &idGen)
	holiday, newyear, eventPeriods := insertPeriods(require, as, &idGen)

	for _, event := range []istructs.IPLogEvent{eventPrices, eventDepartments, eventPeriods} {
		require.Nil(processor.SendSync(event))
	}

	// insert coca-cola
	{
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.0)
			newArPriceCUD(event, 1, 3, happyHourPriceID, 1.5)
			{
				newArPriceExceptionCUD(event, 3, 4, holiday, 1.0)
				newArPriceExceptionCUD(event, 3, 5, newyear, 0.8)
			}
		}))
		require.Nil(processor.SendSync(event))
	}

	cocaColaDocID = idGen.idmap[1]
	cocaColaNormalPriceElementId := idGen.idmap[2]
	cocaColaHappyHourPriceElementId := idGen.idmap[3]
	cocaColaHappyHourExceptionHolidayElementId := idGen.idmap[4]
	cocaColaHappyHourExceptionNewYearElementId := idGen.idmap[5]

	// insert fanta
	{
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 6, coldDrinks, test.fantaNumber, "Fanta")
			newArPriceCUD(event, 6, 7, normalPriceID, 2.1)
			{
				newArPriceExceptionCUD(event, 7, 9, holiday, 1.6)
				newArPriceExceptionCUD(event, 7, 10, newyear, 1.2)
			}
			newArPriceCUD(event, 6, 8, happyHourPriceID, 1.6)
			{
				newArPriceExceptionCUD(event, 8, 11, holiday, 1.1)
			}
		}))
		require.Nil(processor.SendSync(event))
	}

	fantaDocID := idGen.idmap[6]
	fantaNormalPriceElementId := idGen.idmap[7]
	fantaNormalExceptionHolidayElementId := idGen.idmap[9]
	fantaNormalExceptionNewYearElementId := idGen.idmap[10]
	fantaHappyHourPriceElementId := idGen.idmap[8]
	fantaHappyHourExceptionHolidayElementId := idGen.idmap[11]

	// modify coca-cola
	{
		event := saveEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
			newArPriceExceptionCUD(event, cocaColaNormalPriceElementId, 15, holiday, 1.8)
			updateArPriceExceptionCUD(event, as, cocaColaHappyHourExceptionHolidayElementId, holiday, 0.9)
		}))
		require.Nil(processor.SendSync(event))
	}
	cocaColaNormalExceptionHolidayElementId := idGen.idmap[15]
	require.NotEqual(istructs.NullRecordID, cocaColaNormalExceptionHolidayElementId)

	// Check expected projection values
	{ // coca-cola
		docId := cocaColaDocID
		requireArticle(require, "Coca-cola", test.cocaColaNumber, as, docId)
		requireArPrice(require, normalPriceID, 2.0, as, docId, cocaColaNormalPriceElementId)
		{
			requireArPriceException(require, holiday, 1.8, as, docId, cocaColaNormalExceptionHolidayElementId)
		}
		requireArPrice(require, happyHourPriceID, 1.5, as, docId, cocaColaHappyHourPriceElementId)
		{
			requireArPriceException(require, holiday, 0.9, as, docId, cocaColaHappyHourExceptionHolidayElementId)
			requireArPriceException(require, newyear, 0.8, as, docId, cocaColaHappyHourExceptionNewYearElementId)
		}
	}
	{ // fanta
		docId := fantaDocID
		requireArticle(require, "Fanta", test.fantaNumber, as, docId)
		requireArPrice(require, normalPriceID, 2.1, as, docId, fantaNormalPriceElementId)
		{
			requireArPriceException(require, holiday, 1.6, as, docId, fantaNormalExceptionHolidayElementId)
			requireArPriceException(require, newyear, 1.2, as, docId, fantaNormalExceptionNewYearElementId)
		}
		requireArPrice(require, happyHourPriceID, 1.6, as, docId, fantaHappyHourPriceElementId)
		{
			requireArPriceException(require, holiday, 1.1, as, docId, fantaHappyHourExceptionHolidayElementId)
		}
	}
}

func TestBasicUsage_QueryFunc_Collection(t *testing.T) {
	require := require.New(t)
	// Fill the collection projection
	Test_Collection_3levels(t)

	request := []byte(`{
						"args":{
							"Schema":"test.articles"
						},
						"elements": [
							{
								"fields": ["name", "number"],
								"refs": [["id_department", "name"]]
							},
							{
								"path": "article_prices",
								"fields": ["price"],
								"refs": [["id_prices", "name"]]
							}
						],
						"orderBy":[{"field":"name"}]
					}`)
	serviceChannel := make(iprocbus.ServiceChannel)
	out := newTestSender()

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter)
	authz := iauthnzimpl.NewDefaultAuthorizer()
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, func(ctx context.Context, sender interface{}) queryprocessor.IResultSenderClosable { return out }, provider, maxPrepareQueries, imetrics.Provide(), "vvm",
		authn, authz, testCfgs)
	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.workspace, nil, request, collectionFuncResource, "", sysToken)
	<-out.done

	out.requireNoError(require)
	require.Equal(2, len(out.resultRows)) // 2 rows

	json, err := json.Marshal(out.resultRows)
	require.NoError(err)
	require.NotNil(json)

	{
		row := 0
		require.Equal(2, len(out.resultRows[row])) // 2 elements in a row
		{
			elem := 0
			require.Equal(1, len(out.resultRows[row][elem]))    // 1 element row in 1st element
			require.Equal(3, len(out.resultRows[row][elem][0])) // 3 cell in a row element
			name := out.resultRows[row][elem][0][0]
			number := out.resultRows[row][elem][0][1]
			department := out.resultRows[row][elem][0][2]
			require.Equal("Coca-cola", name)
			require.Equal(int32(10), number)
			require.Equal("Cold Drinks", department)
		}
		{
			elem := 1
			require.Equal(2, len(out.resultRows[row][elem])) // 2 element rows in 2nd element
			{
				elemRow := 0
				require.Equal(2, len(out.resultRows[row][elem][elemRow])) // 2 cells in a row element
				price := out.resultRows[row][elem][elemRow][0]
				pricename := out.resultRows[row][elem][elemRow][1]
				require.Equal(float32(2.0), price)
				require.Equal("Normal Price", pricename)
			}
			{
				elemRow := 1
				require.Equal(2, len(out.resultRows[row][elem][elemRow])) // 2 cells in a row element
				price := out.resultRows[row][elem][elemRow][0]
				pricename := out.resultRows[row][elem][elemRow][1]
				require.Equal(float32(1.5), price)
				require.Equal("Happy Hour Price", pricename)
			}
		}
	}
	{
		row := 1
		require.Equal(2, len(out.resultRows[row])) // 2 elements in a row
		{
			elem := 0
			require.Equal(1, len(out.resultRows[row][elem]))    // 1 element row in 1st element
			require.Equal(3, len(out.resultRows[row][elem][0])) // 3 cell in a row element
			name := out.resultRows[row][elem][0][0]
			number := out.resultRows[row][elem][0][1]
			department := out.resultRows[row][elem][0][2]
			require.Equal("Fanta", name)
			require.Equal(int32(12), number)
			require.Equal("Cold Drinks", department)
		}
		{
			elem := 1
			require.Equal(2, len(out.resultRows[row][elem])) // 2 element rows in 2nd element
			{
				elemRow := 0
				require.Equal(2, len(out.resultRows[row][elem][elemRow])) // 2 cells in a row element
				price := out.resultRows[row][elem][elemRow][0]
				pricename := out.resultRows[row][elem][elemRow][1]
				require.Equal(float32(2.1), price)
				require.Equal("Normal Price", pricename)
			}
			{
				elemRow := 1
				require.Equal(2, len(out.resultRows[row][elem][elemRow])) // 2 cells in a row element
				price := out.resultRows[row][elem][elemRow][0]
				pricename := out.resultRows[row][elem][elemRow][1]
				require.Equal(float32(1.6), price)
				require.Equal("Happy Hour Price", pricename)
			}
		}
	}
}

func TestBasicUsage_QueryFunc_CDoc(t *testing.T) {
	require := require.New(t)
	// Fill the collection projection
	Test_Collection_3levels(t)

	request := fmt.Sprintf(`{
		"args":{
			"ID":%d
		},
		"elements": [
			{
				"fields": ["Result"]
			}
		]
	}`, int64(cocaColaDocID))

	serviceChannel := make(iprocbus.ServiceChannel)
	out := newTestSender()

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter)
	authz := iauthnzimpl.NewDefaultAuthorizer()
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, func(ctx context.Context, sender interface{}) queryprocessor.IResultSenderClosable {
		return out
	}, provider, maxPrepareQueries, imetrics.Provide(), "vvm", authn, authz, testCfgs)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.workspace, nil, []byte(request), cdocFuncResource, "", sysToken)
	<-out.done

	out.requireNoError(require)
	require.Equal(1, len(out.resultRows))          // 1 row
	require.Equal(1, len(out.resultRows[0]))       // 1 element in a row
	require.Equal(1, len(out.resultRows[0][0]))    // 1 row element in an element
	require.Equal(1, len(out.resultRows[0][0][0])) // 1 cell in a row element

	value := out.resultRows[0][0][0][0]
	expected := `{

		"article_prices":[
			{
				"article_price_exceptions":[
					{
						"id_periods":131076,
						"price":1.8,
						"sys.ID":131089,
						"sys.IsActive":true
					}
				],
				"id_prices":131072,
				"price":2,
				"sys.ID":131079,
				"sys.IsActive":true
			},
			{
				"article_price_exceptions":[
					{
						"id_periods":131076,
						"price":0.9,
						"sys.ID":131081,
						"sys.IsActive":true
					},
					{
						"id_periods":131077,
						"price":0.8,
						"sys.ID":131082,
						"sys.IsActive":true
					}
				],
				"id_prices":131073,
				"price":1.5,
				"sys.ID":131080,
				"sys.IsActive":true
			}
		],
		"id_department":131074,
		"name":"Coca-cola",
		"number":10,
		"sys.ID":131078,
		"sys.IsActive":true,
		"xrefs":{
			"test.departments":{
				"131074":{
					"name":"Cold Drinks",
					"number":1,
					"sys.ID":131074,
					"sys.IsActive":true
				}
			},
			"test.periods":{
				"131076":{
					"name":"Holiday",
					"number":1,
					"sys.ID":131076,
					"sys.IsActive":true
				},
				"131077":{
					"name":"New Year",
					"number":2,
					"sys.ID":131077,
					"sys.IsActive":true
				}
			},
			"test.prices":{
				"131072":{
					"name":"Normal Price",
					"number":1,
					"sys.ID":131072,
					"sys.IsActive":true
				},
				"131073":{
					"name":"Happy Hour Price",
					"number":2,
					"sys.ID":131073,
					"sys.IsActive":true
				}
			}
		}

	}`
	require.JSONEq(expected, value.(string))
}

func TestBasicUsage_State(t *testing.T) {
	require := require.New(t)
	// Fill the collection projection
	Test_Collection_3levels(t)

	serviceChannel := make(iprocbus.ServiceChannel)
	out := newTestSender()

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter)
	authz := iauthnzimpl.NewDefaultAuthorizer()
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, func(ctx context.Context, sender interface{}) queryprocessor.IResultSenderClosable {
		return out
	}, provider, maxPrepareQueries, imetrics.Provide(), "vvm", authn, authz, testCfgs)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.workspace, nil, []byte(`{"args":{"After":0},"elements":[{"fields":["State"]}]}`), stateFuncResource, "", sysToken)
	<-out.done

	out.requireNoError(require)
	require.Equal(1, len(out.resultRows))          // 1 row
	require.Equal(1, len(out.resultRows[0]))       // 1 element in a row
	require.Equal(1, len(out.resultRows[0][0]))    // 1 row element in an element
	require.Equal(1, len(out.resultRows[0][0][0])) // 1 cell in a row element
	expected := `{
		"test.article_price_exceptions":{
			"131081":{
				"id_periods":131076,
				"price":0.9,
				"sys.ID":131081,
				"sys.IsActive":true,
				"sys.ParentID":131080
			},
			"131082":{
				"id_periods":131077,
				"price":0.8,
				"sys.ID":131082,
				"sys.IsActive":true,
				"sys.ParentID":131080
			},
			"131085":{
				"id_periods":131076,
				"price":1.6,
				"sys.ID":131085,
				"sys.IsActive":true,
				"sys.ParentID":131084
			},
			"131086":{
				"id_periods":131077,
				"price":1.2,
				"sys.ID":131086,
				"sys.IsActive":true,
				"sys.ParentID":131084
			},
			"131088":{
				"id_periods":131076,
				"price":1.1,
				"sys.ID":131088,
				"sys.IsActive":true,
				"sys.ParentID":131087
			},
			"131089":{
				"id_periods":131076,
				"price":1.8,
				"sys.ID":131089,
				"sys.IsActive":true,
				"sys.ParentID":131079
			}
		},
		"test.article_prices":{
			"131079":{
				"id_prices":131072,
				"price":2,
				"sys.ID":131079,
				"sys.IsActive":true,
				"sys.ParentID":131078
			},
			"131080":{
				"id_prices":131073,
				"price":1.5,
				"sys.ID":131080,
				"sys.IsActive":true,
				"sys.ParentID":131078
			},
			"131084":{
				"id_prices":131072,
				"price":2.1,
				"sys.ID":131084,
				"sys.IsActive":true,
				"sys.ParentID":131083
			},
			"131087":{
				"id_prices":131073,
				"price":1.6,
				"sys.ID":131087,
				"sys.IsActive":true,
				"sys.ParentID":131083
			}
		},
		"test.articles":{
			"131078":{
				"id_department":131074,
				"name":"Coca-cola",
				"number":10,
				"sys.ID":131078,
				"sys.IsActive":true
			},
			"131083":{
				"id_department":131074,
				"name":"Fanta",
				"number":12,
				"sys.ID":131083,
				"sys.IsActive":true
			}
		},
		"test.departments":{
			"131074":{
				"name":"Cold Drinks",
				"number":1,
				"sys.ID":131074,
				"sys.IsActive":true
			},
			"131075":{
				"name":"Hot Drinks",
				"number":2,
				"sys.ID":131075,
				"sys.IsActive":true
			}
		},
		"test.periods":{
			"131076":{
				"name":"Holiday",
				"number":1,
				"sys.ID":131076,
				"sys.IsActive":true
			},
			"131077":{
				"name":"New Year",
				"number":2,
				"sys.ID":131077,
				"sys.IsActive":true
			}
		},
		"test.prices":{
			"131072":{
				"name":"Normal Price",
				"number":1,
				"sys.ID":131072,
				"sys.IsActive":true
			},
			"131073":{
				"name":"Happy Hour Price",
				"number":2,
				"sys.ID":131073,
				"sys.IsActive":true
			}
		}
	}`
	require.JSONEq(expected, out.resultRows[0][0][0][0].(string))
}

func TestState_withAfterArgument(t *testing.T) {
	require := require.New(t)
	// Fill the collection projection
	Test_Collection_3levels(t)

	serviceChannel := make(iprocbus.ServiceChannel)
	out := newTestSender()

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter)
	authz := iauthnzimpl.NewDefaultAuthorizer()
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, time.Now)
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, func(ctx context.Context, sender interface{}) queryprocessor.IResultSenderClosable {
		return out
	}, provider, maxPrepareQueries, imetrics.Provide(), "vvm", authn, authz, testCfgs)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.workspace, nil, []byte(`{"args":{"After":5},"elements":[{"fields":["State"]}]}`), stateFuncResource, "", sysToken)
	<-out.done

	out.requireNoError(require)
	require.Equal(1, len(out.resultRows))          // 1 row
	require.Equal(1, len(out.resultRows[0]))       // 1 element in a row
	require.Equal(1, len(out.resultRows[0][0]))    // 1 row element in an element
	require.Equal(1, len(out.resultRows[0][0][0])) // 1 cell in a row element
	expected := `
	{
		"test.article_price_exceptions":{
			"131081":{
				"id_periods":131076,
				"price":0.9,
				"sys.ID":131081,
				"sys.IsActive":true,
				"sys.ParentID":131080
			},
			"131089":{
				"id_periods":131076,
				"price":1.8,
				"sys.ID":131089,
				"sys.IsActive":true,
				"sys.ParentID":131079
			}
		}
	}`
	require.JSONEq(expected, out.resultRows[0][0][0][0].(string))
}

func createEvent(require *require.Assertions, app istructs.IAppStructs, generator istructs.IDGenerator, bld istructs.IRawEventBuilder) istructs.IPLogEvent {
	rawEvent, buildErr := bld.BuildRawEvent()
	var pLogEvent istructs.IPLogEvent
	var err error
	pLogEvent, err = app.Events().PutPlog(rawEvent, buildErr, generator)
	require.NoError(err)
	return pLogEvent
}

func saveEvent(require *require.Assertions, app istructs.IAppStructs, generator istructs.IDGenerator, bld istructs.IRawEventBuilder) (pLogEvent istructs.IPLogEvent) {
	pLogEvent = createEvent(require, app, generator, bld)
	err := app.Records().Apply(pLogEvent)
	require.NoError(err)
	require.Equal("", pLogEvent.Error().ErrStr())
	return
}

func newPriceCUD(bld istructs.IRawEventBuilder, recordId istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "prices"))
	rec.PutRecordID(appdef.SystemField_ID, recordId)
	rec.PutString(test.priceNameIdent, name)
	rec.PutInt32(test.priceNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newPeriodCUD(bld istructs.IRawEventBuilder, recordId istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "periods"))
	rec.PutRecordID(appdef.SystemField_ID, recordId)
	rec.PutString(test.periodNameIdent, name)
	rec.PutInt32(test.periodNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newDepartmentCUD(bld istructs.IRawEventBuilder, recordId istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "departments"))
	rec.PutRecordID(appdef.SystemField_ID, recordId)
	rec.PutString(test.depNameIdent, name)
	rec.PutInt32(test.depNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newArticleCUD(bld istructs.IRawEventBuilder, articleRecordId, department istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "articles"))
	rec.PutRecordID(appdef.SystemField_ID, articleRecordId)
	rec.PutString(test.articleNameIdent, name)
	rec.PutInt32(test.articleNumberIdent, number)
	rec.PutRecordID(test.articleDeptIdent, department)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArticleCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articleRecordId istructs.RecordID, number int32, name string) {
	rec, err := app.Records().Get(test.workspace, false, articleRecordId)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article %d not found", articleRecordId))
	}
	if err != nil {
		panic(err)
	}
	writer := bld.CUDBuilder().Update(rec)
	writer.PutString(test.articleNameIdent, name)
	writer.PutInt32(test.articleNumberIdent, number)
}

func newArPriceCUD(bld istructs.IRawEventBuilder, articleRecordId, articlePriceRecordId istructs.RecordID, idPrice istructs.RecordID, price float32) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "article_prices"))
	rec.PutRecordID(appdef.SystemField_ID, articlePriceRecordId)
	rec.PutRecordID(appdef.SystemField_ParentID, articleRecordId)
	rec.PutString(appdef.SystemField_Container, "article_prices")
	rec.PutRecordID(test.articlePricesPriceIdIdent, idPrice)
	rec.PutFloat32(test.articlePricesPriceIdent, price)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArPriceCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articlePriceRecordId istructs.RecordID, idPrice istructs.RecordID, price float32) {
	rec, err := app.Records().Get(test.workspace, true, articlePriceRecordId)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article price %d not found", articlePriceRecordId))
	}
	writer := bld.CUDBuilder().Update(rec)
	writer.PutRecordID(test.articlePricesPriceIdIdent, idPrice)
	writer.PutFloat32(test.articlePricesPriceIdent, price)
}

func newArPriceExceptionCUD(bld istructs.IRawEventBuilder, articlePriceRecordId, articlePriceExceptionRecordId, period istructs.RecordID, price float32) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "article_price_exceptions"))
	rec.PutRecordID(appdef.SystemField_ID, articlePriceExceptionRecordId)
	rec.PutRecordID(appdef.SystemField_ParentID, articlePriceRecordId)
	rec.PutString(appdef.SystemField_Container, "article_price_exceptions")
	rec.PutRecordID(test.articlePriceExceptionsPeriodIdIdent, period)
	rec.PutFloat32(test.articlePriceExceptionsPriceIdent, price)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArPriceExceptionCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articlePriceExceptionRecordId, idPeriod istructs.RecordID, price float32) {
	rec, err := app.Records().Get(test.workspace, true, articlePriceExceptionRecordId)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article price exception %d not found", articlePriceExceptionRecordId))
	}

	writer := bld.CUDBuilder().Update(rec)
	writer.PutRecordID(test.articlePriceExceptionsPeriodIdIdent, idPeriod)
	writer.PutFloat32(test.articlePriceExceptionsPriceIdent, price)
}
func insertPrices(require *require.Assertions, app istructs.IAppStructs, idGen *idsGeneratorType) (normalPrice, happyHourPrice istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen.newID, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPriceCUD(event, 51, 1, "Normal Price")
		newPriceCUD(event, 52, 2, "Happy Hour Price")
	}))
	return idGen.idmap[51], idGen.idmap[52], event
}

func insertPeriods(require *require.Assertions, app istructs.IAppStructs, idGen *idsGeneratorType) (holiday, newYear istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen.newID, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPeriodCUD(event, 71, 1, "Holiday")
		newPeriodCUD(event, 72, 2, "New Year")
	}))
	return idGen.idmap[71], idGen.idmap[72], event
}

func insertDepartments(require *require.Assertions, app istructs.IAppStructs, idGen *idsGeneratorType) (coldDrinks istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen.newID, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newDepartmentCUD(event, 61, 1, "Cold Drinks")
		newDepartmentCUD(event, 62, 2, "Hot Drinks")
	}))
	coldDrinks = idGen.idmap[61]
	return
}

type eventCallback func(event istructs.IRawEventBuilder)

func newModify(app istructs.IAppStructs, gen *idsGeneratorType, cb eventCallback) istructs.IRawEventBuilder {
	newOffset := gen.nextOffset()
	builder := app.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				HandlingPartition: test.partition,
				Workspace:         test.workspace,
				QName:             appdef.NewQName("test", "modify"),
				PLogOffset:        newOffset,
				WLogOffset:        newOffset,
			},
		})
	cb(builder)
	return builder
}

func Test_Idempotency(t *testing.T) {
	require := require.New(t)
	appConfigs, asp := appConfigs()
	provider := istructsmem.Provide(appConfigs, iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp)
	as, err := provider.AppStructs(test.appQName)
	require.NoError(err)

	// create command processor
	actualizer := provideSyncActualizer(context.Background(), as, istructs.PartitionID(1))
	processor := pipeline.NewSyncPipeline(context.Background(), "partition processor", pipeline.WireSyncOperator("actualizer", actualizer))
	defer actualizer.Close()

	// ID and Offset generators
	idGen := newIdsGenerator()

	coldDrinks, _ := insertDepartments(require, as, &idGen)

	// CUDs: Insert coca-cola
	event1 := createEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
		newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
	}))
	require.Nil(as.Records().Apply(event1))
	cocaColaDocID = idGen.idmap[1]
	require.Nil(processor.SendSync(event1))

	// CUDs: modify coca-cola number and normal price
	event2 := createEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
		updateArticleCUD(event, as, cocaColaDocID, test.cocaColaNumber2, "Coca-cola")
	}))
	require.Nil(as.Records().Apply(event2))
	require.Nil(processor.SendSync(event2))

	// simulate sending event with the same offset
	idGen.decOffset()
	event2copy := createEvent(require, as, idGen.newID, newModify(as, &idGen, func(event istructs.IRawEventBuilder) {
		updateArticleCUD(event, as, cocaColaDocID, test.cocaColaNumber, "Coca-cola")
	}))
	require.Nil(as.Records().Apply(event2copy))
	require.Nil(processor.SendSync(event2copy))

	// Check expected projection values
	{ // coca-cola
		requireArticle(require, "Coca-cola", test.cocaColaNumber2, as, cocaColaDocID)
	}

}
