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
    VIEW Results(
        Pk int32,
        Key varchar(50),
        IntVal int32,
        QNameVal qname,
        PRIMARY KEY((Pk), Key)
    ) AS RESULT OF ProjectorToTestWlogStorage;
    EXTENSION ENGINE WASM(
        COMMAND dummyCmd();
        COMMAND CmdToTestWlogStorage(cmdToTestWlogStorageParam) RETURNS cmdToTestWlogStorageResult;
        PROJECTOR ProjectorToTestWlogStorage AFTER EXECUTE ON CmdToTestWlogStorage INTENTS(View(Results));
    );
)