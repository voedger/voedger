### Story
As a Heeus Architect I want to have a description where and how the data is stored in the Federation

### Solution Principles
- current solution is actual for Heeus CE and SE only
- Storage could be:
  - `AppPartitionStorage`
    - cached
    - created\released per each app partition deploy\undeploy
    - used by processors, actualizers
  - `AppStorages`
    - uncached
    - created\released per app deploy\undeploy
    - used by router to manage BLOBs and certificates
- storage driver name and its params are configured via cmd line
- cache principles
  - select by an exact id -> cached always
  - `IAppStorage.Get()` and `.GetBatch()` cached, `.Read()` - not

### Concepts
```mermaid
erDiagram
Federation ||..|{ Cluster : has
Cluster ||..|{ Application : has
Cluster ||..|| ClusterStorage : has
Application ||..|{ app_AppPartition : has
app_AppPartition ||..|| PLogPartition : uses
app_AppPartition ||..|{ Workspace : uses
ClusterStorage ||..|{ AppStorage : "has Small Object Storage"
ClusterStorage ||..|| BLOBStorage : has
BLOBStorage ||..|| AmazonS3 : like
BLOBStorage ||..|{ BLOB : has
BLOBStorage ||..|| AppStorage : "can use"
BLOB ||..|| Workspace : "linked to"
AppStorage ||..|| PLog : has
PLog ||..|{ PLogPartition: has
PLogPartition ||..|{ Actualizer : "used e.g. by"
AppStorage ||..|{ Workspace : has
Workspace ||..|| WLog : has
Workspace ||..o{ View : has
Workspace ||..o{ Table : has
WLog ||..|{ QueryProcessor : "used e.g. by"
```

### Components
#### Heeus CE and SE executables
```mermaid
erDiagram
ce_exe ||..|| ce_appStorageProviderFactory : provides
se_exe ||..|| se_appStorageProviderFactory : provides
ce_exe ||..|| ce_cmdLine : reads
se_exe ||..|| se_cmdLine : reads
ce_appStorageProviderFactory ||..|| ce-se_CLI : "passed to"
se_appStorageProviderFactory ||..|| ce-se_CLI : "passed to"
ce_cmdLine ||..|| ce-se_CLI : "passed to"
se_cmdLine ||..|| ce-se_CLI : "passed to"

ce-se_CLI ||..|| imetrics_Provide : "calls"
imetrics_Provide ||..|| imetrics_IMetrics : "provides"

ce-se_CLI ||..|| params : "parses cmd line and provides *struct"
params ||..|| storageCacheSize : has
params ||..|| vvmName : has
params ||..|| driverParams : has
params ||..|| appStorageDriverName : has
os_HostName ||..|| vvmName : "is default for"

driverParams ||..|| appStorageProviderFactory : "used by"
driverParams ||..|| appStorageProviderFactory : "used by"
appStorageDriverName ||..|| appStorageProviderFactory : "used by"
ce-se_CLI ||..|| appStorageProviderFactory : receives

appStorageProviderFactory ||..|| ce-se_provideIAppStorageProvider : "passed to and called by"
appStorageDriverName ||..|| ce-se_provideIAppStorageProvider : "passed to"
appStorageDriverName ||..|| appStorageProviderFactory : "used by"
driverParams ||..|| ce-se_provideIAppStorageProvider : "passed to"
ce-se_provideIAppStorageProvider ||..|| istorage_IAppStorageProvider : returns
istorage_IAppStorageProvider ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
imetrics_IMetrics ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
storageCacheSize ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
vvmName ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
storageCacheSize ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
istorage_IAppStorageProvider ||..|| ce-se_NewAppPartitionStorageFactory : "passed to"
ce-se_NewAppPartitionStorageFactory ||..|| ce-se_AppPartitionStorages : "returns *struct"
ce-se_AppPartitionStorages ||..|| ce-se_AppPartitionStorageFactory : "has func()"
ce-se_AppPartitionStorages ||..|| ce-se_AppStorages : "has *struct"
ce-se_AppPartitionStorageFactory ||..|| ce-se_AppPartitionStorage : returns
ce-se_AppPartitionStorage ||..|| istorage_IAppStorage : "is cached"
ce-se_AppStorages ||..|| istorage_IAppStorage : "has method to return uncached"
```
#### ce-se package
##### AppStorages
```mermaid
erDiagram
ce-se_NewAppPartitionStorageFactory ||..|| ce-se_AppPartitionStorages : "returns *struct"
ce-se_AppPartitionStorages ||..|| ce-se_AppPartitionStorageFactory : has
ce-se_AppPartitionStorages ||..|| ce-se_AppStorages : has
ce-se_AppStorages ||..|| ce-se_provideBLOBStorage : "used by"
ce-se_AppStorages ||..|| ce-se_provideRouterStorage : "used by"
ce-se_provideBLOBStorage ||..|| iblobstoragestg_IBlobberAppStorage : "creates IAppStorage for sys/blobber app"
iblobstoragestg_IBlobberAppStorage ||..|| istorage_IAppStorage : is
ce-se_provideRouterStorage ||..|| dbcertcache_IRouterAppStorage : "creates IAppStorage for sys/router app"
dbcertcache_IRouterAppStorage ||..|| istorage_IAppStorage : is
iblobstoragestg_IBlobberAppStorage ||..|| iblobstoragestg_Provide : "used by"
iblobstoragestg_Provide ||..|| iblobstorage_IBLOBStorage : returns

dbcertcache_IRouterAppStorage ||..|| dbcertcache_New : "used by"
dbcertcache_New ||..|| autocert_Cache : "returns"
autocert_Cache ||..|| ihttpimpl_httpProcessor : "used for https by"

iblobstorage_IBLOBStorage ||..|| ihttpimpl_router : "used by"
ihttpimpl_router ||..|| ihttpimpl_httpProcessor : "contained by"

iblobstorage_IBLOBStorage{
  func WriteBLOB
  func ReadBLOB
  func QueryBLOBState
}
```
##### AppPartitionStorage
```mermaid
erDiagram
ce-se_NewAppPartitionStorageFactory ||..|| istoragecache_New : "uses"
istoragecache_New ||..|| ce-se_AppPartitionStorageFactory : "used to create"

ce-se_NewAppPartitionStorageFactory ||..|| ce-se_AppPartitionStorages : "returns *struct"
ce-se_AppPartitionStorages ||..|| ce-se_AppPartitionStorageFactory : has
ce-se_AppPartitionStorages ||..|| ce-se_AppStorages : has

ce-se_AppPartitionStorageFactory ||..|| ce-se_AppPartitionFactory : "used by"
ce-se_AppPartitionFactory ||..|| app_AppPartition : "to build *struct"
app_AppPartition ||..|| AppPartitionStorage : has
AppPartitionStorage ||..|{ QueryProcessor : "used by"
AppPartitionStorage ||..|| CommandProcessor : "used by"
AppPartitionStorage ||..|{ Actualizer : "used by"
AppPartitionStorage ||..|| istorage_IAppStorage : is
```
