-- noinspection SqlNoDataSourceInspectionForFile

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
    TYPE CreateLoginParams(
        Login                       varchar,
        AppName                     varchar,
        SubjectKind                 int32,
        WSKindInitializationData    varchar(1024),
        ProfileCluster              int32,
        ProfileToken                int32
    );
    TYPE CreateLoginUnloggedParams(
        Email varchar,
        Password varchar
    );
    TABLE SomeTable INHERITS CDoc( -- NodeRemoved: removed in new.sql
        A varchar,
        B varchar
    );
    TABLE AnotherOneTable INHERITS CDoc(
        A varchar,
        B varchar,
        C varchar
    );
    TYPE SomeType(
        A varchar,
        B int
    );
    TYPE SomeType2(
        A varchar,
        B int,
        C int,
        D int
    );
    VIEW SomeView(
        A int,
        B int,
        PRIMARY KEY ((A), B)
    ) AS RESULT OF NewType;
    EXTENSION ENGINE BUILTIN (
        COMMAND CreateLogin(CreateLoginParams, UNLOGGED CreateLoginUnloggedParams) RETURNS void;
        COMMAND SomeCommand(SomeType, UNLOGGED SomeType) RETURNS SomeType;
    )
);

ALTERABLE WORKSPACE Profile(
    TABLE ProfileTable INHERITS CDoc(-- NodeRemoved: removed in new.sql
        A varchar
    );
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
