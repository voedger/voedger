/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*/

APPLICATION vrestaurant();

-- Declare tag to assign it later to definition(s)
TAG BackofficeTag;
TAG PosTag;

-- TABLE BOEntity : is an Abstract base data struct for many CDOC tables
ABSTRACT TABLE BOEntity INHERITS CDoc( -- TODO: ABSTRACT
    Name varchar(50) NOT NULL, -- TODO NOT NULL everywhere
    Number int NOT NULL -- Number sequence(1) ??? smm
) WITH Tags=(BackofficeTag);

-- TABLE Person : is an Abstract data struct for Waiters, Clients, Adminitsrators, Manager
ABSTRACT TABLE Person INHERITS BOEntity ( --TODO:  ABSTRACT
    Address varchar(50), 
    Email varchar(50) CHECK '^\\S+@\\S+\\.\\S+$',
    Phone varchar(20),
    Picture blob
) WITH Tags=(BackofficeTag);

WORKSPACE Restaurant (
    DESCRIPTOR (
	    Address varchar(50),
	    Currency varchar(3),
	    Phone varchar(20),
	    OpenHours    int,
	    OpenMinutes  int,
	    OwnerName varchar(30)
    );

    ROLE LocationUser;
    ROLE LocationManager;

    -- CDOC data schemes

    -- TABLE Client   : describes restaurant client entity
    TABLE Client INHERITS Person(
        -- access to alcohol
        Datebirth int64, 
        -- payment card number, used for payments in Restaurant
        Card varchar(20),
        -- percent of permanent discount
        DiscountPercent int 
    );

    -- TABLE Register   : describes payment resgitration devices
    TABLE Register INHERITS Person(
        Code varchar(20) -- personal code in inner login system    
    );

    -- TABLE Position   : Restaurant job list
    TABLE Position INHERITS BOEntity(
    );

    -- TABLE POSUser   : describes restaurant user entity (Waiter/Administrator/Manager)
    TABLE POSUser INHERITS Person(
        -- personal code in inner login system    
        Code varchar(20), 
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
            Tableno int,  
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
        DepartamentID ref(Department),
        -- article barcode to order by scanner
        Barcode varchar(20),  
        -- article sale price 
        Price currency, 
        -- V.A.T. in percent
        VatPercent currency, 
        -- Absolut V.A.T. value 
        Vat float32         
    );

    -- TABLE PaymentType : defines types of payment modes, using in Restaurant
    TABLE PaymentType INHERITS BOEntity(
        --inner value of type
        Kind int 
    );

    -- TABLE Transaction   : defines parameters of table, occupied by client
    TABLE Transaction INHERITS WDoc(
        Name varchar(50),
        Number int,
        Tableno int, 
        --time of very first order on table
        OpenTimeStamp timestamp, 
        -- time of final payment and closing table transaction
        CloseTimeStamp timestamp,
        -- POS user, who created made very first order
        CreatorID ref(POSUser) NOT NULL, 
        -- client, assigned to transaction
        Client ref(Client) NOT NULL 
    ) WITH Tags=(PosTag);

    -- TABLE Orders     : defines parameters of order on table. One transaction can have several orders
    TABLE Order INHERITS ODoc(
        TransactionID ref(Transaction) NOT NULL, 
        -- time of creating order
        OrdTimeStamp timestamp, 
        UserID ref(POSUser) NOT NULL, 
        -- TABLE order_items : the list articles, options, comments, from which order consists of
        OrderItem TABLE OrderItem (
            Order ref(Order) NOT NULL,
            -- can be null for text comments
            ArticleID ref(Article), 
            -- number of articles in order
            Quantity int,           
            -- text message, added to the order
            Comment varchar(50),
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
	    Number int,  
        -- time of Bill creating
        PayTimeStamp timestamp,
        Tips float32,
        -- TABLE BillPayments  : Defines set of payment methods related to bill
        BillPayment TABLE BillPayment (
            Bill ref(Bill) NOT NULL,
            PaymentTypeID ref(PaymentType) NOT NULL,
            -- amount of payment
            Amount currency
        )
    ) WITH Tags=(PosTag);

    EXTENSION ENGINE BUILTIN (
	

	    SYNC PROJECTOR UpdateTableStatus
            AFTER INSERT ON Transaction OR
	        AFTER EXECUTE WITH PARAM ON (Order, Bill)
		INTENTS(View(TableStatus));

	    PROJECTOR UpdateSalesReport
            AFTER INSERT ON Transaction OR 
	        AFTER EXECUTE WITH PARAM ON Bill
		INTENTS(View(SalesPerDay));

    );

-- ACLs
    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    -- GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Order TO LocationUser;
    -- GRANT EXECUTE ON COMMAND MakeOrder TO LocationUser;
    -- GRANT EXECUTE ON COMMAND MakePayment TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    -- VIEW TableStatus     : keeps actual status of table(free/occupied)
    VIEW TableStatus (
        Dummy int,
        TableNumber int,
        --  status of table(free/occupied)
        Status int,
        PRIMARY KEY ((Dummy), TableNumber)
    ) AS RESULT OF UpdateTableStatus;

    -- VIEW SalesPerDay     : sales report per day
    VIEW SalesPerDay(
        Year int32,
        Month int32, 
        Day int32, 
        Number int32, 
        DepartmentID ref(Department) NOT NULL,
        ArticleID ref(Article) NOT NULL,
        Quantity int32, --!!! Must be float32
        Amount int32,--!!! Must be Currency
        Vat int32, --!!! Must be float32
        VatPercent int32, --!!! Must be Currency
        PaymentTypeID ref(PaymentType) NOT NULL,
        PRIMARY KEY ((Year, Month, Day), Number)
    ) AS RESULT OF UpdateSalesReport;
);    

