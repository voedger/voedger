# airs-bp example

Prototype of extensions to build airs-bp application

## Context

- [vpm schema](https://github.com/voedger/voedger/issues/1476)
- https://github.com/voedger/voedger/tree/main/staging/src/github.com/voedger/exttinygo
- https://github.com/voedger/voedger/blob/main/pkg/iextenginewazero/_testdata/basicusage/main.go

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

### github.com/untillpro/airs-bp3/apps/untill/airsbp/schemas.sql

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

### github.com/untillpro/airs-bp3/packages/air/air.sql

```sql
-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

IMPORT SCHEMA 'github.com/untillpro/airs-scheme/bp3' AS untill;

WORKSPACE RestaurantWS (

	TABLE ProformaPrinted INHERITS ODoc (
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
```

### github.com/untillpro/airs-scheme/bp3

```sql
TABLE articles INHERITS CDoc (
	article_number int32,
	name varchar(255),
    ...
)

TABLE untill_users INHERITS CDoc (
	name varchar(255),
	mandates bytes(10),
	user_void int32 NOT NULL,
	user_code varchar(50),
	user_card varchar(50),
	language varchar(100),
    ...
)
```