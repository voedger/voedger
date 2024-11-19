# airs-bp example

Prototype of extensions to build airs-bp application

## Context

- [vpm schema](https://github.com/voedger/voedger/issues/1476)
- https://github.com/voedger/voedger/tree/main/staging/src/github.com/voedger/exttinygo
- https://github.com/voedger/voedger/blob/main/pkg/iextenginewazero/_testdata/basicusage/main.go
- https://github.com/untillpro/airs-bp3/blob/2d0d38d1b73f85165a520659cf1e5cad4e67a950/packages/air/pbilldates/impl_fillpbilldates.go#L25

## Problems

- QNames: `IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill`
- Names conflict: Air, Air_ProformaPrinted
- Query by partial key
- missing function body
- Naming: `articles`, `articles.article_number`
  - `schemas.Untill.Articles.MustGetValue(schemas.ID(12))`
  - `println(v.Article_number())`
  - Solution: semantic analysis shall verify qnames/names uniqueness using case-insensitive mode
```go
//export hostPanic
func hostPanic(msgPtr, msgSize uint32)

//export hostRowWriterPutBytes
func hostRowWriterPutBytes(id uint64, typ uint32, namePtr, nameSize, valuePtr, valueSize uint32)
```

## Schemas

### airsbp

https://github.com/untillpro/airs-bp3/blob/main/apps/untill/airsbp/schemas.vsql

```sql
-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill;
IMPORT SCHEMA 'github.com/untillpro/airs-bp3/packages/air';

APPLICATION airsbp (
	USE untill;
	USE air;
);
```

### air

https://github.com/untillpro/airs-bp3/blob/main/packages/air/air.vsql

```sql
-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill;

WORKSPACE RestaurantWS (

	TABLE ProformaPrinted INHERITS sys.ODoc (
		Number int32 NOT NULL,
		UserID ref(untill.untill_users) NOT NULL,
		Timestamp int64 NOT NULL,
		BillID ref(untill.bill) NOT NULL
	);

	VIEW PbillDates (
		Year int32 NOT NULL,
		DayOfYear int32 NOT NULL,
		FirstOffset int64 NOT NULL,
		LastOffset int64 NOT NULL,
		PRIMARY KEY ((Year), DayOfYear)
	) AS RESULT OF FillPbillDates;

	COMMAND Orders(untill.orders);

	COMMAND Pbill(untill.pbill) RETURNS CmdPBillResult;

	TYPE CmdPBillResult (
		Number int32 NOT NULL
	);

	TABLE NextNumbers INHERITS sys.CSingleton (
		NextPBillNumber int32
	);

```

### untill

https://github.com/untillpro/airs-scheme/blob/master/bp3/schema.vsql

```sql

TABLE bill INHERITS sys.WDoc (
	close_datetime int64,
	table_name varchar(50),
	tableno int32 NOT NULL,
	id_untill_users ref(untill_users) NOT NULL,
	table_part varchar(1) NOT NULL,
)

TABLE articles INHERITS sys.CDoc (
	article_number int32,
	name varchar(255),
    ...
)

TABLE untill_users INHERITS sys.CDoc (
	name varchar(255),
	mandates bytes(10),
	user_void int32 NOT NULL,
	user_code varchar(50),
	user_card varchar(50),
	language varchar(100),
    ...
)


TABLE pbill INHERITS sys.ODoc (
	id_bill int64 NOT NULL,
	id_untill_users ref(untill_users) NOT NULL,
	number int32,
	pbill_item pbill_item
}


TABLE pbill_item INHERITS sys.ORecord (
	id_untill_users ref(untill_users) NOT NULL,
	tableno int32 NOT NULL,
	rowbeg int32 NOT NULL
)

```