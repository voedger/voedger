-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov
SCHEMA sys;

-- many, target app, user profile
TABLE ChildWorkspace INHERITS CDoc (
	WSName text NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData text,
	TemplateName text,
	TemplateParams text,
	WSClusterID int32 NOT NULL,
	WSID int64,  -- to be updated afterwards
	WSError text -- to be updated afterwards
);

-- target app, (target cluster, base profile WSID)
TABLE WorkspaceID INHERITS CDoc (
	OwnerWSID int64 NOT NULL,
	OwnerQName qname NOT NULL,
	OwnerID int64 NOT NULL,
	OwnerApp text NOT NULL,
	WSName text NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData text,
	TemplateName text,
	TemplateParams text,
	WSID int64
);

-- target app, new WSID
TABLE WorkspaceDescriptor INHERITS Singleton (
	OwnerWSID int64, -- owner* fields made non-required for app workspaces
	OwnerQName qname,
	OwnerID int64,
	OwnerApp text, -- QName -> each target app must know the owner QName -> string
	WSName text NOT NULL,
	WSKind qname NOT NULL,
	WSKindInitializationData text,
	TemplateName text,
	TemplateParams text,
	WSID int64,
	CreateError text,
	CreatedAtMs int64 NOT NULL,
	InitStartedAtMs int64,
	InitError text,
	InitCompletedAtMs int64,
	Status int32
);

