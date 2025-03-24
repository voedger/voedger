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

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/processors/actualizers"
	queryprocessor "github.com/voedger/voedger/pkg/processors/query"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/vvm/engines"
)

var cocaColaDocID istructs.RecordID
var qNameTestWSKind = appdef.NewQName(appdef.SysPackage, "test_ws")

const maxPrepareQueries = 10

func deployTestApp(t *testing.T) (appParts appparts.IAppPartitions, appStructs istructs.IAppStructs, cleanup func(),
	statelessResources istructsmem.IStatelessResources) {
	require := require.New(t)

	cfgs := make(istructsmem.AppConfigsType, 1)
	asp := istorageimpl.Provide(mem.Provide(coreutils.MockTime))

	// airs-bp application config. For tests «istructs.AppQName_test1_app1» is used
	adb := builder.New()
	statelessResources = istructsmem.NewStatelessResources()
	cfg := cfgs.AddBuiltInAppConfig(test.appQName, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)

	adb.AddPackage("test", "test.org/test")
	wsName := appdef.NewQName(appdef.SysPackage, "test_wsWS")
	wsb := adb.AddWorkspace(wsName)

	{
		wsb.AddCDoc(qNameTestWSKind).SetSingleton()
		wsb.SetDescriptor(qNameTestWSKind)
		wsdescutil.AddWorkspaceDescriptorStubDef(wsb)

		// this should be done in tests only. Runtime -> the projector is defined in sys.vsql already
		prj := wsb.AddProjector(QNameProjectorCollection)
		prj.SetSync(true).
			Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update},
			filter.Types(appdef.TypeKind_CDoc, appdef.TypeKind_CRecord))
		prj.Intents().
			Add(sys.Storage_View, QNameCollectionView) // this view will be added below
	}
	{
		// fill IAppDef with funcs. That is done here manually because we o not use sys.vsql here
		qNameCollectionParams := appdef.NewQName(appdef.SysPackage, "CollectionParams")

		// will add func definitions to AppDef manually because local test does not use sql. In runtime these definitions will come from sys.vsql
		wsb.AddObject(qNameCollectionParams).
			AddField(Field_Schema, appdef.DataKind_string, true).
			AddField(field_ID, appdef.DataKind_RecordID, false)

		wsb.AddQuery(qNameQueryCollection).
			SetParam(qNameCollectionParams).
			SetResult(appdef.QNameANY)

		qNameGetCDocParams := appdef.NewQName(appdef.SysPackage, "GetCDocParams")
		wsb.AddObject(qNameGetCDocParams).
			AddField(field_ID, appdef.DataKind_int64, true)

		qNameGetCDocResult := appdef.NewQName(appdef.SysPackage, "GetCDocResult")
		wsb.AddObject(qNameGetCDocResult).
			AddField("Result", appdef.DataKind_string, true)

		wsb.AddQuery(qNameQueryGetCDoc).
			SetParam(qNameGetCDocParams).
			SetResult(qNameGetCDocResult)

		qNameStateParams := appdef.NewQName(appdef.SysPackage, "StateParams")
		wsb.AddObject(qNameStateParams).
			AddField(field_After, appdef.DataKind_int64, true)

		qNameStateResult := appdef.NewQName(appdef.SysPackage, "StateResult")
		wsb.AddObject(qNameStateResult).
			AddField(field_State, appdef.DataKind_string, true)

		wsb.AddQuery(qNameQueryState).
			SetParam(qNameStateParams).
			SetResult(qNameStateResult)

		wsb.AddRole(iauthnz.QNameRoleAuthenticatedUser)
		wsb.AddRole(iauthnz.QNameRoleEveryone)
		wsb.AddRole(iauthnz.QNameRoleSystem)
		wsb.AddRole(iauthnz.QNameRoleAnonymous)
		wsb.AddRole(iauthnz.QNameRoleProfileOwner)

	}
	{ // "modify" function
		wsb.AddCommand(test.modifyCmdName)
		cfg.Resources.Add(istructsmem.NewCommandFunction(test.modifyCmdName, istructsmem.NullCommandExec))
	}
	{ // CDoc: articles
		articles := wsb.AddCDoc(test.tableArticles)
		articles.
			AddField(test.articleNameIdent, appdef.DataKind_string, true).
			AddField(test.articleNumberIdent, appdef.DataKind_int32, false).
			AddField(test.articleDeptIdent, appdef.DataKind_RecordID, false)
		articles.
			AddContainer(test.tableArticlePrices.Entity(), test.tableArticlePrices, appdef.Occurs(0), appdef.Occurs(100))
	}
	{ // CDoc: departments
		departments := wsb.AddCDoc(test.tableDepartments)
		departments.
			AddField(test.depNameIdent, appdef.DataKind_string, true).
			AddField(test.depNumberIdent, appdef.DataKind_int32, false)
	}
	{ // CDoc: periods
		periods := wsb.AddCDoc(test.tablePeriods)
		periods.
			AddField(test.periodNameIdent, appdef.DataKind_string, true).
			AddField(test.periodNumberIdent, appdef.DataKind_int32, false)
	}
	{ // CDoc: prices
		prices := wsb.AddCDoc(test.tablePrices)
		prices.
			AddField(test.priceNameIdent, appdef.DataKind_string, true).
			AddField(test.priceNumberIdent, appdef.DataKind_int32, false)
	}

	{ // CDoc: article prices
		articlesPrices := wsb.AddCRecord(test.tableArticlePrices)
		articlesPrices.
			AddField(test.articlePricesPriceIdIdent, appdef.DataKind_RecordID, true).
			AddField(test.articlePricesPriceIdent, appdef.DataKind_float32, true)
		articlesPrices.
			AddContainer(test.tableArticlePriceExceptions.Entity(), test.tableArticlePriceExceptions, appdef.Occurs(0), appdef.Occurs(100))
	}

	{ // CDoc: article price exceptions
		articlesPricesExceptions := wsb.AddCRecord(test.tableArticlePriceExceptions)
		articlesPricesExceptions.
			AddField(test.articlePriceExceptionsPeriodIdIdent, appdef.DataKind_RecordID, true).
			AddField(test.articlePriceExceptionsPriceIdent, appdef.DataKind_float32, true)
	}

	// kept here to keep local tests working without sql
	actualizers.ProvideViewDef(wsb, QNameCollectionView, func(b appdef.IViewBuilder) {
		b.Key().PartKey().AddField(Field_PartKey, appdef.DataKind_int32)
		b.Key().ClustCols().
			AddField(Field_DocQName, appdef.DataKind_QName).
			AddRefField(Field_DocID).
			AddRefField(field_ElementID)
		b.Value().
			AddField(Field_Record, appdef.DataKind_Record, true).
			AddField(state.ColOffset, appdef.DataKind_int64, true)
	})

	// TODO: remove it after https://github.com/voedger/voedger/issues/56
	appDef, err := adb.Build()
	require.NoError(err)

	Provide(statelessResources)

	appStructsProvider := istructsmem.Provide(cfgs, iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp)

	secretReader := isecretsimpl.ProvideSecretReader()
	n10nBroker, n10nBrokerCleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:                1000,
		ChannelsPerSubject:      10,
		Subscriptions:           1000,
		SubscriptionsPerSubject: 10,
	}, coreutils.NewITime())

	appParts, appPartsCleanup, err := appparts.New2(context.Background(), appStructsProvider,
		actualizers.NewSyncActualizerFactoryFactory(actualizers.ProvideSyncActualizerFactory(), secretReader, n10nBroker, statelessResources),
		appparts.NullActualizerRunner,
		appparts.NullSchedulerRunner,
		engines.ProvideExtEngineFactories(
			engines.ExtEngineFactoriesConfig{
				AppConfigs:         cfgs,
				StatelessResources: statelessResources,
				WASMConfig:         iextengine.WASMFactoryConfig{},
			}, "", imetrics.Provide()),
		iratesce.TestBucketsFactory)
	require.NoError(err)
	appParts.DeployApp(test.appQName, nil, appDef, test.totalPartitions, test.appEngines, 1)
	appParts.DeployAppPartitions(test.appQName, []istructs.PartitionID{test.partition})

	// create stub for cdoc.sys.WorkspaceDescriptor to make query processor work
	as, err := appStructsProvider.BuiltIn(test.appQName)
	require.NoError(err)
	err = wsdescutil.CreateCDocWorkspaceDescriptorStub(as, test.partition, test.workspace, wsdescutil.TestWsDescName, 1, 1)
	require.NoError(err)

	cleanup = func() {
		appPartsCleanup()
		n10nBrokerCleanup()
	}

	return appParts, as, cleanup, statelessResources
}

// Test executes 3 operations with CUDs:
// - insert coca-cola with 2 prices
// - modify coca-cola and 1 price
// - insert fanta with 2 prices
// ...then launches Collection actualizer and waits until it reads all the log.
// Then projection values checked.
func TestBasicUsage_Collection(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	// Command processor
	processor := testProcessor(appParts)

	// ID and Offset generators
	idGen := newIdsGenerator()

	normalPriceID, happyHourPriceID, _ := insertPrices(require, appStructs, &idGen)
	coldDrinks, _ := insertDepartments(require, appStructs, &idGen)

	{ // CUDs: Insert coca-cola
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
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
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			updateArticleCUD(event, appStructs, cocaColaDocID, test.cocaColaNumber2, "Coca-cola")
			updateArPriceCUD(event, appStructs, cocaColaNormalPriceElementId, normalPriceID, 2.2)
		}))
		require.NoError(processor.SendSync(event))
	}

	{ // CUDs: insert fanta
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 7, coldDrinks, test.fantaNumber, "Fanta")
			newArPriceCUD(event, 7, 8, normalPriceID, 2.1)
			newArPriceCUD(event, 7, 9, happyHourPriceID, 1.7)
		}))
		require.NoError(processor.SendSync(event))
	}
	fantaDocID := idGen.idmap[7]
	fantaNormalPriceElementId := idGen.idmap[8]
	fantaHappyHourPriceElementId := idGen.idmap[9]

	// Check expected projection values
	{ // coca-cola
		requireArticle(require, "Coca-cola", test.cocaColaNumber2, appStructs, cocaColaDocID)
		requireArPrice(require, normalPriceID, 2.2, appStructs, cocaColaDocID, cocaColaNormalPriceElementId)
		requireArPrice(require, happyHourPriceID, 1.8, appStructs, cocaColaDocID, cocaColaHappyHourPriceElementId)
	}
	{ // fanta
		requireArticle(require, "Fanta", test.fantaNumber, appStructs, fantaDocID)
		requireArPrice(require, normalPriceID, 2.1, appStructs, fantaDocID, fantaNormalPriceElementId)
		requireArPrice(require, happyHourPriceID, 1.7, appStructs, fantaDocID, fantaHappyHourPriceElementId)
	}

}

func Test_updateChildRecord(t *testing.T) {
	require := require.New(t)

	_, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	// ID and Offset generators
	idGen := newIdsGenerator()

	normalPriceID, _, _ := insertPrices(require, appStructs, &idGen)
	coldDrinks, _ := insertDepartments(require, appStructs, &idGen)

	{ // CUDs: Insert coca-cola
		saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.4)
		}))
	}

	cocaColaNormalPriceElementId := idGen.idmap[2]

	{ // CUDs: modify normal price
		saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			updateArPriceCUD(event, appStructs, cocaColaNormalPriceElementId, normalPriceID, 2.2)
		}))
	}

	rec, err := appStructs.Records().Get(test.workspace, true, cocaColaNormalPriceElementId)
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

func cp_Collection_3levels(t *testing.T, appParts appparts.IAppPartitions, appStructs istructs.IAppStructs) {
	require := require.New(t)

	// Command processor
	processor := testProcessor(appParts)

	// ID and Offset generators
	idGen := newIdsGenerator()

	normalPriceID, happyHourPriceID, eventPrices := insertPrices(require, appStructs, &idGen)
	coldDrinks, eventDepartments := insertDepartments(require, appStructs, &idGen)
	holiday, newyear, eventPeriods := insertPeriods(require, appStructs, &idGen)

	for _, event := range []istructs.IPLogEvent{eventPrices, eventDepartments, eventPeriods} {
		require.NoError(processor.SendSync(event))
	}

	// insert coca-cola
	{
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.0)
			newArPriceCUD(event, 1, 3, happyHourPriceID, 1.5)
			{
				newArPriceExceptionCUD(event, 3, 4, holiday, 1.0)
				newArPriceExceptionCUD(event, 3, 5, newyear, 0.8)
			}
		}))
		require.NoError(processor.SendSync(event))
	}

	cocaColaDocID = idGen.idmap[1]
	cocaColaNormalPriceElementId := idGen.idmap[2]
	cocaColaHappyHourPriceElementId := idGen.idmap[3]
	cocaColaHappyHourExceptionHolidayElementId := idGen.idmap[4]
	cocaColaHappyHourExceptionNewYearElementId := idGen.idmap[5]

	// insert fanta
	{
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
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
		require.NoError(processor.SendSync(event))
	}

	fantaDocID := idGen.idmap[6]
	fantaNormalPriceElementId := idGen.idmap[7]
	fantaNormalExceptionHolidayElementId := idGen.idmap[9]
	fantaNormalExceptionNewYearElementId := idGen.idmap[10]
	fantaHappyHourPriceElementId := idGen.idmap[8]
	fantaHappyHourExceptionHolidayElementId := idGen.idmap[11]

	// modify coca-cola
	{
		event := saveEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
			newArPriceExceptionCUD(event, cocaColaNormalPriceElementId, 15, holiday, 1.8)
			updateArPriceExceptionCUD(event, appStructs, cocaColaHappyHourExceptionHolidayElementId, holiday, 0.9)
		}))
		require.NoError(processor.SendSync(event))
	}
	cocaColaNormalExceptionHolidayElementId := idGen.idmap[15]
	require.NotEqual(istructs.NullRecordID, cocaColaNormalExceptionHolidayElementId)

	// Check expected projection values
	{ // coca-cola
		docId := cocaColaDocID
		requireArticle(require, "Coca-cola", test.cocaColaNumber, appStructs, docId)
		requireArPrice(require, normalPriceID, 2.0, appStructs, docId, cocaColaNormalPriceElementId)
		{
			requireArPriceException(require, holiday, 1.8, appStructs, docId, cocaColaNormalExceptionHolidayElementId)
		}
		requireArPrice(require, happyHourPriceID, 1.5, appStructs, docId, cocaColaHappyHourPriceElementId)
		{
			requireArPriceException(require, holiday, 0.9, appStructs, docId, cocaColaHappyHourExceptionHolidayElementId)
			requireArPriceException(require, newyear, 0.8, appStructs, docId, cocaColaHappyHourExceptionNewYearElementId)
		}
	}
	{ // fanta
		docId := fantaDocID
		requireArticle(require, "Fanta", test.fantaNumber, appStructs, docId)
		requireArPrice(require, normalPriceID, 2.1, appStructs, docId, fantaNormalPriceElementId)
		{
			requireArPriceException(require, holiday, 1.6, appStructs, docId, fantaNormalExceptionHolidayElementId)
			requireArPriceException(require, newyear, 1.2, appStructs, docId, fantaNormalExceptionNewYearElementId)
		}
		requireArPrice(require, happyHourPriceID, 1.6, appStructs, docId, fantaHappyHourPriceElementId)
		{
			requireArPriceException(require, holiday, 1.1, appStructs, docId, fantaHappyHourExceptionHolidayElementId)
		}
	}
}

func Test_Collection_3levels(t *testing.T) {
	appParts, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	cp_Collection_3levels(t, appParts, appStructs)
}

func TestBasicUsage_QueryFunc_Collection(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs)

	requestBody := []byte(`{
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

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(
		serviceChannel,
		appParts,
		maxPrepareQueries,
		imetrics.Provide(), "vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)
	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(coreutils.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.partition, test.workspace, responder, requestBody, qNameQueryCollection, "", sysToken)
	})

	resultRows := getResultRows(sender, require)

	require.Len(resultRows, 2) // 2 rows

	json, err := json.Marshal(resultRows)
	require.NoError(err)
	require.NotNil(json)

	{
		row := 0
		require.Len(resultRows[row], 2) // 2 elements in a row
		{
			elem := 0
			require.Len(resultRows[row][elem], 1)    // 1 element row in 1st element
			require.Len(resultRows[row][elem][0], 3) // 3 cell in a row element
			name := resultRows[row][elem][0][0]
			number := resultRows[row][elem][0][1]
			department := resultRows[row][elem][0][2]
			require.Equal("Coca-cola", name)
			require.EqualValues(10, number)
			require.Equal("Cold Drinks", department)
		}
		{
			elem := 1
			require.Len(resultRows[row][elem], 2) // 2 element rows in 2nd element
			{
				elemRow := 0
				require.Len(resultRows[row][elem][elemRow], 2) // 2 cells in a row element
				price := resultRows[row][elem][elemRow][0]
				pricename := resultRows[row][elem][elemRow][1]
				require.EqualValues(2.0, price)
				require.Equal("Normal Price", pricename)
			}
			{
				elemRow := 1
				require.Len(resultRows[row][elem][elemRow], 2) // 2 cells in a row element
				price := resultRows[row][elem][elemRow][0]
				pricename := resultRows[row][elem][elemRow][1]
				require.EqualValues(1.5, price)
				require.Equal("Happy Hour Price", pricename)
			}
		}
	}
	{
		row := 1
		require.Len(resultRows[row], 2) // 2 elements in a row
		{
			elem := 0
			require.Len(resultRows[row][elem], 1)    // 1 element row in 1st element
			require.Len(resultRows[row][elem][0], 3) // 3 cell in a row element
			name := resultRows[row][elem][0][0]
			number := resultRows[row][elem][0][1]
			department := resultRows[row][elem][0][2]
			require.Equal("Fanta", name)
			require.EqualValues(12, number)
			require.Equal("Cold Drinks", department)
		}
		{
			elem := 1
			require.Len(resultRows[row][elem], 2) // 2 element rows in 2nd element
			{
				elemRow := 0
				require.Len(resultRows[row][elem][elemRow], 2) // 2 cells in a row element
				price := resultRows[row][elem][elemRow][0]
				pricename := resultRows[row][elem][elemRow][1]
				require.EqualValues(2.1, price)
				require.Equal("Normal Price", pricename)
			}
			{
				elemRow := 1
				require.Len(resultRows[row][elem][elemRow], 2) // 2 cells in a row element
				price := resultRows[row][elem][elemRow][0]
				pricename := resultRows[row][elem][elemRow][1]
				require.EqualValues(1.6, price)
				require.Equal("Happy Hour Price", pricename)
			}
		}
	}
}

func TestBasicUsage_QueryFunc_CDoc(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs)

	requestBody := fmt.Sprintf(`{
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

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, appParts, maxPrepareQueries, imetrics.Provide(),
		"vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(coreutils.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.partition, test.workspace, responder, []byte(requestBody), qNameQueryGetCDoc, "", sysToken)
	})

	resultRows := getResultRows(sender, require)

	require.Len(resultRows, 1)          // 1 row
	require.Len(resultRows[0], 1)       // 1 element in a row
	require.Len(resultRows[0][0], 1)    // 1 row element in an element
	require.Len(resultRows[0][0][0], 1) // 1 cell in a row element

	value := resultRows[0][0][0][0]
	expected := `{

		"article_prices":[
			{
				"article_price_exceptions":[
					{
						"id_periods":3.22685000131076e+14,
						"price":1.8,
						"sys.ID":3.22685000131089e+14,
						"sys.IsActive":true
					}
				],
				"id_prices":3.22685000131072e+14,
				"price":2,
				"sys.ID":3.22685000131079e+14,
				"sys.IsActive":true
			},
			{
				"article_price_exceptions":[
					{
						"id_periods":3.22685000131076e+14,
						"price":0.9,
						"sys.ID":3.22685000131081e+14,
						"sys.IsActive":true
					},
					{
						"id_periods":3.22685000131077e+14,
						"price":0.8,
						"sys.ID":3.22685000131082e+14,
						"sys.IsActive":true
					}
				],
				"id_prices":3.22685000131073e+14,
				"price":1.5,
				"sys.ID":3.2268500013108e+14,
				"sys.IsActive":true
			}
		],
		"id_department":3.22685000131074e+14,
		"name":"Coca-cola",
		"number":10,
		"sys.ID":3.22685000131078e+14,
		"sys.IsActive":true,
		"xrefs":{
			"test.departments":{
				"322685000131074":{
					"name":"Cold Drinks",
					"number":1,
					"sys.ID":3.22685000131074e+14,
					"sys.IsActive":true
				}
			},
			"test.periods":{
				"322685000131076":{
					"name":"Holiday",
					"number":1,
					"sys.ID":322685000131076,
					"sys.IsActive":true
				},
				"322685000131077":{
					"name":"New Year",
					"number":2,
					"sys.ID":322685000131077,
					"sys.IsActive":true
				}
			},
			"test.prices":{
				"322685000131072":{
					"name":"Normal Price",
					"number":1,
					"sys.ID":322685000131072,
					"sys.IsActive":true
				},
				"322685000131073":{
					"name":"Happy Hour Price",
					"number":2,
					"sys.ID":322685000131073,
					"sys.IsActive":true
				}
			}
		}

	}`
	require.JSONEq(expected, value.(string))
}

func TestBasicUsage_State(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs)

	serviceChannel := make(iprocbus.ServiceChannel)

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, appParts, maxPrepareQueries, imetrics.Provide(),
		"vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(coreutils.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.partition, test.workspace, responder, []byte(`{"args":{"After":0},"elements":[{"fields":["State"]}]}`),
			qNameQueryState, "", sysToken)
	})

	resultRows := getResultRows(sender, require)

	require.Len(resultRows, 1)          // 1 row
	require.Len(resultRows[0], 1)       // 1 element in a row
	require.Len(resultRows[0][0], 1)    // 1 row element in an element
	require.Len(resultRows[0][0][0], 1) // 1 cell in a row element
	expected := `{
		"test.article_price_exceptions":{
			"322685000131081":{
				"id_periods":3.22685000131076e+14,
				"price":0.9,
				"sys.ID":322685000131081,
				"sys.IsActive":true,
				"sys.ParentID":3.2268500013108e+14
			},
			"322685000131082":{
				"id_periods": 3.22685000131077e+14,
				"price":0.8,
				"sys.ID":322685000131082,
				"sys.IsActive":true,
				"sys.ParentID":3.2268500013108e+14
			},
			"322685000131085":{
				"id_periods":3.22685000131076e+14,
				"price":1.6,
				"sys.ID":322685000131085,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131084e+14
			},
			"322685000131086":{
				"id_periods":3.22685000131077e+14,
				"price":1.2,
				"sys.ID":322685000131086,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131084e+14
			},
			"322685000131088":{
				"id_periods":3.22685000131076e+14,
				"price":1.1,
				"sys.ID":322685000131088,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131087e+14
			},
			"322685000131089":{
				"id_periods":3.22685000131076e+14,
				"price":1.8,
				"sys.ID":322685000131089,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131079e+14
			}
		},
		"test.article_prices":{
			"322685000131079":{
				"id_prices":3.22685000131072e+14,
				"price":2,
				"sys.ID":322685000131079,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131078e+14
			},
			"322685000131080":{
				"id_prices":3.22685000131073e+14,
				"price":1.5,
				"sys.ID":322685000131080,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131078e+14
			},
			"322685000131084":{
				"id_prices":3.22685000131072e+14,
				"price":2.1,
				"sys.ID":322685000131084,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131083e+14
			},
			"322685000131087":{
				"id_prices":3.22685000131073e+14,
				"price":1.6,
				"sys.ID":322685000131087,
				"sys.IsActive":true,
				"sys.ParentID":3.22685000131083e+14
			}
		},
		"test.articles":{
			"322685000131078":{
				"id_department":3.22685000131074e+14,
				"name":"Coca-cola",
				"number":10,
				"sys.ID":322685000131078,
				"sys.IsActive":true
			},
			"322685000131083":{
				"id_department":3.22685000131074e+14,
				"name":"Fanta",
				"number":12,
				"sys.ID":322685000131083,
				"sys.IsActive":true
			}
		},
		"test.departments":{
			"322685000131074":{
				"name":"Cold Drinks",
				"number":1,
				"sys.ID":322685000131074,
				"sys.IsActive":true
			},
			"322685000131075":{
				"name":"Hot Drinks",
				"number":2,
				"sys.ID":322685000131075,
				"sys.IsActive":true
			}
		},
		"test.periods":{
			"322685000131076":{
				"name":"Holiday",
				"number":1,
				"sys.ID":322685000131076,
				"sys.IsActive":true
			},
			"322685000131077":{
				"name":"New Year",
				"number":2,
				"sys.ID":322685000131077,
				"sys.IsActive":true
			}
		},
		"test.prices":{
			"322685000131072":{
				"name":"Normal Price",
				"number":1,
				"sys.ID":322685000131072,
				"sys.IsActive":true
			},
			"322685000131073":{
				"name":"Happy Hour Price",
				"number":2,
				"sys.ID":322685000131073,
				"sys.IsActive":true
			}
		}
	}`
	require.JSONEq(expected, resultRows[0][0][0][0].(string))
}

func TestState_withAfterArgument(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs)

	serviceChannel := make(iprocbus.ServiceChannel)

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, appParts, maxPrepareQueries, imetrics.Provide(),
		"vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(coreutils.MockTime, bus.GetTestSendTimeout(), func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.partition, test.workspace, responder, []byte(`{"args":{"After":5},"elements":[{"fields":["State"]}]}`),
			qNameQueryState, "", sysToken)
	})

	resultRows := getResultRows(sender, require)

	// out.requireNoError(require)
	require.Len(resultRows, 1)          // 1 row
	require.Len(resultRows[0], 1)       // 1 element in a row
	require.Len(resultRows[0][0], 1)    // 1 row element in an element
	require.Len(resultRows[0][0][0], 1) // 1 cell in a row element
	expected := `
	{
		"test.article_price_exceptions":{
			"322685000131081":{
				"id_periods":3.22685000131076e+14,
				"price":0.9,
				"sys.ID":322685000131081,
				"sys.IsActive":true,
				"sys.ParentID":3.2268500013108e+14
			},
			"322685000131089":{
				"id_periods":3.22685000131076e+14,
				"price":1.8,
				"sys.ID":322685000131089,
				"sys.IsActive":true,
				"sys.ParentID": 3.22685000131079e+14
			}
		}
	}`
	require.JSONEq(expected, resultRows[0][0][0][0].(string))
}

func getResultRows(sender bus.IRequestSender, require *require.Assertions) []resultRow {
	respCh, _, respErr, err := sender.SendRequest(context.Background(), bus.Request{})
	require.NoError(err)
	resultRows := []resultRow{}
	for elem := range respCh {
		_ = elem
		bts, err := json.Marshal(elem)
		require.NoError(err)
		var resultRow resultRow
		require.NoError(json.Unmarshal(bts, &resultRow))
		resultRows = append(resultRows, resultRow)
	}
	require.NoError(*respErr)
	return resultRows
}

func createEvent(require *require.Assertions, app istructs.IAppStructs, generator istructs.IIDGenerator, bld istructs.IRawEventBuilder) istructs.IPLogEvent {
	rawEvent, buildErr := bld.BuildRawEvent()
	var pLogEvent istructs.IPLogEvent
	var err error
	pLogEvent, err = app.Events().PutPlog(rawEvent, buildErr, generator)
	require.NoError(err)
	return pLogEvent
}

func saveEvent(require *require.Assertions, app istructs.IAppStructs, generator istructs.IIDGenerator, bld istructs.IRawEventBuilder) (pLogEvent istructs.IPLogEvent) {
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
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPriceCUD(event, 51, 1, "Normal Price")
		newPriceCUD(event, 52, 2, "Happy Hour Price")
	}))
	return idGen.idmap[51], idGen.idmap[52], event
}

func insertPeriods(require *require.Assertions, app istructs.IAppStructs, idGen *idsGeneratorType) (holiday, newYear istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPeriodCUD(event, 71, 1, "Holiday")
		newPeriodCUD(event, 72, 2, "New Year")
	}))
	return idGen.idmap[71], idGen.idmap[72], event
}

func insertDepartments(require *require.Assertions, app istructs.IAppStructs, idGen *idsGeneratorType) (coldDrinks istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
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

	appParts, appStructs, cleanup, _ := deployTestApp(t)
	defer cleanup()

	// create command processor
	processor := testProcessor(appParts)

	// ID and Offset generators
	idGen := newIdsGenerator()

	coldDrinks, _ := insertDepartments(require, appStructs, &idGen)

	// CUDs: Insert coca-cola
	event1 := createEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
		newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
	}))
	require.NoError(appStructs.Records().Apply(event1))
	cocaColaDocID = idGen.idmap[1]
	require.NoError(processor.SendSync(event1))

	// CUDs: modify coca-cola number and normal price
	event2 := createEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
		updateArticleCUD(event, appStructs, cocaColaDocID, test.cocaColaNumber2, "Coca-cola")
	}))
	require.NoError(appStructs.Records().Apply(event2))
	require.NoError(processor.SendSync(event2))

	// simulate sending event with the same offset
	idGen.decOffset()
	event2copy := createEvent(require, appStructs, &idGen, newModify(appStructs, &idGen, func(event istructs.IRawEventBuilder) {
		updateArticleCUD(event, appStructs, cocaColaDocID, test.cocaColaNumber, "Coca-cola")
	}))
	require.NoError(appStructs.Records().Apply(event2copy))
	require.NoError(processor.SendSync(event2copy))

	// Check expected projection values
	{ // coca-cola
		requireArticle(require, "Coca-cola", test.cocaColaNumber2, appStructs, cocaColaDocID)
	}

}
