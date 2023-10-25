-- Copyright (c) 2020-present unTill Software Development Group B.V.
-- @author Denis Gribanov

TABLE Login INHERITS CDoc (
	ProfileCluster int32 NOT NULL,
	PwdHash bytes NOT NULL,
	AppName varchar NOT NULL,
	SubjectKind int32,
	LoginHash varchar NOT NULL,
	WSID int64,      -- to be written after workspace init
	WSError varchar(1024), -- to be written after workspace init
	WSKindInitializationData varchar(1024) NOT NULL
);