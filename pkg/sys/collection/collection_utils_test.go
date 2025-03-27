/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
)

type testDataType struct {
	appQName        appdef.AppQName
	totalPartitions istructs.NumAppPartitions
	appEngines      [appparts.ProcessorKind_Count]uint

	pkgName string

	// common event entites
	partitionIdent string
	partition      istructs.PartitionID
	workspace      istructs.WSID
	plogStartOfs   istructs.Offset

	// function
	modifyCmdName       appdef.QName
	modifyCmdParamsName appdef.QName
	modifyCmdResultName appdef.QName

	// records
	tableArticles      appdef.QName
	articleNameIdent   string
	articleNumberIdent string
	articleDeptIdent   string

	tableArticlePrices        appdef.QName
	articlePricesPriceIdent   string
	articlePricesPriceIdIdent string

	tableArticlePriceExceptions         appdef.QName
	articlePriceExceptionsPeriodIdIdent string
	articlePriceExceptionsPriceIdent    string

	tableDepartments appdef.QName
	depNameIdent     string
	depNumberIdent   string

	tablePrices      appdef.QName
	priceNameIdent   string
	priceNumberIdent string

	tablePeriods      appdef.QName
	periodNameIdent   string
	periodNumberIdent string

	// backoffice
	cocaColaNumber  int32
	cocaColaNumber2 int32
	fantaNumber     int32
}

const OccursUnbounded = appdef.Occurs(0xffff)

var test = testDataType{
	appQName:        istructs.AppQName_test1_app1,
	totalPartitions: 100,
	appEngines:      appparts.PoolSize(100, 100, 0, 0),

	pkgName: "test",

	partitionIdent:      "Partition",
	partition:           55,
	workspace:           1234,
	plogStartOfs:        1,
	modifyCmdName:       appdef.NewQName("test", "modify"),
	modifyCmdParamsName: appdef.NewQName("test", "modifyArgs"),
	modifyCmdResultName: appdef.NewQName("test", "modifyResult"),

	/////
	tableArticles:      appdef.NewQName("test", "articles"),
	articleNameIdent:   "name",
	articleNumberIdent: "number",
	articleDeptIdent:   "id_department",

	tableArticlePrices:        appdef.NewQName("test", "article_prices"),
	articlePricesPriceIdIdent: "id_prices",
	articlePricesPriceIdent:   "price",

	tableArticlePriceExceptions:         appdef.NewQName("test", "article_price_exceptions"),
	articlePriceExceptionsPeriodIdIdent: "id_periods",
	articlePriceExceptionsPriceIdent:    "price",

	tableDepartments: appdef.NewQName("test", "departments"),
	depNameIdent:     "name",
	depNumberIdent:   "number",

	tablePrices:      appdef.NewQName("test", "prices"),
	priceNameIdent:   "name",
	priceNumberIdent: "number",

	tablePeriods:      appdef.NewQName("test", "periods"),
	periodNameIdent:   "name",
	periodNumberIdent: "number",

	// backoffice
	cocaColaNumber:  10,
	cocaColaNumber2: 11,
	fantaNumber:     12,
}

type testCmdWorkpeace struct {
	appPart appparts.IAppPartition
	event   istructs.IPLogEvent
}

func (w testCmdWorkpeace) AppPartition() appparts.IAppPartition { return w.appPart }
func (w testCmdWorkpeace) Event() istructs.IPLogEvent           { return w.event }

func (w *testCmdWorkpeace) Borrow(ctx context.Context, appParts appparts.IAppPartitions) (err error) {
	w.appPart, err = appParts.WaitForBorrow(ctx, test.appQName, test.partition, appparts.ProcessorKind_Command)
	return err
}

func (w *testCmdWorkpeace) Command(e any) error {
	w.event = e.(istructs.IPLogEvent)
	return nil
}

func (w *testCmdWorkpeace) Actualizers(ctx context.Context) error {
	return w.appPart.DoSyncActualizer(ctx, w)
}

func (w *testCmdWorkpeace) Release() {
	p := w.appPart
	w.appPart = nil
	if p != nil {
		p.Release()
	}
}

type testCmdProc struct {
	pipeline.ISyncPipeline
	appParts  appparts.IAppPartitions
	ctx       context.Context
	workpeace testCmdWorkpeace
}

func testProcessor(appParts appparts.IAppPartitions) *testCmdProc {
	proc := &testCmdProc{
		appParts:  appParts,
		ctx:       context.Background(),
		workpeace: testCmdWorkpeace{},
	}
	proc.ISyncPipeline = pipeline.NewSyncPipeline(proc.ctx, "partition processor",
		pipeline.WireSyncOperator("Borrow", pipeline.NewSyncOp(
			func(ctx context.Context, _ pipeline.IWorkpiece) error {
				return proc.workpeace.Borrow(ctx, appParts)
			})),
		pipeline.WireSyncOperator("Command", pipeline.NewSyncOp(
			func(_ context.Context, event pipeline.IWorkpiece) error {
				return proc.workpeace.Command(event)
			})),
		pipeline.WireSyncOperator("SyncActualizers", pipeline.NewSyncOp(
			func(ctx context.Context, _ pipeline.IWorkpiece) error {
				return proc.workpeace.Actualizers(ctx)
			})),
		pipeline.WireSyncOperator("Release", pipeline.NewSyncOp(
			func(context.Context, pipeline.IWorkpiece) error {
				proc.workpeace.Release()
				return nil
			})))
	return proc
}

type idsGeneratorType struct {
	istructs.IIDGenerator
	idmap          map[istructs.RecordID]istructs.RecordID
	nextPlogOffset istructs.Offset
}

func (me *idsGeneratorType) NextID(rawID istructs.RecordID, t appdef.IType) (storageID istructs.RecordID, err error) {
	if storageID, err = me.IIDGenerator.NextID(rawID, t); err != nil {
		return istructs.NullRecordID, err
	}
	me.idmap[rawID] = storageID
	return
}

func (me *idsGeneratorType) nextOffset() (offset istructs.Offset) {
	offset = me.nextPlogOffset
	me.nextPlogOffset++
	return
}

func (me *idsGeneratorType) decOffset() {
	me.nextPlogOffset--
}

func newIdsGenerator() idsGeneratorType {
	return idsGeneratorType{
		idmap:          make(map[istructs.RecordID]istructs.RecordID),
		nextPlogOffset: test.plogStartOfs,
		IIDGenerator:   istructsmem.NewIDGenerator(),
	}
}

func requireArticle(require *require.Assertions, name string, number int32, as istructs.IAppStructs, articleId istructs.RecordID) {
	kb := as.ViewRecords().KeyBuilder(QNameCollectionView)
	kb.PutInt32(Field_PartKey, PartitionKeyCollection)
	kb.PutQName(Field_DocQName, test.tableArticles)
	kb.PutRecordID(Field_DocID, articleId)
	kb.PutRecordID(field_ElementID, istructs.NullRecordID)
	value, err := as.ViewRecords().Get(test.workspace, kb)
	require.NoError(err)
	recArticle := value.AsRecord(Field_Record)
	require.Equal(name, recArticle.AsString(test.articleNameIdent))
	require.Equal(number, recArticle.AsInt32(test.articleNumberIdent))
}

func requireArPrice(require *require.Assertions, priceId istructs.RecordID, price float32, as istructs.IAppStructs, articleId, articlePriceId istructs.RecordID) {
	kb := as.ViewRecords().KeyBuilder(QNameCollectionView)
	kb.PutInt32(Field_PartKey, PartitionKeyCollection)
	kb.PutQName(Field_DocQName, test.tableArticles)
	kb.PutRecordID(Field_DocID, articleId)
	kb.PutRecordID(field_ElementID, articlePriceId)
	value, err := as.ViewRecords().Get(test.workspace, kb)
	require.NoError(err)
	recArticlePrice := value.AsRecord(Field_Record)
	require.Equal(priceId, recArticlePrice.AsRecordID(test.articlePricesPriceIdIdent))
	require.Equal(price, recArticlePrice.AsFloat32(test.articlePricesPriceIdent))
}

func requireArPriceException(require *require.Assertions, periodId istructs.RecordID, price float32, as istructs.IAppStructs, articleId, articlePriceExceptionId istructs.RecordID) {
	kb := as.ViewRecords().KeyBuilder(QNameCollectionView)
	kb.PutInt32(Field_PartKey, PartitionKeyCollection)
	kb.PutQName(Field_DocQName, test.tableArticles)
	kb.PutRecordID(Field_DocID, articleId)
	kb.PutRecordID(field_ElementID, articlePriceExceptionId)
	value, err := as.ViewRecords().Get(test.workspace, kb)
	require.NoError(err)
	recArticlePriceException := value.AsRecord(Field_Record)
	require.Equal(periodId, recArticlePriceException.AsRecordID(test.articlePriceExceptionsPeriodIdIdent))
	require.Equal(price, recArticlePriceException.AsFloat32(test.articlePriceExceptionsPriceIdent))
}

type resultElementRow []interface{}

type resultElement []resultElementRow

type resultRow []resultElement
