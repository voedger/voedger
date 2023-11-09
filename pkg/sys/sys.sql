-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

ABSTRACT TABLE CRecord();

ABSTRACT TABLE WRecord();

ABSTRACT TABLE ORecord();

ABSTRACT TABLE CDoc INHERITS CRecord();

ABSTRACT TABLE ODoc INHERITS ORecord();

ABSTRACT TABLE WDoc INHERITS WRecord();

ABSTRACT TABLE Singleton INHERITS CDoc();

ALTERABLE WORKSPACE AppWorkspaceWS();

ABSTRACT WORKSPACE Workspace (
    TABLE ChildWorkspace INHERITS CDoc (
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSClusterID int32 NOT NULL,
        WSID int64,           -- to be updated afterwards
        WSError varchar(1024) -- to be updated afterwards
    );

    -- target app, (target cluster, base profile WSID)
    TABLE WorkspaceID INHERITS CDoc (
        OwnerWSID int64 NOT NULL,
        OwnerQName qname, -- Deprecated: use OwnerQName2
        OwnerID int64 NOT NULL,
        OwnerApp varchar NOT NULL,
        WSName varchar NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData varchar(1024),
        TemplateName varchar,
        TemplateParams varchar(1024),
        WSID int64,
        OwnerQName2 text
    );

    -- target app, new WSID
    TABLE WorkspaceDescriptor INHERITS Singleton (
        -- owner* fields made non-required for app workspaces
        OwnerWSID int64,
        OwnerQName qname, -- Deprecated: use OwnerQName2
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
        Status int32,
        OwnerQName2 text
    );

    TABLE UserProfile INHERITS Singleton (DisplayName varchar);

    TABLE DeviceProfile INHERITS Singleton ();

    TABLE AppWorkspace INHERITS Singleton ();

    TABLE BLOB INHERITS WDoc (status int32 NOT NULL);

    TABLE Subject INHERITS CDoc (
        Login varchar NOT NULL,
        SubjectKind int32 NOT NULL,
        Roles varchar(1024) NOT NULL,
        ProfileWSID int64 NOT NULL,
        UNIQUEFIELD Login
    );

    TABLE Invite INHERITS CDoc (
        SubjectKind int32,
        Login varchar NOT NULL,
        Email varchar NOT NULL,
        Roles varchar(1024),
        ExpireDatetime int64,
        VerificationCode varchar,
        State int32 NOT NULL,
        Created int64,
        Updated int64 NOT NULL,
        SubjectID ref,
        InviteeProfileWSID int64,
        UNIQUEFIELD Email
    );

    TABLE JoinedWorkspace INHERITS CDoc (
        Roles varchar(1024) NOT NULL,
        InvitingWorkspaceWSID int64 NOT NULL,
        WSName varchar NOT NULL
    );


    TYPE EchoParams (Text text NOT NULL);

    TYPE EchoResult (Res text NOT NULL);

    TYPE RefreshPrincipalTokenResult (
        NewPrincipalToken text NOT NULL
    );

    TYPE EnrichPrincipalTokenParams (
        Login text NOT NULL
    );

    TYPE EnrichPrincipalTokenResult (
        EnrichedToken text NOT NULL
    );

    TYPE GRCountResult (
        NumGoroutines int32 NOT NULL
    );

    TYPE ModulesResult (
        Modules text(32768) NOT NULL
    );

    TYPE RenameQNameParams (
        ExistingQName qname NOT NULL,
        NewQName text NOT NULL
    );

    TYPE CollectionParams (
        Schema text NOT NULL,
        ID int64
    );

    TYPE GetCDocParams (
        ID int64 NOT NULL
    );

    TYPE GetCDocResult (
        Result text(32768) NOT NULL
    );

    TYPE StateParams (
        After int64 NOT NULL
    );

    TYPE StateResult (
        State text(32768) NOT NULL
    );

    TYPE DescribePackageNamesResult (
        Names text NOT NULL
    );

    TYPE DescribePackageParams (
        PackageName text NOT NULL
    );

    TYPE DescribePackageResult (
        PackageDesc text NOT NULL
    );

    TYPE InitiateInvitationByEMailParams (
        Email text NOT NULL,
        Roles text NOT NULL,
        ExpireDatetime int64 NOT NULL,
        EmailTemplate text(32768) NOT NULL,
        EmailSubject text NOT NULL
    );

    TYPE InitiateJoinWorkspaceParams (
        InviteID ref NOT NULL,
        VerificationCode text NOT NULL
    );

    TYPE InitiateUpdateInviteRolesParams (
        InviteID ref NOT NULL,
        Roles text NOT NULL,
        EmailTemplate text(32768) NOT NULL,
        EmailSubject text NOT NULL
    );

    TYPE InitiateCancelAcceptedInviteParams (
        InviteID ref NOT NULL
    );

    TYPE CancelSentInviteParams (
        InviteID ref NOT NULL
    );

    TYPE CreateJoinedWorkspaceParams (
        Roles text NOT NULL,
        InvitingWorkspaceWSID int64 NOT NULL,
        WSName text NOT NULL
    );

    TYPE UpdateJoinedWorkspaceRolesParams (
        Roles text NOT NULL,
        InvitingWorkspaceWSID int64 NOT NULL
    );

    TYPE DeactivateJoinedWorkspaceParams (
        InvitingWorkspaceWSID int64 NOT NULL
    );

    TYPE JournalParams (
        From int64 NOT NULL,
        Till int64 NOT NULL,
        EventTypes text NOT NULL,
        IndexForTimestamps text,
        RangeUnit text
    );

    TYPE JournalResult (
        Offset int64 NOT NULL,
        EventTime int64 NOT NULL,
        Event text NOT NULL
    );

    TYPE SqlQueryParams (
        Query text NOT NULL
    );

    TYPE SqlQueryResult (
        Result text NOT NULL
    );

    TYPE InitiateEmailVerificationParams (
        Entity text NOT NULL, -- must be string, not QName, because target app could not know that QName. E.g. unknown QName «registry.ResetPasswordByEmailUnloggedParams»: name not found
        Field text NOT NULL,
        Email text NOT NULL,
        TargetWSID int64 NOT NULL,
        ForRegistry bool, -- to issue token for sys/registry/pseudoWSID/c.sys.ResetPassword, not for the current app
        Language text
    );

    TYPE InitialEmailVerificationResult (
        VerificationToken text(32768) NOT NULL
    );

    TYPE IssueVerifiedValueTokenParams (
        VerificationToken text(32768) NOT NULL,
        VerificationCode text NOT NULL,
        ForRegistry bool
    );

    TYPE IssueVerifiedValueTokenResult (
        VerifiedValueToken text NOT NULL
    );

    TYPE SendEmailVerificationCodeParams (
        VerificationCode text NOT NULL,
        Email text NOT NULL,
        Reason text NOT NULL,
        Language text
    );

    TYPE InitChildWorkspaceParams (
        WSName text NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData text,
        WSClusterID int32 NOT NULL,
        TemplateName text,
        TemplateParams text
    );

    TYPE CreateWorkspaceIDParams (
        OwnerWSID int64 NOT NULL,
        OwnerQName qname, -- Deprecated: use OwnerQName2
        OwnerID int64 NOT NULL,
        OwnerApp text NOT NULL,
        WSName text NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData text,
        TemplateName text,
        TemplateParams text,
        OwnerQName2 text
    );

    TYPE CreateWorkspaceParams (
        OwnerWSID int64 NOT NULL,
        OwnerQName qname, -- Deprecated: use OwnerQName2
        OwnerID int64 NOT NULL,
        OwnerApp text NOT NULL,
        WSName text NOT NULL,
        WSKind qname NOT NULL,
        WSKindInitializationData text,
        TemplateName text,
        TemplateParams text,
        OwnerQName2 text
    );

    TYPE OnWorkspaceDeactivatedParams (
        OwnerWSID int64 NOT NULL,
        WSName text NOT NULL
    );

    TYPE OnJoinedWorkspaceDeactivatedParams (
        InvitedToWSID int64 NOT NULL
    );

    TYPE OnChildWorkspaceDeactivatedParams (
        OwnedID int64 NOT NULL
    );

    TYPE QueryChildWorkspaceByNameParams (
        WSName text NOT NULL
    );

    TYPE QueryChildWorkspaceByNameResult (
        WSName text NOT NULL,
        WSKind text NOT NULL,
        WSKindInitializationData text NOT NULL,
        TemplateName text NOT NULL,
        TemplateParams text,
        WSID int64,
        WSError text
    );

    EXTENSION ENGINE BUILTIN (
        QUERY Echo(EchoParams) RETURNS EchoResult;
        QUERY GRCount RETURNS GRCountResult;
        QUERY Modules RETURNS ModulesResult;
        COMMAND RenameQName(RenameQNameParams);

        QUERY RefreshPrincipalToken RETURNS RefreshPrincipalTokenResult;
        QUERY EnrichPrincipalToken(EnrichPrincipalTokenParams) RETURNS EnrichPrincipalTokenResult;

        -- QUERY Collection(CollectionParams) RETURNS ANY;
        QUERY GetCDoc(GetCDocParams) RETURNS GetCDocResult;
        QUERY State(StateParams) RETURNS StateResult;

        QUERY DescribePackageNames RETURNS DescribePackageNamesResult;
        QUERY DescribePackage(DescribePackageParams) RETURNS DescribePackageResult;

        QUERY Journal(JournalParams) RETURNS JournalResult;

        QUERY SqlQuery(SqlQueryParams) RETURNS SqlQueryResult;

        QUERY IssueVerifiedValueToken(IssueVerifiedValueTokenParams) RETURNS IssueVerifiedValueTokenResult;

        COMMAND UploadBLOBHelper;
        COMMAND DownloadBLOBHelper;

        COMMAND InitiateInvitationByEMail(InitiateInvitationByEMailParams);
        COMMAND InitiateJoinWorkspace(InitiateJoinWorkspaceParams);
        COMMAND InitiateUpdateInviteRoles(InitiateUpdateInviteRolesParams);
        COMMAND InitiateCancelAcceptedInvite(InitiateCancelAcceptedInviteParams);
        COMMAND InitiateLeaveWorkspace;
        COMMAND CancelSentInvite(CancelSentInviteParams);
        COMMAND CreateJoinedWorkspace(CreateJoinedWorkspaceParams);
        COMMAND UpdateJoinedWorkspaceRoles(UpdateJoinedWorkspaceRolesParams);
        COMMAND DeactivateJoinedWorkspace(DeactivateJoinedWorkspaceParams);
        QUERY QueryChildWorkspaceByName(QueryChildWorkspaceByNameParams) RETURNS QueryChildWorkspaceByNameResult;

        QUERY InitiateEmailVerification(InitiateEmailVerificationParams) RETURNS InitialEmailVerificationResult;
        COMMAND SendEmailVerificationCode(SendEmailVerificationCodeParams);

        COMMAND InitChildWorkspace(InitChildWorkspaceParams);
        COMMAND CreateWorkspaceID(CreateWorkspaceIDParams);
        COMMAND CreateWorkspace(CreateWorkspaceParams);
        COMMAND OnWorkspaceDeactivated(OnWorkspaceDeactivatedParams);
        COMMAND OnJoinedWorkspaceDeactivated(OnJoinedWorkspaceDeactivatedParams);
        COMMAND OnChildWorkspaceDeactivated(OnChildWorkspaceDeactivatedParams);

        SYNC PROJECTOR RecordsRegistryProjector ON (CRecord, WRecord, ORecord) INTENTS(View(RecordsRegistry));
    );

    VIEW RecordsRegistry (
        IDHi int64 NOT NULL,
        ID ref NOT NULL,
        WLogOffset int64 NOT NULL,
        QName qname NOT NULL,
        PRIMARY KEY ((IDHi), ID)
    ) AS RESULT OF sys.RecordsRegistryProjector;
);

EXTENSION ENGINE BUILTIN (
    STORAGE Record(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
        GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
        INSERT SCOPE(COMMANDS),
        UPDATE SCOPE(COMMANDS)
    ) ENTITY RECORD;

    -- used to validate projector state/intents declaration
    STORAGE View(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
        GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ SCOPE(QUERIES, PROJECTORS),
        INSERT SCOPE(PROJECTORS),
        UPDATE SCOPE(PROJECTORS)
    ) ENTITY VIEW;

    STORAGE WLog(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE PLog(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
        READ SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE AppSecret(
        GET SCOPE(COMMANDS, QUERIES, PROJECTORS)
    );

    STORAGE RequestSubject(
        GET SCOPE(COMMANDS, QUERIES)
    );

    STORAGE Http(
        READ SCOPE(QUERIES, PROJECTORS)
    );

    STORAGE SendMail(
        INSERT SCOPE(PROJECTORS)
    );

    STORAGE Result(
        INSERT SCOPE(COMMANDS)
    );
);
