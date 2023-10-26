-- Copyright (c) 2021-present unTill Pro, Ltd.
-- @author Denis Gribanov
TABLE Login INHERITS CDoc (
	ProfileCluster int32 NOT NULL,
	PwdHash bytes NOT NULL,
	AppName varchar NOT NULL,
	SubjectKind int32,
	LoginHash varchar NOT NULL,
	WSID int64,
	-- to be written after workspace init
	WSError varchar(1024),
	-- to be written after workspace init
	WSKindInitializationData varchar(1024) NOT NULL
);