# Create Workspace v2

Create User/Login/App Workspaces

## Motivation

- As a system architect I want to redesign Workspaces in particular I want to get rid of "pseudo workspaces"
  - Currently there are a lot of "pseudo workspaces" (2^16) and this makes hard bulding list of logins
- launchpad: [Create Workspace](https://dev.heeus.io/launchpad/#!17898)
- launchpad: [Create Workspaces v2](https://dev.heeus.io/launchpad/#!21010)

## Content

- [Principles](#solution-principles)
- [Create Login](#create-login)
- [Create ChildWorkspace](#create-childworkspace)
- [Related commits](#related-commits)
- [Notes](#notes)
- [See also](#see-also)

Commands:

- [c.sys.InitChildWorkspace()](#csysinitchildworkspace)
- [c.sys.CreateWorkspaceID()](#csyscreateworkspaceid)
- [c.sys.CreateWorkspace()](#csyscreateworkspace)
- [UpdateOwner()](#updateownerwsparams-newwsid-wsdescriniterror)

Projectors:

- [aproj.sys.InvokeCreateWorkspaceID()](#aprojsysinvokecreateworkspaceid)
- [aproj.sys.InvokeCreateWorkspace()](#aprojsysinvokecreateworkspace)
- [aproj.sys.InitiateWorkspace()](#aprojsysinitiateworkspace)


## Principles

- AppWorkspaces are created by system when an App is deployed to a Cluster
- Workspaces of other kinds must be explicitely created (and initialized)
    - It is not possible to work with uninitialized workspaces
- Client calls `c.registry.CreateLogin` using pseudo WS calculated as (main cluster, crc16(login))
- If router sees that baseWSID of WSID is < MaxPseudoBaseWSID then it replaces that pseudo base WSID with app base WSID:
  - (main cluser, (baseWSID %% numAppWorkspaces) + FirstBaseAppWSID)
- `crc16 = crc32.ChecksumIEEE & (MaxUint32 >> 16)`
- `cdoc.registry.Login` stores login hash only

## Create Login

// FIXME: cdoc.sys.WorkspaceID is a large collection, must be wdoc.sys.WorkspaceID

|entity|app|ws|cluster
|---|---|---|---|
|c.registry.CreateLogin()|sys/registry|pseudoWS|main
|cdoc.registry.Login (owner)<br/>aproj.sys.InvokeCreateWorkspaceID|sys/registry|app ws|main
|c.sys.CreateWorkspaceID()<br/>cdoc.sys.WorkspaceID<br/>aproj.sys.InvokeCreateWorkspace()|Target App|(Target Cluster, base App WSID)|Target Cluster
|c.sys.CreateWorkspace()<br/>cdoc.sys.WorkspaceDescriptor<br/>cdoc.sys.UserProfile/DeviceProfile<br/>aproj.sys.InitializeWorkspace()|Target App|new WSID|Target Cluster

## Create ChildWorkspace

|entity|app|ws|cluster|
|---|---|---|---|
|c.sys.InitChildWorkspace()<br/>cdoc.sys.ChildWorkspace (owner)<br/>aproj.sys.InvokeCreateWorkspaceID()|Tagret App|Profile|Profile Cluster
|c.sys.CreateWorkspaceID()<br/>cdoc.sys.WorkspaceID<br/>aproj.sys.InvokeCreateWorkspace()|Target App|(Target Cluster, CRC16(ownerWSID+"/"+wsName))|Target Cluster
|c.sys.CreateWorkspace()<br/>cdoc.sys.WorkspaceDescriptor<br/>cdoc.$wsKind (air.Restaurant)<br/>aproj.sys.InitializeWorkspace()|Target App|new WSID|Target Cluster

## c.sys.InitChildWorkspace()


- AuthZ: Owner
  - PrincipalToken in header
  - PrincipalToken.ProfileWSID == Request.WSID
- Params: wsName, wsKind, wsKindInitializationData, templateName, templateParams (JSON), wsClusterID
- Check that wsName does not exist yet: View<ChildWorkspaceIdx>{pk: dummy, cc: wsName, value: idOfChildWorkspace}
  - 409 conflict
- Create CDoc<ChildWorkspace> {wsName, wsKind, wsKindInitializationData, templateName, templateParams, wsClusterID, /* Updated aftewards by UpdateOwner*/ WSID, wsError}
  - Trigger Projector<A, InvokeCreateWorkspaceID>
  - Trigger Projector<ChildWorkspaceIdx>

Subject:
- Call WS[Subject.ProfileWSID].InitChildWorkspace()
  - wsName
  - wsKind
  - wsKindInitializationData // JSON
  - wsClusterID // Ideally user should be asked which cluster to use
- Call WS[Subject.ProfileWSID].q.QueryChildWorkspaceByName() until (WSID || wsError)
  - Returns all fields of CDoc<ChildWorkspace>

## aproj.sys.InvokeCreateWorkspaceID()

- Triggered by CDoc<ChildWorkspace>
- PseudoWSID = NewWSID(wsClusterID, CRC32(wsName))
- // PseudoWSID  is needed to avoid WSID generation bottlenecks
- Call WS[$PseudoWSID].c.CreateWorkspaceID()

## c.sys.CreateWorkspaceID()

- AuthZ: System
  - SystemToken in header
- Params: (`wsParams`) ownerWSID, ownerQName2, ownerID, wsName, wsKind, wsKindInitializationData, templateName, templateParams (JSON), wsClusterID
  - ownerWSID
    - For profiles: PseudoWSID is calculated (from login) by client as NewWSID(1, ISO CRC32(login))
    - For subject workspaces: ProfileWSID
- Check that ownerWSID + wsName does not exist yet: View<WorkspaceIDIdx> to deduplication
  - pk: ownerWSID
  - cc: wsName
  - val: WSID
- Get new WSID from View[NextBaseWSID]
- Create WDoc[WorkspaceID]{wsParams, WSID: $NewWSID}
  - Triggers Projector[A, InvokeCreateWorkspace]
  - Triggers Projector[WorkspaceIDIdx]


## aproj.sys.InvokeCreateWorkspace()

- Triggered by WDoc<WorkspaceID>
- WS[new.WSID].c.CreateWorkspace()

## c.sys.CreateWorkspace()

- AuthZ: System
  - SystemToken in header

- Params: wsParams, WSID
- Check that CDoc<sys.WorkspaceDescriptor> does not exist yet (IRecords.GetSingleton())
  - return ok otherwise
- if wsKindInitializationData is not valid
  - error = "Invalid workspaced descriptor data: ???"
- Create CDoc<sys.WorkspaceDescriptor>{wsParams, WSID, createError: error, createdAtMs int64,  /* Updated aftewards */ initStartedAtMs int64, initError, initCompletedAtMs int64}
  - Trigger Projector<A, InitializeWorkspace>
- if not error
  - Create CDoc<wsKind>{wsKindInitializationData}

## aproj.sys.InitializeWorkspace()

- // error handling: just return
- // Triggered by  CDoc<sys.WorkspaceDescriptor>

- If updated return // We do NOT react on update since we update record from projector
- If len(new.createError) > 0
  - UpdateOwner(wsParams, new.WSID, new.createError)
  - return

- // Must exist
- wsDescr = GetSingleton(CDoc<sys.WorkspaceDescriptor>)

- if wsDecr.initStartedAtMs == 0
  - WS[currentWS].c.sys.CUD(wsDescr.ID, initStartedAtMs)
  - err = workspace.buildWorkspace() // to init data
  - if err != nil: error = ("Workspace data initialization failed: v", err)
  - WS[currentWS].c.sys.CUD(wsDescr.ID, initError: error, initCompletedAtMs)
  - UpdateOwner(wsParams, new.WSID, error)
  - return
- else if wsDecr.initCompletedAtMs == 0
  - error = "Workspace data initialization was interrupted"
  - WS[currentWS].c.sys.CUD(wsDescr.ID, initError: error, initCompletedAtMs)
  - UpdateOwner(wsParams, new.WSID, error)
  - return
- else // initCompletedAtMs > 0
  - UpdateOwner(wsParams, new.WSID, wsDescr.initError)
  - return

## UpdateOwner(wsParams, new.WSID, wsDescr.initError)

- WS[wsParams.ownerWSID].c.sys.CUD(wsParams.ownerID, WSID, wsError)

## Related commits

- https://github.com/untillpro/airs-bp3/commit/97e00bec13020357215a39d1587aa30441d6231a
- https://github.com/voedger/voedger/pkg/router/commit/d31bbd2c8740183a7cbf0020395e6eb9ebdad641
- https://github.com/heeus/core/commit/05b23292969b0fd78e367d77ad9d8310ce01e7d7

## Notes

- Unable to work at AppWS because it is located in sys/registry app whereas we are logged in a target app. "token issued for another application" error will be result
- Workspace initialized check is made in command processor only. Query processor just returns empty result because there is no data in non-inited workspace
  - `c.sys.CreateWorkspace` or `c.sys.CreateWorkspaceID` or (`c.sys.CUD` + System Principal) -> ok
  - `cdoc.WorkspaceDescriptor` exists ->
    - `.initCompletedAtMs` > 0 && len(`.initError`) == 0 -> ok
    - `c.sys.CUD` -> will check after `parseCUDs` stage if we are updating `cdoc.WorkspaceDescriptor` or `WDoc<BLOB>` now
      - there is update only of (`cdoc.WorkspaceDescriptor` or `WDoc<BLOB>`) only among CUDs -> ok
  - -> 403 forbidden + `workspace is not initialized`
- `aproj.sys.InitializeWorkspace`: `wsKind` == `registry.AppWorkspace` -> self-initialized already, skip further work
- App Workspace has `cdoc.WorkspaceDescriptor` only, there is no `cdoc.$wsKind`
- `cdoc.WorkspaceDescriptor`, `cdoc.WorkspaceID`, `c.sys.CeateWorkspace`, `c.sys.CreateWorkspaceID`:
  - `ownerID`, `ownerQName2`, `ownerWSID` fields are made non-required becuase they are empty in App Workspace
  - `ownerApp` field added to know in which app to update the owner
- AppWorkspaces are [initialized automatically](https://github.com/untillpro/airs-bp3/blob/21010-AD-Workspace-ER/hvm/provide.go#L53) after wiring the VVM before launch
  - for each app
    - PLog and WLog offsets are starting from `istructs.FirstOffset`
    - for each App WS Number
      - AppWSID = (mainClusterID, wsNum + FirstBaseAppWSID)
      - `cdoc.WorkspaceDescriptor` exists already at AppWSID -> skip
      - generate new Sync Raw Event
      - add CUD create `cdoc.WorkspaceDescriptor` to the Event
      - put PLog
      - apply PLog event to records
      - put WLog
      - incr PLog and WLog offsets
  - //TODO AppWorkspaces must be created when application is deployed
- App Workspaces amount is defined per app, default 10

## See also

- Originated from [Create Workspace v2](https://dev.heeus.io/launchpad/#!21010)
