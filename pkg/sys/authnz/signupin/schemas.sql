-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

SCHEMA sys;

-- cdoc.sys.Login must be known in each target app. "unknown ownerQName scheme cdoc.sys.Login" on c.sys.CreatWorkspaceID otherwise
-- has no ownerApp field because it is sys/registry always
TABLE Login INHERITS CDoc (
	ProfileCluster int32 NOT NULL,
	PwdHash bytes NOT NULL,
	AppName varchar NOT NULL,
	SubjectKind int32,
	LoginHash varchar NOT NULL,
	WSID int64,      -- to be written after workspace init
	WSError varchar, -- to be written after workspace init
	WSKindInitializationData text NOT NULL
);
