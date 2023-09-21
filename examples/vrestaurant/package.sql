SCHEMA vrestaurant;

-- TABLE BOEntity : is an Abstract base data struct for many CDOC tables
ABSTRACT TABLE BOEntity INHERITS CDoc( 
    Name text NOT NULL, 
    Number int NOT NULL -- Number sequence(1) ??? smm
) WITH Tags=(BackofficeTag);

-- TABLE Person : is an Abstract data struct for Waiters, Clients, Adminitsrators, Manager
ABSTRACT TABLE Person INHERITS BOEntity ( --TODO:  ABSTRACT
    Address text, --TODO: get rid of text, use varchar, varchar(30) by default? smm
    Email text NOT NULL,
    Phone text,
    Picture blob
) WITH Tags=(BackofficeTag);

WORKSPACE Restaurant (
    DESCRIPTOR (
	    Address text,
	    Currency text NOT NULL,
	    Phone text,
	    OpenHours    int NOT NULL,
	    OpenMinutes  int NOT NULL,
	    OwnerName text
    );

    ROLE LocationUser;
    ROLE LocationManager;

    -- Declare tag to assign it later to definition(s)
    TAG BackofficeTag;
    TAG PosTag;

    -- CDOC data schemes

    -- TABLE Client   : describes restaurant client entity
    TABLE Client INHERITS Person(
        -- access to alcohol
        Datebirth int64, 
        -- payment card number, used for payments in Restaurant
        Card text,       
        -- percent of permanent discount
        DiscountPercent int 
    );

    -- TABLE Register   : describes payment resgitration devices
    TABLE Register INHERITS Person(
        Code text -- personal code in inner login system    
    );

    -- TABLE Position   : Restaurant job list
    TABLE Position INHERITS BOEntity(
    );

    -- TABLE POSUser   : describes restaurant user entity (Waiter/Administrator/Manager)
    TABLE POSUser INHERITS Person(
        -- personal code in inner login system    
        Code text NOT NULL, 
	    PositionID ref(Position),
        -- wage/salary rate
        Wage float32
    );

    -- TABLE TablePlan : describes Physical plan of tables/bar counters. etc. in Restaurant.
    TABLE TablePlan INHERITS BOEntity (
        -- Image of restaurant plan
        Picture blob, 
        Width int,
        Height int,
        -- List of tables on table plan
        TableItem TABLE TableItem (
            Tableno int NOT NULL,  
            -- color of empty table
    	    Color int,    
            -- number of table chairs
            Chairs int,
            Left int,
            Top int,
 	        Width int,
	        Height int
        )
    );

    -- TABLE departments : defines Restaurant department button entity
    TABLE Department INHERITS BOEntity (
    );

    -- TABLE Article : defines Restaurant articles button entity
    TABLE Article INHERITS BOEntity(
        DepartamentID ref(Department) NOT NULL,
        -- article barcode to order by scanner
        Barcode text,  
        -- article sale price 
        Price currency NOT NULL, 
        -- V.A.T. in percent
        VatPercent currency NOT NULL,
        -- Absolut V.A.T. value 
        Vat float32 NOT NULL        
    );

    -- TABLE PaymentType : defines types of payment modes, using in Restaurant
    TABLE PaymentType INHERITS BOEntity(
        --inner value of type
        Kind int NOT NULL
    );

    -- TABLE Transaction   : defines parameters of table, occupied by client
    TABLE Transaction INHERITS WDoc(
        Name text,
        Number int NOT NULL,
        Tableno ref(TableItem) NOT NULL, 
        --time of very first order on table
        OpenTimeStamp timestamp NOT NULL, 
        -- time of final payment and closing table transaction
        CloseTimeStamp timestamp,
        -- POS user, who created made very first order
        CreatorID ref(POSUser) NOT NULL, 
        -- client, assigned to transaction
        Client ref(Client)
    ) WITH Tags=(PosTag);

    -- TABLE Orders     : defines parameters of order on table. One transaction can have several orders
    TABLE Order INHERITS ODoc(
        TransactionID ref(Transaction) NOT NULL, 
        -- time of creating order
        OrdTimeStamp timestamp NOT NULL, 
        UserID ref(POSUser) NOT NULL, 
        -- TABLE order_items : the list articles, options, comments, from which order consists of
        OrderItem TABLE OrderItem (
            Order ref(Order) NOT NULL,
            -- can be null for text comments
            ArticleID ref(Article),  
            -- number of articles in order
            Quantity int,           
            -- text message, added to the order
            Comment text,           
            Price currency,
            VatPercent currency,
            Vat float32
        )
    ) WITH Tags=(PosTag);

    -- TABLE Bill          : defines parameters of bill on table. One transaction can have several bills
    TABLE Bill INHERITS ODoc(
        TransactionID ref(Transaction) NOT NULL, 
        --   ref of POSUser, who took created Transaction
       	AuthorID ref(POSUser) NOT NULL,
	    RegisterID ref(Register) NOT NULL,
        -- bill number, unique per cash register   
	    Number int NOT NULL,  
        -- time of Bill creating
        PayTimeStamp timestamp NOT NULL,
        Tips float32,
        -- TABLE BillPayments  : Defines set of payment methods related to bill
        BillPayment TABLE BillPayment (
            Bill ref(Bill) NOT NULL,
            PaymentTypeID ref(PaymentType) NOT NULL,
            -- amount of payment
            Amount currency NOT NULL
        )
    ) WITH Tags=(PosTag);

    EXTENSION ENGINE WASM (
	    SYNC PROJECTOR UpdateTableStatus
	        ON INSERT IN (Order, Bill)
		INTENTS(View TableStatus);
    );

    EXTENSION ENGINE BUILTIN (
        PROJECTOR UpdateSalesReport
	        ON INSERT Bill 
		INTENTS(View SalesPerDay);
    );

-- ACLs
    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    -- VIEW TableStatus     : keeps actual status of table(free/occupied)
    VIEW TableStatus(
        Tableno int,
        -- Not paid sum
        NotPaid currency, 
        -- status of table(free/occupied)
        Status int,
        PRIMARY KEY (Tableno)
    ) AS RESULT OF UpdateTableStatus;

    -- VIEW SalesPerDay     : sales report per day
    VIEW SalesPerDay(
        Year int32,
        Month int32, 
        Day int32, 
        Number int32, 
        DepartmentID id NOT NULL,
        ArticleID id NOT NULL,
        Quantity int32, --!!! Must be float32
        Amount int32,--!!! Must be Currency
        Vat int32, --!!! Must be float32
        VatPercent int32, --!!! Must be Currency
        PaymentTypeID id NOT NULL,
        PRIMARY KEY (Year, Month, Day, Number)
    ) AS RESULT OF UpdateSalesReport;
);    

