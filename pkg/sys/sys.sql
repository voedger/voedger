-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

ABSTRACT TABLE CRecord();
ABSTRACT TABLE WRecord();
ABSTRACT TABLE ORecord();

ABSTRACT TABLE CDoc INHERITS CRecord();
ABSTRACT TABLE ODoc INHERITS ORecord();
ABSTRACT TABLE WDoc INHERITS WRecord();

ABSTRACT TABLE Singleton INHERITS CDoc();

ABSTRACT WORKSPACE Workspace (
    -- cdoc.sys.Login must be known in each target app. "unknown ownerQName scheme cdoc.sys.Login" on c.sys.CreatWorkspaceID otherwise
    -- has no ownerApp field because it is sys/registry always
    TABLE Login INHERITS CDoc (
        ProfileCluster int32 NOT NULL,
        PwdHash bytes NOT NULL,
        AppName varchar NOT NULL,
        SubjectKind int32,
        LoginHash varchar NOT NULL,
        WSID int64,      -- to be written after workspace init
        WSError varchar(1024), -- to be written after workspace init
        WSKindInitializationData varchar(1024) NOT NULL
    );

    -- many, target app, user profile
    TABLE ChildWorkspace INHERITS CDoc (
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSClusterID int32 NOT NULL,
        WSID int64,  -- to be updated afterwards
        WSError varchar(1024) -- to be updated afterwards
    );

    -- target app, (target cluster, base profile WSID)
    TABLE WorkspaceID INHERITS CDoc (
        OwnerWSID int64 NOT NULL,
        OwnerQName qname NOT NULL,
        OwnerID int64 NOT NULL,
        OwnerApp varchar NOT NULL,
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSID int64
    );

    -- target app, new WSID
    TABLE WorkspaceDescriptor INHERITS Singleton (
        OwnerWSID int64, -- owner* fields made non-required for app workspaces
        OwnerQName qname,
        OwnerID int64,
        OwnerApp varchar, -- QName -> each target app must know the owner QName -> string
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSID int64,
        CreateError varchar(1024),
        CreatedAtMs int64 NOT NULL,
        InitStartedAtMs int64,
        InitError varchar(1024),
        InitCompletedAtMs int64,
        Status int32
    );

	TYPE EchoParams (
		Text text NOT NULL
	);

	TYPE EchoResult (
		Res text NOT NULL
	);

	EXTENSION ENGINE BUILTIN (
		QUERY Echo(EchoParams) RETURNS EchoResult;

		-- Login is randomly taken name because it is required to specify something in the sql. Actually the projector will start on an any document.
		SYNC PROJECTOR RecordsRegistryProjector ON (Login) INTENTS(View(RecordsRegistry));
	);

	VIEW RecordsRegistry (
		IDHi int64 NOT NULL,
		ID ref NOT NULL,
		WLogOffset int64 NOT NULL,
		QName qname NOT NULL,
		PRIMARY KEY ((IDHi), ID)
	) AS RESULT OF sys.RecordsRegistryProjector;

    TABLE UserProfile INHERITS Singleton (
        DisplayName varchar
    );

    TABLE DeviceProfile INHERITS Singleton ();

    TABLE AppWorkspace INHERITS Singleton ();

    TABLE BLOB INHERITS WDoc (
        status int32 NOT NULL
    );

    TABLE Subject INHERITS CDoc (
        Login varchar NOT NULL,
        SubjectKind int32 NOT NULL,
        Roles varchar(1024) NOT NULL,
        ProfileWSID int64 NOT NULL,
        UNIQUEFIELD Login
    );

    TABLE Invite INHERITS CDoc (
        SubjectKind int32,
        Login varchar NOT NULL,
        Email varchar NOT NULL,
        Roles varchar(1024),
        ExpireDatetime int64,
        VerificationCode varchar,
        State int32 NOT NULL,
        Created int64,
        Updated int64 NOT NULL,
        SubjectID ref,
        InviteeProfileWSID int64,
        UNIQUEFIELD Email
    );

    TABLE JoinedWorkspace INHERITS CDoc (
        Roles varchar(1024) NOT NULL,
        InvitingWorkspaceWSID int64 NOT NULL,
        WSName varchar NOT NULL
    );
    EXTENSION ENGINE BUILTIN ()
);

EXTENSION ENGINE BUILTIN (

    STORAGE Record(
        GET         SCOPE(COMMANDS, QUERIES, PROJECTORS),
        GETBATCH    SCOPE(COMMANDS, QUERIES, PROJECTORS),
        INSERT      SCOPE(COMMANDS),
        UPDATE      SCOPE(COMMANDS)
    ) ENTITY RECORD; -- used to validate projector state/intents declaration


    STORAGE View(
        GET         SCOPE(COMMANDS, QUERIES, PROJECTORS),
        GETBATCH    SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ        SCOPE(QUERIES, PROJECTORS),
        INSERT      SCOPE(PROJECTORS),
        UPDATE      SCOPE(PROJECTORS)
    ) ENTITY VIEW;

    STORAGE WLog(
        GET     SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ    SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE PLog(
        GET     SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ    SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE AppSecret(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS)
    );

    STORAGE RequestSubject(
        GET SCOPE(COMMANDS, QUERIES)
    );

    STORAGE Http (
        READ SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE SendMail(
        INSERT SCOPE(PROJECTORS)
    );

    STORAGE Result(
        INSERT SCOPE(COMMANDS)
    );

)

