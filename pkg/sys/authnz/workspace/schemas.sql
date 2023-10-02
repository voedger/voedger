-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

-- many, target app, user profile
TABLE ChildWorkspace INHERITS CDoc (
	WSName varchar NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData varchar(1024),
	TemplateName varchar,
	TemplateParams varchar(1024),
	WSClusterID int32 NOT NULL,
	WSID int64,  -- to be updated afterwards
	WSError varchar(1024) -- to be updated afterwards
);

-- target app, (target cluster, base profile WSID)
TABLE WorkspaceID INHERITS CDoc (
	OwnerWSID int64 NOT NULL,
	OwnerQName qname NOT NULL,
	OwnerID int64 NOT NULL,
	OwnerApp varchar NOT NULL,
	WSName varchar NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData varchar(1024),
	TemplateName varchar,
	TemplateParams varchar(1024),
	WSID int64
);

-- target app, new WSID
TABLE WorkspaceDescriptor INHERITS Singleton (
	OwnerWSID int64, -- owner* fields made non-required for app workspaces
	OwnerQName qname,
	OwnerID int64,
	OwnerApp varchar, -- QName -> each target app must know the owner QName -> string
	WSName varchar NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData varchar(1024),
	TemplateName varchar,
	TemplateParams varchar(1024),
	WSID int64,
	CreateError varchar(1024),
	CreatedAtMs int64 NOT NULL,
	InitStartedAtMs int64,
	InitError varchar(1024),
	InitCompletedAtMs int64,
	Status int32
);

