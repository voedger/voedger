SCHEMA vrestaurant;

-- TYPE Order: describes data structure, retuned after executing Command Order
TYPE Order (
    ord_datetime int64,
    id_user int64
);

TYPE OrderItems( 
    id_article int64,
    quantity int,
    comment text,
    price float32,
    vat_percent float32,
    vat float32
);
-- TYPE Payment: describes data structure,  retuned after executing Command Pay
TYPE Payment (
    pay_datetime int64,
    id_user int64,
    tips float32 
);

TYPE TypeNameNumber (
    Name text,
    Number int
);  
    

TYPE PaymentItems(
    payment_type int64,
    payment_name text,
    amount float32
);

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;
TAG PosTag;

-- Comments section uses to predefine comments to schema data structs
-- Comments for CDOC schemes
COMMENT SettingComment "Restaurant settings defines information about";
COMMENT TablePlanComment "Table plan defines restaurant tables schema";
COMMENT UserComment "User defines restaurant staff data";
COMMENT ClientComment "Client defines customer data";
COMMENT DepartmentComment "Department defines department for article sets";
COMMENT ArticlePriceComment "Article price defines price for article by price level";
COMMENT PaymentTypeComment "Payment type defines parameters of Payment method";

-- Comments for WDOC/ODOC schemes
COMMENT TransactionComment "Transaction defines properties of ordered table";
COMMENT OrderComment "Order defines act of ordering";
COMMENT BillComment "Bill defines act of payment";
COMMENT ChefOrderQueueComment "Queue of orders on cook screen";

--      need to add in future
--      Tables:
--          ums  - unity of measure
--          ingredients
--          recipes
--          suppliers
--          stock_orders
--          stock_order_items

-- WDOC data schemes

-- TABLE restaurant_settings: has commnon partameters for Restaurant
--   name       : Restaurant name 
--   location   : Full adrress
--   phone      : Restaurant phone
--   open_time  : Time of opening 
--   owner_name : Owner name, adrress, phone ...
TABLE restaurant_settings INHERITS Singleton(
	name text,
	location text,
	currency text,
	phone text,
	open_time int64,
	owner_name text
) WITH Comment=SettingComment;

-- TABLE boEntity : is an Abstract base data struct for many CDOC tables
--   name   : Entity name 
--   number : Entity number
TABLE boEntity INHERITS CDoc OF TypeNameNumber(
) WITH Tags=[BackofficeTag];

-- TABLE price_levels: describes Price level entity. 
-- Restaurant can use different price levels for f.e. outside, inside and take away servives. 
TABLE price_levels INHERITS boEntity(
    
);

-- TABLE person : is an Abstract data struct for Waiters, Clients, Adminitsrators, Manager
--   address    : Person address
--   email      : email of person
--   phone      : phone number of person
--   picture    : Image of person
TABLE person INHERITS boEntity (
	address text,
    email text,
	phone text,
    picture blob
) WITH Tags=[BackofficeTag];

-- TABLE clients    : describes restaurant client entity
--   datebirth      : Date of birth, can be needed to define age limitations
--   card           : Client payment card number, used for payments in Restaurant
--   id_price_level : link to Price level, using for this client. Can be null(empty)
TABLE clients INHERITS person(
    datebirth int64,
    card text,
    discount_percent int,
    id_price_level ref(price_levels)
) WITH Comment=ClientComment;

-- TABLE user   : describes restaurant user entity (Waiter/Administrator/Manager)
--   code       : personal code in inner login syustem    
--   position   : Name of position in Restaurant
--   wage       : Wage/salary rate
TABLE users INHERITS person(
    code text,
	position text,
    wage float32
) WITH Comment=UserComment;

-- TABLE table_plan   : describes Physical plan of tables/bar counters. etc. in Restaurant.
--   picture          : Image of restaurant plan
-- TABLE table_items  : Each table plan has several tables
--   tableno          : Number of table    
--   color            : Default table color on plan 
--   chairs           : Number of chiar, assigned to the table
--   left, top        : Table position on Table plan
--   width, height    : Table size
TABLE table_plan INHERITS boEntity (
    picture blob,
    table_items TABLE table_items (
        id_table_plan ref(table_plan) NOT NULL,
        tableno int,
    	color int,
        chairs int,
        left int,
        top int,
 	    width int,
	    height int    
    )
) WITH Comment=TablePlanComment;

-- TABLE btn_entity : is an Abstract entity for items on buttons, like departments/articles/options/menus.
--   sequence : serial number of item list
--   color    : Background button color
--   picture  : Item image
TABLE btn_entity INHERITS boEntity (
    sequence int,
    color int,
    picture blob
);

-- TABLE departments : defines Restaurant department button entity
TABLE departments INHERITS btn_entity (
) WITH Comment=DepartmentComment;

-- TABLE articles : defines Restaurant articles button entity
--   barcode        : article barcode to order by scanner
--   prep_time_min  : default time of article preparation
TABLE articles INHERITS btn_entity(
    id_departament ref(departments),
    barcode text,
    prep_time_min int
) WITH Tags=[BackofficeTag];

-- TABLE article_prices : defines prices of articles for different price levels
--   price              : Sale article price
--   vat_percent        : article VAT in percents
--   vat                : article VAT absolute value
--   vat_sign           : signature of VAT
--   prices, vat are the values in currency from restaurant_settings
TABLE article_prices INHERITS CDoc(
   id_price_level ref(price_levels) NOT NULL, 
   id_article ref(articles) NOT NULL, 
   price float32,
   vat_percent float32,
   vat float32,
   vat_sign text
) WITH Tags=[BackofficeTag], Comment=ArticlePriceComment;

-- TABLE payment_types : defines types of payment modes, using in Restaurant
--   kind               : Inner value of type
TABLE payment_types INHERITS boEntity(
    kind int
);

-- TABLE transactions   : defines parameters of table, occupied by client
--   tableno            : Number of table
--   name               : Name of transaction
--   open_datetime      : Time of very first order on table
--   close_datetime     : Time of final payment and closing table transaction
--   id_user            : Ref on User, who made very first order on table
--   id_client          : Ref on Client, paying the bill
TABLE transactions INHERITS WDoc OF TypeNameNumber(
    tableno int, 
    open_datetime int64,
    close_datetime int64,
    id_user ref(users) NOT NULL, 
    id_client ref(clients) NOT NULL 
) WITH Tags=[PosTag], Comment=TransactionComment;

-- TABLE orders     : defines parameters of order on table. One transaction can have several orders
--   id_transaction : ref on parent transactions
--   ord_datetime   : Time of creating order
--   id_user        : ref on user, who created the order
-- TABLE order_items : the list articles, options, comments, from which order consists of
--   id_article     : ref on article(can be null)
--   quantity       : Number of articles in order
--   comment        : Text message, added to the order
--   price          : Price of article, when order is created
--   vat_percent    : VAT% of article
--   vat            : absolute value VAT of article
TABLE orders INHERITS ODoc(
    id_transaction ref(transactions) NOT NULL, 
    ord_datetime int64,
    id_user ref(users) NOT NULL, 
    order_items TABLE order_items (
        id_order ref(orders) NOT NULL,
        id_article ref(articles), -- can be null for text comments
        quantity int,
        comment text,
        price float32,
        vat_percent float32,
        vat float32
    )
) WITH Tags=[PosTag], Comment=OrderComment;

-- TABLE bill       : defines parameters of bill on table. One transaction can have several bills
--   id_user        : ref of User, who took created bill and takes payment
--   number         : Bill number, unique per cash register 
--   pay_datetime   : Time of Bill creating
--   tips           : waiter Tips amount, included into bill
-- TABLE bill_payments  : Defines set of payment methods related to bill
--   payment_type   : ref on payment_kind, defines kind of payment
--   amount         : Amount of payment
TABLE bills INHERITS ODoc(
    id_transaction ref(transactions) NOT NULL, 
    id_user ref(users) NOT NULL, 
    number int,
    pay_datetime int64,
    tips float32,
    bill_payments TABLE bill_payments (
        id_bill ref(bills) NOT NULL,
        payment_type ref(payment_types) NOT NULL,
        amount float32
    )
) WITH Tags=[PosTag], Comment=BillComment;


-- TABLE ChefOrderQueue : defines set of Bill in Cook screen
--  sequence            : sertial number of transactuon in list 
--  status              : state of the transaction(Pending, Active, Ready, Delivered)
TABLE ChefOrderQueue INHERITS ODoc(
    id_transaction ref(transactions) NOT NULL, 
    id_user ref(users) NOT NULL, 
    sequence int,
    status int
) WITH Tags=[PosTag], Comment=ChefOrderQueueComment;

WORKSPACE Restaurant (
    EXTENSION ENGINE BUILTIN (
    -- COMMAND order: creates Order ?
        COMMAND order(vrestaurant.Order) RETURNS vrestaurant.Order;
    -- COMMAND pay: pays bill ?
        COMMAND pay(vrestaurant.Payment) RETURNS vrestaurant.Payment;
        PROJECTOR UpdateSales ON COMMAND IN (order, pay) MAKES SalesView;
    );

    -- VIEW TableStatus     : keeps actual status of table(free/occupied)
    --  tableno             : Table number
    --  status   - state of table(free/occupied)
    TABLE table_status INHERITS ODoc(
        tableno int,
        status int
    );

    -- VIEW XZReports   : keeps printed XZ reports 
    VIEW XZReports(
        Year int32,
        Month int32, 
        Day int32, 
        Kind int32, 
        Number int32, 
        XZReportWDocID id NOT NULL,
        PRIMARY KEY ((Year), Month, Day, Kind, Number)
    ) AS RESULT OF vrestaurant.UpdateSales;

);    
