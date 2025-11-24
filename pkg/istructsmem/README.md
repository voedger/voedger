# Events

- `istructsmem.eventType`
  * Represents event builders (IRawEventBuilder and IRawEventBuilder)  and events (IAbstractEvent, IDbEvent and other)

- `istructsmem.appEventsType`
  * Represents IEvents

  * `appEventsType.GetSyncRawEventBuilder`
    + Returns Synced raw event builder

  * `appEventsType.PutPlog`
    + Saves IRawEvent and return IPLogEvent
    + Argument is cloned and saved

  * `appEventsType.PutPlog`
    + Saves IPLogEvent and return IWLogEvent
    + Event entities are cloned and saved

  * `appEventsType.ReadPLog`
    + Reads PLog from specified offset to specified event count by callback function

  * `appEventsType.ReadWLog`
    + Reads WLog from specified offset to specified event count by callback function

# Records
- `istructsmem.recordType`
  * represents IRecord and other records

- `istructsmem.appRecordsType`
  * represents IRecords

  * `appRecordsType.Apply`
    + Saves IPLogEvent CUDs records and returns IRecords by callback function
  * `appRecordsType.Read`
    + Reads record from storage by specified ID and returns IRecord


# Views

- `istructsmem.viewRecordType`
  * represents IKeyBuilder

- `istructsmem.appViewRecordsType`
  * represents IViewRecords

  * `appViewRecordsType.KeyBuilder`
    + Returns new key builder for specified view
  * `appViewRecordsType.NewValueBuilder`
    + Returns new empty view value builder for specified view
  * `appViewRecordsType.UdateValueBuilder`
    + Returns new view value builder for specified view assigned from specified exists view value

  * `appViewRecordsType.Put`
    + Puts view record (key and value) into the storage

  * `appViewRecordsType.Read`
    + Reads view records (key and value) for specified key from the storage by calling callback function.
    + Key may be build partially. In this case all view records, witch keys starts with specified key, will be reads

```mermaid
  erDiagram
  statelessPackage ||..|| sys: "e.g."
  statelessPackage || ..|| IStatelessPkgBuilder: accepts
  statelessPackage ||..|| IStatelessPkgResourcesBuilder: fills
  IStatelessPkgResourcesBuilder ||..|| IStatelessPkgBuilder: "obtained from"
  statelessPackage ||..|| IStatelessPkg: returns
  IStatelessPkg ||..|| IStatelessPkgBuilder: "built from"
  IStatelessPkgBuilder {
    func AddPackage(path)IStatelessPkgResourcesBuilder
    func Build()IStatelessPkg
  }
  VVM ||..|{ IStatelessPkg: "wires as map by path"
  IStatelessPkg ||..|{ AppConfigType: "wired into"
  IStatelessPkg ||..|{ Processor: "provided to"
  IStatelessPkg ||..|| ExtEngine: "provided to"


  Application ||..|| AppConfigType: fills
  Application ||..|| PackageFS: provides
  PackageFS ||..|| "sql files": "is FS with"
  statefullPackage ||..|| AppConfigType: fills

```

```mermaid
  erDiagram
  VVM ||..|{ BuiltinAppBuilder: collects
  BuiltinAppBuilder ||..|| func: is
  BuiltinAppBuilder ||..|| BuiltinAppDef: returns
  BuiltinAppBuilder ||..|| AppConfigType: fills
  AppConfigType ||..|| AppConfigsType: "is part of"
  BuiltinAppDef {
    AppDeploymentDescriptor field
	  AppQName field
	  PackageFS slice
  }
  BuiltinAppDef ||..|| BuiltinApp: "sql parsed into"
  BuiltinApp {
    	AppDeploymentDescriptor field
    	Name field
    	IAppDef field
  }
  BuiltinAppDef ||..|| BuiltInAppPackages: "wired into"
  BuiltinApp ||..|| BuiltInAppPackages: "wired into"
  BuiltInAppPackages {
    BuiltInApp field
	  PackageFS slice
  }
  BuiltInAppPackages ||..|| AppsArtefacts: "wired into"
  AppsArtefacts {
    AppConfigsType field
	  BuiltInAppPackages slice
  }
  AppConfigsType ||..|| AppsArtefacts: "wired into"
  BuiltInAppPackages }|..|| AppsArtefacts: "wired into"
  AppsArtefacts ||..|| AppResources: "wired into"
  AppResources {
    AppConfigsType field
		StatelessPackages slice
  }
  AppConfigsType ||..|| AppResources: "wired into"

```