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

