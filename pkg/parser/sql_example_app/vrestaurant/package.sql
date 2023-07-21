SCHEMA vrestaurant;

WORKSPACE Restaurant (
    DESCRIPTOR (
	    Address text,
	    Currency text,
	    Phone text,
	    OpenTimeStamp int64, -- TODO timestamp
	    OwnerName text
    );

    ROLE LocationUser;
    ROLE LocationManager;

    -- Declare tag to assign it later to definition(s)
    TAG BackofficeTag;
    TAG PosTag;

    -- CDOC data schemes

    -- TABLE BOEntity : is an Abstract base data struct for many CDOC tables
    TABLE BOEntity INHERITS CDoc( 
        Name text, 
        Number int
    ) WITH Tags=(BackofficeTag);

    -- TABLE Person : is an Abstract data struct for Waiters, Clients, Adminitsrators, Manager
    TABLE Person INHERITS BOEntity ( 
	    Address text,
        Email text,
	    Phone text,
        Picture blob
    ) WITH Tags=(BackofficeTag);

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
        Code text, 
	    PositionID ref(Position),
        -- wage/salary rate
        Wage float32
    );

    -- TABLE TablePlan    : describes Physical plan of tables/bar counters. etc. in Restaurant.
    TABLE TablePlan INHERITS BOEntity (
        -- Image of restaurant plan
        Picture blob, 
        Width int,
        Height int,
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
    ) WITH Comment="Table plan defines restaurant tables schema";

    -- TABLE departments : defines Restaurant department button entity
    TABLE Department INHERITS BOEntity (
    );

    -- TABLE Article : defines Restaurant articles button entity
    TABLE Article INHERITS BOEntity(
        DepartamentID ref(Department),
        -- article barcode to order by scanner
        Barcode text,  
        -- article sale price 
        Price float32, 
        -- V.A.T. in percent
        VatPercent float32, 
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
        Name text,
        Number int,
        Tableno int, 
        --time of very first order on table
        OpenTimeStamp int64, 
        -- time of final payment and closing table transaction
        CloseTimeStamp int64,
        -- POS user, who created made very first order
        CreatorID ref(POSUser) NOT NULL, 
        -- client, assigned to transaction
        Client ref(Client) NOT NULL 
    ) WITH Tags=(PosTag);

    -- TABLE Orders     : defines parameters of order on table. One transaction can have several orders
    -- TABLE order_items : the list articles, options, comments, from which order consists of
    TABLE Order INHERITS ODoc(
        TransactionID ref(Transaction) NOT NULL, 
        -- time of creating order
        OrdTimeStamp int64, 
        UserID ref(POSUser) NOT NULL, 
        OrderItem TABLE OrderItem (
            Order ref(Order) NOT NULL,
            -- can be null for text comments
            ArticleID ref(Article), 
            -- number of articles in order
            Quantity int,           
            -- text message, added to the order
            Comment text,           
            Price int64,
            VatPercent float32,
            Vat float32
        )
    ) WITH Tags=(PosTag);

    -- TABLE Bill          : defines parameters of bill on table. One transaction can have several bills
    -- TABLE BillPayments  : Defines set of payment methods related to bill
    TABLE Bills INHERITS ODoc(
        TransactionID ref(Transaction) NOT NULL, 
        --   ref of POSUser, who took created Transaction
       	AuthorID ref(POSUser) NOT NULL,
	    RegisterID ref(Register) NOT NULL,
        -- bill number, unique per cash register   
	    Number int,  
        -- time of Bill creating
        PayTimeStamp int64,
        Tips float32,
        BillPayments TABLE BillPayments (
            Bill ref(Bill) NOT NULL,
            PaymentType ref(PaymentTypes) NOT NULL,
            -- amount of payment
            Amount int64
        )
    ) WITH Tags=(PosTag);

    EXTENSION ENGINE BUILTIN (
	    COMMAND MakeOrder(vrestaurant.Order);
	    COMMAND MakePayment(vrestaurant.Payment);
	
	    PROJECTOR UpdateTableStatus
	        ON COMMAND IN (Order, MakePayment)
		INTENTS(View TableStatus);

	    PROJECTOR UpdateSalesReport
	        ON COMMAND IN (MakePayment)
		INTENTS(View SalesPerDay);

    );

-- ACLs
    GRANT ALL ON ALL TABLES WITH TAG BackofficeTag TO LocationManager;
    GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    GRANT SELECT ON TABLE Order TO LocationUser;
    GRANT EXECUTE ON COMMAND MakeOrder TO LocationUser;
    GRANT EXECUTE ON COMMAND MakePayment TO LocationUser;
    GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;

    -- VIEW TableStatus     : keeps actual status of table(free/occupied)
    VIEW TableStatus (
        TableNumber int,
        --  status of table(free/occupied)
        Status int,
        PRIMARY KEY (TableNumber)
    ) AS RESULT OF UpdateTableStatus;

    -- VIEW SalesPerDay     : sales report per day
    VIEW SalesPerDay(
        Year int32,
        Month int32,
        Day int32,
        Number int32, 
        Article text,
        -- can be f.e. 0.34
        Quantity float32, 
        Price int64,
        Vat float64,
        VatPercent float64,
        PaymentType ref(PaymentTypes) NOT NULL,
        PRIMARY KEY (Year, Month, Day, Number)
    ) AS RESULT OF UpdateSalesReport;
);    
