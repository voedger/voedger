-- Copyright (c) 2020-present unTill Pro, Ltd.

-- note: this schema is for tests only. Voedger sys package uses copy of this schema
APPLICATION TEST();

ABSTRACT TABLE CRecord();
ABSTRACT TABLE WRecord();
ABSTRACT TABLE ORecord();

ABSTRACT TABLE CDoc INHERITS CRecord();
ABSTRACT TABLE ODoc INHERITS ORecord();
ABSTRACT TABLE WDoc INHERITS WRecord();

ABSTRACT TABLE Singleton INHERITS CDoc();

ABSTRACT WORKSPACE Workspace (
    TYPE CreateLoginUnloggedParams(
        Password varchar, -- OrderChanged
        Email varchar --OrderChanged
    );
    TYPE CreateLoginParams(
        --Login                       varchar, -- NodeRemoved
        AppName                     varchar,
        SubjectKind                 int32,
        WSKindInitializationData    varchar(1024),
        ProfileCluster              int64, -- ValueChanged: int32 in old version, int64 in new version
        ProfileToken                int64, -- ValueChanged: int32 in old version, int64 in new version
        NewField                    varchar -- appending field is allowed
    );
    TABLE AnotherOneTable INHERITS CDoc(
        A varchar,
        B varchar,
        D varchar, -- NodeInserted
        C int32 -- ValueChanged: varchar in old version, int32 in new version
    );
    EXTENSION ENGINE BUILTIN (
        COMMAND CreateLogin(CreateLoginParams, UNLOGGED CreateLoginUnloggedParams) RETURNS void;
    )
);

ALTERABLE WORKSPACE Profile(

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

    STORAGE Subject(
        GET SCOPE(COMMANDS, QUERIES)
    );

    STORAGE Http (
        READ SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE SendMail(
        INSERT SCOPE(PROJECTORS)
    );

    STORAGE CmdResult(
        INSERT SCOPE(COMMANDS)
    );

)
