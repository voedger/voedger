-- Copyright (c) 2020-present unTill Pro, Ltd.

SCHEMA untill

TABLE bill OF WDOC (
    tableno int32 not null,
    id_untill_users id references untill_users,
    table_part text,
    id_courses id references courses,
    id_clients id references clients,
    name text
    -- ...
);


PROCEDURE MyOrdersValidator2(sys.TableRow) ENGINE WASM;
TABLE orders OF ODOC (
    working_day text not null check('^[0-9]{8}$'),
    ord_datetime int64,
    id_bill id references bill,
    -- ...
    TABLE order_item {
        id_orders id references orders,
        quantity int32,
        id_prices id references prices,
        price int64,

        id_articles id references articles,
        -- ...
    }
    CHECK MyOrdersCheck AS PROCEDURE MyOrdersValidator(sys.TableRow) ENGINE WASM,
    CHECK MyOrdersCheck2 IS MyOrdersValidator2
) WITH Tag IS PosTag;


TABLE articles OF CDOC {
    name text,
    number int32,
    id_department id references departments,
    -- ...
}

TABLE currency OF CDOC (
    code text NOT NULL,
    name text NOT NULL,
    round int32 NOT NULL CHECK(round>=0 && round<=4),
    round int32 NOT NULL CHECK MyValidator,
    eurozone boolean NOT NULL,
    rate float32 NOT NULL,
    symbol text,
    sym_alignment int32,
    digcode int32,
    round_down int64,
    CHECK MyValidator(round, rate)

)