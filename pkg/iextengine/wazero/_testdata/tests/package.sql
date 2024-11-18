/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

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
    TYPE CommandTestStoragesParam(
        IdToRead ref(Doc1)
    );
    TYPE CommandTestStoragesResult(
        ReadValue int32,
        ReadName varchar,
        ReadToken varchar,
        ReadKind int32,
        ReadWSID int64
    );
    VIEW Results(
        Pk int32,
        Key varchar(50),
        IntVal int32,
        QNameVal qname,
        PRIMARY KEY((Pk), Key)
    ) AS RESULT OF ProjectorTestStorageWLog;
    TABLE Doc1 INHERITS sys.CDoc(
        Value int32
    );
    EXTENSION ENGINE WASM(
        COMMAND dummyCmd();
        COMMAND CmdToTestWlogStorage(cmdToTestWlogStorageParam) RETURNS cmdToTestWlogStorageResult;
        COMMAND CommandTestStorages(CommandTestStoragesParam) RETURNS CommandTestStoragesResult;
        PROJECTOR ProjectorTestStorageWLog AFTER EXECUTE ON CmdToTestWlogStorage INTENTS(sys.View(Results));
        PROJECTOR ProjectorTestStorages AFTER EXECUTE ON CmdToTestWlogStorage INTENTS(sys.SendMail);
    );
)