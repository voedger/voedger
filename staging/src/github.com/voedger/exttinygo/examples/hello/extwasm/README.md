## Context

- [vpm schema](https://github.com/voedger/voedger/issues/1476)

## Unclear

- Query by partial key

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
