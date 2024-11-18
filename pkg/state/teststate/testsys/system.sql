-- Copyright (c) 2020-present unTill Pro, Ltd.

-- note: this schema is for tests only

ABSTRACT WORKSPACE Workspace(
	ABSTRACT TABLE CRecord();
	ABSTRACT TABLE WRecord();
	ABSTRACT TABLE ORecord();
	ABSTRACT TABLE CDoc INHERITS CRecord();
	ABSTRACT TABLE ODoc INHERITS ORecord();
	ABSTRACT TABLE WDoc INHERITS WRecord();
	ABSTRACT TABLE CSingleton INHERITS CDoc();

	TYPE Raw(
		Body varchar(65535)
	);

	TABLE WorkspaceDescriptor INHERITS sys.CSingleton (
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

	TABLE ChildWorkspace INHERITS sys.CDoc (
		WSName varchar NOT NULL,
		WSKind qname NOT NULL,
		WSKindInitializationData varchar(1024),
		TemplateName varchar,
		TemplateParams varchar(1024),
		WSClusterID int32 NOT NULL,
		WSID int64,           -- to be updated afterwards
		WSError varchar(1024) -- to be updated afterwards
	);
);

ALTERABLE WORKSPACE Profile();

EXTENSION ENGINE BUILTIN (

	STORAGE Record(
		/*
		Key:
			ID int64 // used to identify record by ID
			Singletone QName // used to identify singleton
		*/
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
		INSERT SCOPE(COMMANDS),
		UPDATE SCOPE(COMMANDS)
	) ENTITY RECORD;

	-- used to validate projector state/intents declaration
	STORAGE View(
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
		READ SCOPE(QUERIES, PROJECTORS),
		INSERT SCOPE(PROJECTORS),
		UPDATE SCOPE(PROJECTORS)
	) ENTITY VIEW;

	STORAGE Uniq(
		/*
		Key:
			One or more unique fields
		Value:
			ID int64 (record ID)
		*/
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS)
	) ENTITY RECORD;

	STORAGE WLog(
		/*
		Key:
			Offset int64
			Count int64 (used for Read operation only)
		Value
			RegisteredAt int64
			SyncedAt int64
			DeviceID int64
			Offset int64
			Synced bool
			QName qname
			CUDs []value {
				IsNew bool
				...CUD fields...
			}
		*/
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		READ SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE AppSecret(
		/*
		Key:
			Secret text
		Value:
			Content text
		*/
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS)
	);

	STORAGE RequestSubject(
		/*
		Key: empty
		Value:
			ProfileWSID int64
			Kind int32
			Name text
			Token texts
		*/
		GET SCOPE(COMMANDS, QUERIES)
	);

	STORAGE Http(
		/*
		Key:
			Method text
			Url text
			Body []byte
			HTTPClientTimeoutMilliseconds int64
			Header text (can be called multiple times)
		Value:
			StatusCode int32
			Body []byte
			Header text (headers combined)

		*/
		READ SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE FederationCommand(
		/*
		Key:
			Owner text (optional, default is current app owner)
			AppName text (optional, default is current app name)
			WSID int64 (optional, default is current workspace)
			Token text (optional, default is system token)
			Command qname
			Body text
		Value:
			StatusCode int32
			NewIDs value {
				rawID1: int64
				rawID2: int64
				...
			}
			Result: value // command result
		*/
		GET SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE FederationBlob(
		/*
		Key:
			Owner text (optional, default is current app owner)
			AppName text (optional, default is current app name)
			WSID int64 (optional, default is current workspace)
			Token text (optional, default is system token)
			BlobID int64
			ExpectedCodes text (optional, comma-separated, default is 200)
		Value:
			Body: []byte // blob content, returned in chunks up to 1024 bytes
		*/
		READ SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE SendMail(
		/*
		Key:
			From text
			To text
			CC text
			BCC text
			Host text - SMTP server
			Port int32 - SMTP server
			Username text - SMTP server
			Password text - SMTP server
			Subject text
			Body text

		*/
		INSERT SCOPE(PROJECTORS)
	);

	STORAGE Result(
		/*
		Key: empty
		ValueBuilder: depends on the result of the Command or Query
		*/
		INSERT SCOPE(COMMANDS, QUERIES)
	);

	STORAGE Response(
		/*
		Key: empty
		ValueBuilder:
			StatusCode int32
			ErrorMessage text
		*/
		INSERT SCOPE(COMMANDS, QUERIES)
	);

	STORAGE Event(
		/*
		Key: empty
		Value
			WLogOffset int64
			Workspace int64
			RegisteredAt int64
			SyncedAt int64
			DeviceID int64
			Offset int64
			Synced bool
			QName qname
			Error value {
				ErrStr text
				ValidEvent bool
				QNameFromParams qname
			}
			ArgumentObject value
			CUDs []value {
				IsNew bool
				...CUD fields...
			}
		*/
		GET SCOPE(PROJECTORS)
	);

	STORAGE CommandContext(
		/*
		Key: empty
		Value
			Workspace int64
			WLogOffset int64
			ArgumentObject value
			ArgumentUnloggedObject value
		*/
		GET SCOPE(COMMANDS)
	);

	STORAGE QueryContext(
		/*
		Key: empty
		Value
			Workspace int64
			WLogOffset int64
			ArgumentObject value
		*/
		GET SCOPE(QUERIES)
	);

	STORAGE JobContext(
		/*
		Key: empty
		Value
			Workspace int64
			UnixTime int64
		*/
		GET SCOPE(JOBS)
	);

	STORAGE Logger(
		/*
		Key:
			LogLevel int32
		Value
			Message text
		*/
		INSERT SCOPE(COMMANDS, QUERIES, PROJECTORS, JOBS)
	);
)
