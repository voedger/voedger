SCHEMA vrestaurant;

    ROLE LocationUser;
    ROLE LocationManager;

    TYPE Timestamp (
        dt int64
    );

    -- TYPE Order: describes data structure, retuned after executing Command Order
    TYPE Order (
        OrdTimeStamp Timestamp,
        UserID ref(POSUsers)
    );

    TYPE OrderItems( 
        ArticleID ref(Articles),
        Quantity int,
        Comment text,
        Price float32,
        Vat_percent float32,
        Vat float32
    );
    
    -- TYPE Payment: describes data structure,  retuned after executing Command Pay
    TYPE Payment (
        PayTimeStamp Timestamp,
        UserID ref(Users),
        Tips float32 
    );

    TYPE PaymentItems(
        payment_type int64,
        payment_name text,
        amount float32
    );

    -- Declare tag to assign it later to definition(s)
    TAG BackofficeTag;
    TAG PosTag;

    -- CDOC data schemes

    -- TABLE restaurant_settings: has commnon partameters for Restaurant
    --   name       : Restaurant name 
    --   location   : Full adrress
    --   phone      : Restaurant phone
    --   open_time  : Time of opening 
    --   owner_name : Owner name, adrress, phone ...
    TABLE RestaurantSettings INHERITS Singleton(
	    Name text,
	    Location text,
	    Currency text,
	    Phone text,
	    OpenTimeStamp Timestamp,
	    OwnerName text
    ) WITH Comment="Restaurant settings defines information about";

    -- TABLE BOEntity : is an Abstract base data struct for many CDOC tables
    --   Name   : Entity name 
    --   Number : Entity number
    TABLE BOEntity INHERITS CDoc(
        Name text,
        Number int
    ) WITH Tags=(BackofficeTag);

    -- TABLE Person : is an Abstract data struct for Waiters, Clients, Adminitsrators, Manager
    --   Address    : Person address
    --   Email      : email of person
    --   Phone      : phone number of person
    --   Picture    : Image of person
    TABLE Person INHERITS BOEntity (
	    Address text,
        Email text,
	    Phone text,
        Picture blob
    ) WITH Tags=(BackofficeTag);

    -- TABLE Clients    : describes restaurant client entity
    --   Datebirth      : Date of birth, can be needed to define age limitations
    --   Card           : Client payment card number, used for payments in Restaurant
    --   Discount_percent : Percent of permanent discount
    TABLE Clients INHERITS Person(
        Datebirth Timestamp,
        Card text,
        Discount_percent int
    ) WITH Comment="Client defines customer data";

    -- TABLE POSUsers   : describes restaurant user entity (Waiter/Administrator/Manager)
    --   Code       : personal code in inner login syustem    
    --   Position   : Name of position in Restaurant
    --   Wage       : Wage/salary rate
    TABLE POSUsers INHERITS Person(
        Code text,
	    Position text,
        Wage float32
    ) WITH Comment="Waiter defines restaurant staff data";

    -- TABLE TablePlan    : describes Physical plan of tables/bar counters. etc. in Restaurant.
    --   Picture          : Image of restaurant plan
    -- TABLE TableItems   : Each table plan has several tables
    --   Tableno          : Number of table    
    --   Color            : Default table color on plan 
    --   Chairs           : Number of chiar, assigned to the table
    --   Left, Top        : Table position on Table plan
    --   Width, Height    : Table size
    TABLE TablePlan INHERITS BOEntity (
        Picture blob,
        TableItems TABLE TableItems (
            TablePlan ref(TablePlan) NOT NULL,
            Tableno int,
    	    Color int,
            Chairs int,
            Left int,
            Top int,
 	        Width int,
	        Height int    
        )
    ) WITH Comment="Table plan defines restaurant tables schema";

    -- TABLE departments : defines Restaurant department button entity
    TABLE Departments INHERITS BOEntity (
    ) WITH Comment="Department defines department for article sets";

    -- TABLE Articles : defines Restaurant articles button entity
    --   Barcode      : article barcode to order by scanner
    --   Price        : article price
    --   VatPercent   : Vate in percent
    --   Vat          : Absolut Vate value
    TABLE Articles INHERITS BOEntity(
        DepartamentID ref(Departments),
        Barcode text,
        Price float32,
        VatPercent float32,
        Vat float32
    ) WITH Comment="Articles defines restaurant article sets";

    -- TABLE payment_types : defines types of payment modes, using in Restaurant
    --   Kind               : Inner value of type
    TABLE PaymentTypes INHERITS BOEntity(
        Kind int
    ) WITH Comment="Payment type defines parameters of Payment method";

    -- TABLE Transactions   : defines parameters of table, occupied by client
    --   Tableno            : Number of table
    --   Name               : Name of transaction
    --   OpenTimeStamp      : Time of very first order on table
    --   CloseTimeStamp     : Time of final payment and closing table transaction
    --   UserID            : Ref on User, who made very first order on table
    --   Client          : Ref on Client, paying the bill
    TABLE Transactions INHERITS WDoc(
        Name text,
        Number int,
        Tableno int, 
        OpenTimeStamp Timestamp,
        CloseTimeStamp Timestamp,
        UserID ref(POSUsers) NOT NULL, 
        Client ref(Clients) NOT NULL 
    ) WITH Tags=(PosTag), Comment="Transaction defines properties of ordered table";

    -- TABLE Orders     : defines parameters of order on table. One transaction can have several orders
    --   TransactionID : ref on parent transactions
    --   OrdTimeStamp   : Time of creating order
    --   UserID        : ref on user, who created the order
    -- TABLE order_items : the list articles, options, comments, from which order consists of
    --   ArticleID     : ref on article(can be null)
    --   Quantity       : Number of articles in order
    --   Comment        : Text message, added to the order
    --   Price          : Price of article, when order is created
    --   VatPercent    : VAT% of article
    --   Vat            : absolute value VAT of article
    TABLE Orders INHERITS ODoc(
        TransactionID ref(Transactions) NOT NULL, 
        OrdTimeStamp Timestamp,
        UserID ref(POSUsers) NOT NULL, 
        OrderItems TABLE OrderItem (
            Order ref(Orders) NOT NULL,
            ArticleID ref(Articles), -- can be null for text comments
            Quantity int,
            Comment text,
            Price float32,
            VatPercent float32,
            Vat float32
        )
    ) WITH Tags=(PosTag), Comment="Order defines act of ordering";

    -- TABLE Bill       : defines parameters of bill on table. One transaction can have several bills
    --   UserID         : ref of User, who took created bill and takes payment
    --   Number         : Bill number, unique per cash register 
    --   PayTimeStamp   : Time of Bill creating
    --   Tips           : waiter Tips amount, included into bill
    -- TABLE BillPayments  : Defines set of payment methods related to bill
    --   PaymentType    : ref on payment_kind, defines kind of payment
    --   Amount         : Amount of payment
    TABLE Bills INHERITS ODoc(
        TransactionID ref(Transactions) NOT NULL, 
        UserID ref(POSUsers) NOT NULL, 
        Number int,
        PayTimeStamp Timestamp,
        Tips float32,
        BillPayments TABLE BillPayments (
            Bill ref(Bills) NOT NULL,
            PaymentType ref(PaymentTypes) NOT NULL,
            Amount float32
        )
    ) WITH Tags=(PosTag), Comment="Bill defines act of payment";

WORKSPACE Restaurant (
    EXTENSION ENGINE BUILTIN (
    -- COMMAND order: creates Order ?
        COMMAND order(vrestaurant.Order) RETURNS vrestaurant.Order;
    -- COMMAND pay: pays bill ?
        COMMAND pay(vrestaurant.Payment) RETURNS vrestaurant.Payment;

        PROJECTOR UpdateTableStatus
            ON COMMAND IN (order, pay)
            INTENTS(View TableStatus);
    );

-- ACLs
    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Orders TO LocationUser;
    GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    -- VIEW TableStatus     : keeps actual status of table(free/occupied)
    --  TableNumber             : Table number
    --  Status   - state of table(free/occupied)
    VIEW TableStatus (
        TableNumber int,
        Status int,
        PRIMARY KEY ((TableNumber))        
    ) AS RESULT OF UpdateTableStatus;

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
