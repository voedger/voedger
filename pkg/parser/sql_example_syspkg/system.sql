-- note: this schema is for tests only. Voedger sys package uses copy of this schema
SCHEMA test_sys;
TABLE CDoc();
TABLE ODoc();
TABLE WDoc();
TABLE Singleton INHERITS CDoc();
TABLE CRecord();
TABLE WRecord();
TABLE ORecord();

EXTENSION ENGINE BUILTIN (

    STORAGE Table( 
        GET,
        GETBATCH,
        INSERT IN COMMANDS,
        UPDATE IN COMMANDS
    ) REQUIRES ENTITY;
    

    STORAGE View(
        GET,
        GETBATCH,
        READ IN QUERIES AND PROJECTORS,
        INSERT IN PROJECTORS,
        UPDATE IN PROJECTORS
    ) REQUIRES ENTITY;

    STORAGE WLog(
        GET,
        READ IN QUERIES AND PROJECTORS
    );

    STORAGE PWLog(
        GET,
        READ IN QUERIES AND PROJECTORS
    );

    STORAGE AppSecrets(GET);

    STORAGE Subject(GET IN COMMANDS AND QUERIES);

    STORAGE Http(READ IN QUERIES AND PROJECTORS);

    STORAGE SendMail(INSERT IN PROJECTORS);

    STORAGE CmdResult(INSERT IN COMMANDS);

)