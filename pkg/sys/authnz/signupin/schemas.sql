-- Copyright (c) 2022-present unTill Pro, Ltd.

SCHEMA sys;

-- has no ownerApp field because it is sys/registry always
TABLE Login INHERITS CDoc (
	ProfileCluster int32 NOT NULL,
	PwdHash bytes NOT NULL,
	AppName string NOT NULL,
	SubjectKind int32,
	LoginHash string NOT NULL,
	WSID int64,     -- to be written after workspace init
	WSError string, -- to be written after workspace init
	WSKindInitializationData string NOT NULL
);
