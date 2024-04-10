-- Copyright (c) 2020-present unTill Pro, Ltd.

SCHEMA air;

IMPORT SCHEMA "github.com/untillpro/airs-bp3/packages/untill"

WORKSPACE Restaurant (

    -- dashboard: hourly sales
    VIEW HourlySalesView(
        yyyymmdd,
        hour,
        total,
        count
    ) AS SELECT
        working_day,
        EXTRACT(hour from ord_datetime),
        (select sum(price * quantity) from order_item),
        (select sum(quantity) from order_item),
        from untill.orders
    WITH Comment=PosComment, PrimaryKey='(yyyymmdd, hour), asdas';

    -- dashboard: daily categories
    VIEW DailyCategoriesView(yyyymmdd PK, id_category, total) A
    SELECT
        working_day as yyyymmdd,
        id_category,
        SUM(price * quantity) as total,
        from untill.orders
            join order_item on order_item.id_orders = orders.id
            join articles on id_articles = articles.id
            join department on id_departments = articles.id_department
            join food_group on id_food_group = department.id_food_group
        group by working_day, id_category


    VIEW TablesOverview(
        partitionKey int32, tableNumber int32, tablePart text, wDocID id,
        PRIMARY KEY((partitionKey), tableno, table_part)
    ) as select
        2 as partitionKey,
        tableno as tableNumber,
        table_part as tablePart,
        sys.ID as id
    from untill.bill

    VIEW TransactionHistory(wDocID id, offs offset, PRIMARY KEY((id), offs)) AS
    select id, sys.Offset from untill.bill
    union all select id_bill, sys.Offset from orders
    union all select id_bill, sys.Offset from pbill ;

    VIEW XZReports(
        Year int32,
        Month int32,
        Day int32,
        Kind int32,
        Number int32,
        XZReportWDocID id,
        PRIMARY KEY((Year), Month, Day, Kind, Number)
    ) AS RESULT OF UpdateXZReportsView



) -- WORKSPACE Restaurant
