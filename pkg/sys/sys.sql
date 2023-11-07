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
    TABLE ChildWorkspace INHERITS CDoc (
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSClusterID int32 NOT NULL,
        WSID int64,           -- to be updated afterwards
        WSError varchar(1024) -- to be updated afterwards
    );

    -- target app, (target cluster, base profile WSID)
    TABLE WorkspaceID INHERITS CDoc (
        OwnerWSID int64 NOT NULL,
        OwnerQName qname, -- Deprecated: use OwnerQName2
        OwnerID int64 NOT NULL,
        OwnerApp varchar NOT NULL,
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSID int64,
        OwnerQName2 text
    );

    -- target app, new WSID
    TABLE WorkspaceDescriptor INHERITS Singleton (
        -- owner* fields made non-required for app workspaces
        OwnerWSID int64,
        OwnerQName qname, -- Deprecated: use OwnerQName2
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
        Status int32,
        OwnerQName2 text
    );

    TYPE EchoParams (Text text NOT NULL);

    TYPE EchoResult (Res text NOT NULL);

    EXTENSION ENGINE BUILTIN (
        -- COMMAND CreateLogin (CreateLoginParams, UNLOGGED CreateLoginUnloggedParams);
        QUERY Echo(EchoParams) RETURNS EchoResult;

    SYNC PROJECTOR RecordsRegistryProjector ON (CDoc, WDoc, ODoc) INTENTS(View(RecordsRegistry));
);

VIEW RecordsRegistry (
    IDHi int64 NOT NULL,
    ID ref NOT NULL,
    WLogOffset int64 NOT NULL,
    QName qname NOT NULL,
    PRIMARY KEY ((IDHi), ID)
) AS RESULT OF sys.RecordsRegistryProjector;

TABLE UserProfile INHERITS Singleton (DisplayName varchar);

TABLE DeviceProfile INHERITS Singleton ();

TABLE AppWorkspace INHERITS Singleton ();

TABLE BLOB INHERITS WDoc (status int32 NOT NULL);

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

);

EXTENSION ENGINE BUILTIN (
    STORAGE Record(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
        GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
        INSERT
            SCOPE(COMMANDS),
        UPDATE
            SCOPE(COMMANDS)
    ) ENTITY RECORD;

-- used to validate projector state/intents declaration
STORAGE View(
    GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
    GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
    READ SCOPE(QUERIES, PROJECTORS),
    INSERT
        SCOPE(PROJECTORS),
    UPDATE
        SCOPE(PROJECTORS)
) ENTITY VIEW;

STORAGE WLog(
    GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
    READ SCOPE(QUERIES, PROJECTORS)
);

STORAGE PLog(
    GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
    READ SCOPE(QUERIES, PROJECTORS)
);

STORAGE AppSecret(GET SCOPE(COMMANDS, QUERIES, PROJECTORS));

STORAGE RequestSubject(GET SCOPE(COMMANDS, QUERIES));

STORAGE Http (READ SCOPE(QUERIES, PROJECTORS));

STORAGE SendMail(
    INSERT
        SCOPE(PROJECTORS)
);

STORAGE Result(
    INSERT
        SCOPE(COMMANDS)
);

)