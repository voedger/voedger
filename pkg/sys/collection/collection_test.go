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
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/isequencer"
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

const (
	maxPrepareQueries = 10
	sendTimeout       = bus.SendTimeout(10 * time.Second)
)

func deployTestApp(t *testing.T) (appParts appparts.IAppPartitions, appStructs istructs.IAppStructs, cleanup func(),
	statelessResources istructsmem.IStatelessResources, idGen *TSidsGeneratorType) {
	require := require.New(t)

	cfgs := make(istructsmem.AppConfigsType, 1)
	asp := istorageimpl.Provide(mem.Provide(testingu.MockTime))

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
			AddField(test.articlePricesPriceIDIdent, appdef.DataKind_RecordID, true).
			AddField(test.articlePricesPriceIdent, appdef.DataKind_float32, true)
		articlesPrices.
			AddContainer(test.tableArticlePriceExceptions.Entity(), test.tableArticlePriceExceptions, appdef.Occurs(0), appdef.Occurs(100))
	}

	{ // CDoc: article price exceptions
		articlesPricesExceptions := wsb.AddCRecord(test.tableArticlePriceExceptions)
		articlesPricesExceptions.
			AddField(test.articlePriceExceptionsPeriodIDIdent, appdef.DataKind_RecordID, true).
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
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), asp, isequencer.SequencesTrustLevel_0, nil)

	secretReader := isecretsimpl.ProvideSecretReader()
	n10nBroker, n10nBrokerCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                1000,
		ChannelsPerSubject:      10,
		Subscriptions:           1000,
		SubscriptionsPerSubject: 10,
	}, timeu.NewITime())

	vvmCtx, cancel := context.WithCancel(context.Background())
	appParts, appPartsCleanup, err := appparts.New2(vvmCtx, appStructsProvider,
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
	idGen = newTSIdsGenerator()
	nextOffset := idGen.nextOffset()
	err = wsdescutil.CreateCDocWorkspaceDescriptorStub(as, test.partition, test.workspace, wsdescutil.TestWsDescName, nextOffset, nextOffset)
	require.NoError(err)

	cleanup = func() {
		cancel()
		appPartsCleanup()
		n10nBrokerCleanup()
	}

	return appParts, as, cleanup, statelessResources, idGen
}

// Test executes 3 operations with CUDs:
// - insert coca-cola with 2 prices
// - modify coca-cola and 1 price
// - insert fanta with 2 prices
// ...then launches Collection actualizer and waits until it reads all the log.
// Then projection values checked.
func TestBasicUsage_Collection(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, _, idGen := deployTestApp(t)
	defer cleanup()

	// Command processor
	processor := testProcessor(appParts)

	normalPriceID, happyHourPriceID, _ := insertPrices(require, appStructs, idGen)
	coldDrinks, _ := insertDepartments(require, appStructs, idGen)

	{ // CUDs: Insert coca-cola
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.4)
			newArPriceCUD(event, 1, 3, happyHourPriceID, 1.8)
		}))
		err := processor.SendSync(event)
		require.NoError(err)
	}

	cocaColaDocID = idGen.idmap[1]
	cocaColaNormalPriceElementID := idGen.idmap[2]
	cocaColaHappyHourPriceElementID := idGen.idmap[3]

	{ // CUDs: modify coca-cola number and normal price
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			updateArticleCUD(event, appStructs, cocaColaDocID, test.cocaColaNumber2, "Coca-cola")
			updateArPriceCUD(event, appStructs, cocaColaNormalPriceElementID, normalPriceID, 2.2)
		}))
		require.NoError(processor.SendSync(event))
	}

	{ // CUDs: insert fanta
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 7, coldDrinks, test.fantaNumber, "Fanta")
			newArPriceCUD(event, 7, 8, normalPriceID, 2.1)
			newArPriceCUD(event, 7, 9, happyHourPriceID, 1.7)
		}))
		require.NoError(processor.SendSync(event))
	}
	fantaDocID := idGen.idmap[7]
	fantaNormalPriceElementID := idGen.idmap[8]
	fantaHappyHourPriceElementID := idGen.idmap[9]

	// Check expected projection values
	{ // coca-cola
		requireArticle(require, "Coca-cola", test.cocaColaNumber2, appStructs, cocaColaDocID)
		requireArPrice(require, normalPriceID, 2.2, appStructs, cocaColaDocID, cocaColaNormalPriceElementID)
		requireArPrice(require, happyHourPriceID, 1.8, appStructs, cocaColaDocID, cocaColaHappyHourPriceElementID)
	}
	{ // fanta
		requireArticle(require, "Fanta", test.fantaNumber, appStructs, fantaDocID)
		requireArPrice(require, normalPriceID, 2.1, appStructs, fantaDocID, fantaNormalPriceElementID)
		requireArPrice(require, happyHourPriceID, 1.7, appStructs, fantaDocID, fantaHappyHourPriceElementID)
	}

}

func Test_updateChildRecord(t *testing.T) {
	require := require.New(t)

	_, appStructs, cleanup, _, idGen := deployTestApp(t)
	defer cleanup()

	normalPriceID, _, _ := insertPrices(require, appStructs, idGen)
	coldDrinks, _ := insertDepartments(require, appStructs, idGen)

	{ // CUDs: Insert coca-cola
		saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			newArticleCUD(event, 1, coldDrinks, test.cocaColaNumber, "Coca-cola")
			newArPriceCUD(event, 1, 2, normalPriceID, 2.4)
		}))
	}

	cocaColaNormalPriceElementID := idGen.idmap[2]

	{ // CUDs: modify normal price
		saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			updateArPriceCUD(event, appStructs, cocaColaNormalPriceElementID, normalPriceID, 2.2)
		}))
	}

	rec, err := appStructs.Records().Get(test.workspace, true, cocaColaNormalPriceElementID)
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

func cp_Collection_3levels(t *testing.T, appParts appparts.IAppPartitions, appStructs istructs.IAppStructs, idGen *TSidsGeneratorType) {
	require := require.New(t)

	// Command processor
	processor := testProcessor(appParts)

	normalPriceID, happyHourPriceID, eventPrices := insertPrices(require, appStructs, idGen)
	coldDrinks, eventDepartments := insertDepartments(require, appStructs, idGen)
	holiday, newyear, eventPeriods := insertPeriods(require, appStructs, idGen)

	for _, event := range []istructs.IPLogEvent{eventPrices, eventDepartments, eventPeriods} {
		require.NoError(processor.SendSync(event))
	}

	// insert coca-cola
	{
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
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
	cocaColaNormalPriceElementID := idGen.idmap[2]
	cocaColaHappyHourPriceElementID := idGen.idmap[3]
	cocaColaHappyHourExceptionHolidayElementID := idGen.idmap[4]
	cocaColaHappyHourExceptionNewYearElementID := idGen.idmap[5]

	// insert fanta
	{
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
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
	fantaNormalPriceElementID := idGen.idmap[7]
	fantaNormalExceptionHolidayElementID := idGen.idmap[9]
	fantaNormalExceptionNewYearElementID := idGen.idmap[10]
	fantaHappyHourPriceElementID := idGen.idmap[8]
	fantaHappyHourExceptionHolidayElementID := idGen.idmap[11]

	// modify coca-cola
	{
		event := saveEvent(require, appStructs, idGen, newModify(appStructs, idGen, func(event istructs.IRawEventBuilder) {
			newArPriceExceptionCUD(event, cocaColaNormalPriceElementID, 15, holiday, 1.8)
			updateArPriceExceptionCUD(event, appStructs, cocaColaHappyHourExceptionHolidayElementID, holiday, 0.9)
		}))
		require.NoError(processor.SendSync(event))
	}
	cocaColaNormalExceptionHolidayElementID := idGen.idmap[15]
	require.NotEqual(istructs.NullRecordID, cocaColaNormalExceptionHolidayElementID)

	// Check expected projection values
	{ // coca-cola
		docID := cocaColaDocID
		requireArticle(require, "Coca-cola", test.cocaColaNumber, appStructs, docID)
		requireArPrice(require, normalPriceID, 2.0, appStructs, docID, cocaColaNormalPriceElementID)
		{
			requireArPriceException(require, holiday, 1.8, appStructs, docID, cocaColaNormalExceptionHolidayElementID)
		}
		requireArPrice(require, happyHourPriceID, 1.5, appStructs, docID, cocaColaHappyHourPriceElementID)
		{
			requireArPriceException(require, holiday, 0.9, appStructs, docID, cocaColaHappyHourExceptionHolidayElementID)
			requireArPriceException(require, newyear, 0.8, appStructs, docID, cocaColaHappyHourExceptionNewYearElementID)
		}
	}
	{ // fanta
		docID := fantaDocID
		requireArticle(require, "Fanta", test.fantaNumber, appStructs, docID)
		requireArPrice(require, normalPriceID, 2.1, appStructs, docID, fantaNormalPriceElementID)
		{
			requireArPriceException(require, holiday, 1.6, appStructs, docID, fantaNormalExceptionHolidayElementID)
			requireArPriceException(require, newyear, 1.2, appStructs, docID, fantaNormalExceptionNewYearElementID)
		}
		requireArPrice(require, happyHourPriceID, 1.6, appStructs, docID, fantaHappyHourPriceElementID)
		{
			requireArPriceException(require, holiday, 1.1, appStructs, docID, fantaHappyHourExceptionHolidayElementID)
		}
	}
}

func Test_Collection_3levels(t *testing.T) {
	appParts, appStructs, cleanup, _, idGen := deployTestApp(t)
	defer cleanup()

	cp_Collection_3levels(t, appParts, appStructs, idGen)
}

func TestBasicUsage_QueryFunc_Collection(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources, idGen := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs, idGen)

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
	sender := bus.NewIRequestSender(testingu.MockTime, sendTimeout, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
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

	appParts, appStructs, cleanup, statelessResources, idGen := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs, idGen)

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
	sender := bus.NewIRequestSender(testingu.MockTime, sendTimeout, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
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
						"id_periods":200005,
						"price":1.8,
						"sys.ID":200018,
						"sys.IsActive":true
					}
				],
				"id_prices":200001,
				"price":2,
				"sys.ID":200008,
				"sys.IsActive":true
			},
			{
				"article_price_exceptions":[
					{
						"id_periods":200005,
						"price":0.9,
						"sys.ID":200010,
						"sys.IsActive":true
					},
					{
						"id_periods":200006,
						"price":0.8,
						"sys.ID":200011,
						"sys.IsActive":true
					}
				],
				"id_prices":200002,
				"price":1.5,
				"sys.ID":200009,
				"sys.IsActive":true
			}
		],
		"id_department":200003,
		"name":"Coca-cola",
		"number":10,
		"sys.ID":200007,
		"sys.IsActive":true,
		"xrefs":{
			"test.departments":{
				"200003":{
					"name":"Cold Drinks",
					"number":1,
					"sys.ID":200003,
					"sys.IsActive":true
				}
			},
			"test.periods":{
				"200005":{
					"name":"Holiday",
					"number":1,
					"sys.ID":200005,
					"sys.IsActive":true
				},
				"200006":{
					"name":"New Year",
					"number":2,
					"sys.ID":200006,
					"sys.IsActive":true
				}
			},
			"test.prices":{
				"200001":{
					"name":"Normal Price",
					"number":1,
					"sys.ID":200001,
					"sys.IsActive":true
				},
				"200002":{
					"name":"Happy Hour Price",
					"number":2,
					"sys.ID":200002,
					"sys.IsActive":true
				}
			}
		}

	}`
	require.JSONEq(expected, value.(string))
}

func TestBasicUsage_State(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources, idGen := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs, idGen)

	serviceChannel := make(iprocbus.ServiceChannel)

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, appParts, maxPrepareQueries, imetrics.Provide(),
		"vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(testingu.MockTime, sendTimeout, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
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
			"200010":{
				"id_periods":200005,
				"price":0.9,
				"sys.ID":200010,
				"sys.IsActive":true,
				"sys.ParentID":200009
			},
			"200011":{
				"id_periods": 200006,
				"price":0.8,
				"sys.ID":200011,
				"sys.IsActive":true,
				"sys.ParentID":200009
			},
			"200014":{
				"id_periods":200005,
				"price":1.6,
				"sys.ID":200014,
				"sys.IsActive":true,
				"sys.ParentID":200013
			},
			"200015":{
				"id_periods":200006,
				"price":1.2,
				"sys.ID":200015,
				"sys.IsActive":true,
				"sys.ParentID":200013
			},
			"200017":{
				"id_periods":200005,
				"price":1.1,
				"sys.ID":200017,
				"sys.IsActive":true,
				"sys.ParentID":200016
			},
			"200018":{
				"id_periods":200005,
				"price":1.8,
				"sys.ID":200018,
				"sys.IsActive":true,
				"sys.ParentID":200008
			}
		},
		"test.article_prices":{
			"200008":{
				"id_prices":200001,
				"price":2,
				"sys.ID":200008,
				"sys.IsActive":true,
				"sys.ParentID":200007
			},
			"200009":{
				"id_prices":200002,
				"price":1.5,
				"sys.ID":200009,
				"sys.IsActive":true,
				"sys.ParentID":200007
			},
			"200013":{
				"id_prices":200001,
				"price":2.1,
				"sys.ID":200013,
				"sys.IsActive":true,
				"sys.ParentID":200012
			},
			"200016":{
				"id_prices":200002,
				"price":1.6,
				"sys.ID":200016,
				"sys.IsActive":true,
				"sys.ParentID":200012
			}
		},
		"test.articles":{
			"200007":{
				"id_department":200003,
				"name":"Coca-cola",
				"number":10,
				"sys.ID":200007,
				"sys.IsActive":true
			},
			"200012":{
				"id_department":200003,
				"name":"Fanta",
				"number":12,
				"sys.ID":200012,
				"sys.IsActive":true
			}
		},
		"test.departments":{
			"200003":{
				"name":"Cold Drinks",
				"number":1,
				"sys.ID":200003,
				"sys.IsActive":true
			},
			"200004":{
				"name":"Hot Drinks",
				"number":2,
				"sys.ID":200004,
				"sys.IsActive":true
			}
		},
		"test.periods":{
			"200005":{
				"name":"Holiday",
				"number":1,
				"sys.ID":200005,
				"sys.IsActive":true
			},
			"200006":{
				"name":"New Year",
				"number":2,
				"sys.ID":200006,
				"sys.IsActive":true
			}
		},
		"test.prices":{
			"200001":{
				"name":"Normal Price",
				"number":1,
				"sys.ID":200001,
				"sys.IsActive":true
			},
			"200002":{
				"name":"Happy Hour Price",
				"number":2,
				"sys.ID":200002,
				"sys.IsActive":true
			}
		}
	}`
	require.JSONEq(expected, resultRows[0][0][0][0].(string))
}

func TestState_withAfterArgument(t *testing.T) {
	require := require.New(t)

	appParts, appStructs, cleanup, statelessResources, idGen := deployTestApp(t)
	defer cleanup()

	// Fill the collection projection
	cp_Collection_3levels(t, appParts, appStructs, idGen)

	serviceChannel := make(iprocbus.ServiceChannel)

	authn := iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs)
	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(test.appQName)
	queryProcessor := queryprocessor.ProvideServiceFactory()(serviceChannel, appParts, maxPrepareQueries, imetrics.Provide(),
		"vvm", authn, tokens, nil, statelessResources, isecretsimpl.TestSecretReader)

	go queryProcessor.Run(context.Background())
	sysToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	sender := bus.NewIRequestSender(testingu.MockTime, sendTimeout, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		serviceChannel <- queryprocessor.NewQueryMessage(context.Background(), test.appQName, test.partition, test.workspace, responder, []byte(`{"args":{"After":6},"elements":[{"fields":["State"]}]}`),
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
			"200010":{
				"id_periods":200005,
				"price":0.9,
				"sys.ID":200010,
				"sys.IsActive":true,
				"sys.ParentID":200009
			},
			"200018":{
				"id_periods":200005,
				"price":1.8,
				"sys.ID":200018,
				"sys.IsActive":true,
				"sys.ParentID": 200008
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
	require.Empty(pLogEvent.Error().ErrStr())
	return
}

func newPriceCUD(bld istructs.IRawEventBuilder, recordID istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "prices"))
	rec.PutRecordID(appdef.SystemField_ID, recordID)
	rec.PutString(test.priceNameIdent, name)
	rec.PutInt32(test.priceNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newPeriodCUD(bld istructs.IRawEventBuilder, recordID istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "periods"))
	rec.PutRecordID(appdef.SystemField_ID, recordID)
	rec.PutString(test.periodNameIdent, name)
	rec.PutInt32(test.periodNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newDepartmentCUD(bld istructs.IRawEventBuilder, recordID istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "departments"))
	rec.PutRecordID(appdef.SystemField_ID, recordID)
	rec.PutString(test.depNameIdent, name)
	rec.PutInt32(test.depNumberIdent, number)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func newArticleCUD(bld istructs.IRawEventBuilder, articleRecordID, department istructs.RecordID, number int32, name string) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "articles"))
	rec.PutRecordID(appdef.SystemField_ID, articleRecordID)
	rec.PutString(test.articleNameIdent, name)
	rec.PutInt32(test.articleNumberIdent, number)
	rec.PutRecordID(test.articleDeptIdent, department)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArticleCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articleRecordID istructs.RecordID, number int32, name string) {
	rec, err := app.Records().Get(test.workspace, false, articleRecordID)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article %d not found", articleRecordID))
	}
	writer := bld.CUDBuilder().Update(rec)
	writer.PutString(test.articleNameIdent, name)
	writer.PutInt32(test.articleNumberIdent, number)
}

func newArPriceCUD(bld istructs.IRawEventBuilder, articleRecordID, articlePriceRecordID istructs.RecordID, idPrice istructs.RecordID, price float32) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "article_prices"))
	rec.PutRecordID(appdef.SystemField_ID, articlePriceRecordID)
	rec.PutRecordID(appdef.SystemField_ParentID, articleRecordID)
	rec.PutString(appdef.SystemField_Container, "article_prices")
	rec.PutRecordID(test.articlePricesPriceIDIdent, idPrice)
	rec.PutFloat32(test.articlePricesPriceIdent, price)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArPriceCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articlePriceRecordID istructs.RecordID, idPrice istructs.RecordID, price float32) {
	rec, err := app.Records().Get(test.workspace, true, articlePriceRecordID)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article price %d not found", articlePriceRecordID))
	}
	writer := bld.CUDBuilder().Update(rec)
	writer.PutRecordID(test.articlePricesPriceIDIdent, idPrice)
	writer.PutFloat32(test.articlePricesPriceIdent, price)
}

func newArPriceExceptionCUD(bld istructs.IRawEventBuilder, articlePriceRecordID, articlePriceExceptionRecordID, period istructs.RecordID, price float32) {
	rec := bld.CUDBuilder().Create(appdef.NewQName("test", "article_price_exceptions"))
	rec.PutRecordID(appdef.SystemField_ID, articlePriceExceptionRecordID)
	rec.PutRecordID(appdef.SystemField_ParentID, articlePriceRecordID)
	rec.PutString(appdef.SystemField_Container, "article_price_exceptions")
	rec.PutRecordID(test.articlePriceExceptionsPeriodIDIdent, period)
	rec.PutFloat32(test.articlePriceExceptionsPriceIdent, price)
	rec.PutBool(appdef.SystemField_IsActive, true)
}

func updateArPriceExceptionCUD(bld istructs.IRawEventBuilder, app istructs.IAppStructs, articlePriceExceptionRecordID, idPeriod istructs.RecordID, price float32) {
	rec, err := app.Records().Get(test.workspace, true, articlePriceExceptionRecordID)
	if err != nil {
		panic(err)
	}
	if rec.QName() == appdef.NullQName {
		panic(fmt.Sprintf("Article price exception %d not found", articlePriceExceptionRecordID))
	}

	writer := bld.CUDBuilder().Update(rec)
	writer.PutRecordID(test.articlePriceExceptionsPeriodIDIdent, idPeriod)
	writer.PutFloat32(test.articlePriceExceptionsPriceIdent, price)
}
func insertPrices(require *require.Assertions, app istructs.IAppStructs, idGen *TSidsGeneratorType) (normalPrice, happyHourPrice istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPriceCUD(event, 51, 1, "Normal Price")
		newPriceCUD(event, 52, 2, "Happy Hour Price")
	}))
	return idGen.idmap[51], idGen.idmap[52], event
}

func insertPeriods(require *require.Assertions, app istructs.IAppStructs, idGen *TSidsGeneratorType) (holiday, newYear istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newPeriodCUD(event, 71, 1, "Holiday")
		newPeriodCUD(event, 72, 2, "New Year")
	}))
	return idGen.idmap[71], idGen.idmap[72], event
}

func insertDepartments(require *require.Assertions, app istructs.IAppStructs, idGen *TSidsGeneratorType) (coldDrinks istructs.RecordID, event istructs.IPLogEvent) {
	event = saveEvent(require, app, idGen, newModify(app, idGen, func(event istructs.IRawEventBuilder) {
		newDepartmentCUD(event, 61, 1, "Cold Drinks")
		newDepartmentCUD(event, 62, 2, "Hot Drinks")
	}))
	coldDrinks = idGen.idmap[61]
	return
}

type eventCallback func(event istructs.IRawEventBuilder)

func newModify(app istructs.IAppStructs, gen *TSidsGeneratorType, cb eventCallback) istructs.IRawEventBuilder {
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
