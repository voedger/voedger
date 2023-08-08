-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

SCHEMA sys;

TABLE Subject INHERITS CDoc (
	Login text NOT NULL,
	SubjectKind int32 NOT NULL,
	Roles text NOT NULL,
	ProfileWSID int64 NOT NULL,
	UNIQUEFIELD Login
);

TABLE Invite INHERITS CDoc (
	SubjectKind int32,
	Login text NOT NULL,
	Email text NOT NULL,
	Roles text,
	ExpireDatetime int64,
	VerificationCode text,
	State int32 NOT NULL,
	Created int64,
	Updated int64 NOT NULL,
	SubjectID ref,
	InviteeProfileWSID int64,
	UNIQUEFIELD Email
);

TABLE JoinedWorkspace INHERITS CDoc (
	Roles text NOT NULL,
	InvitingWorkspaceWSID int64 NOT NULL,
	WSName text NOT NULL
);
