WORKSPACE TestWorkspace (
    DESCRIPTOR TestWorkspaceDescriptor();
    TYPE cmdToTestWlogStorageParam(
        Offset int64,
        Count int64
    );
    TYPE cmdToTestWlogStorageResult(
        ReadQName qname,
        ReadValues int32
    );
    TYPE CmdToTestRecordStorageParam(
        IdToRead ref(Doc1)
    );
    TYPE CmdToTestRecordStorageResult(
        ReadValue int32
    );
    VIEW Results(
        Pk int32,
        Key varchar(50),
        IntVal int32,
        QNameVal qname,
        PRIMARY KEY((Pk), Key)
    ) AS RESULT OF ProjectorToTestWlogStorage;
    TABLE Doc1 INHERITS CDoc(
        Value int32
    );
    EXTENSION ENGINE WASM(
        COMMAND dummyCmd();
        COMMAND CmdToTestWlogStorage(cmdToTestWlogStorageParam) RETURNS cmdToTestWlogStorageResult;
        COMMAND CmdToTestRecordStorage(CmdToTestRecordStorageParam) RETURNS CmdToTestRecordStorageResult;
        PROJECTOR ProjectorToTestWlogStorage AFTER EXECUTE ON CmdToTestWlogStorage INTENTS(View(Results));
        PROJECTOR ProjectorToTestSendMailStorage AFTER EXECUTE ON CmdToTestWlogStorage INTENTS(SendMail);
    );
)