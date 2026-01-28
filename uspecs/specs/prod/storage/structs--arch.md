# Context subsystem architecture: Structured storage

## Overview

Structured storage is Voedger's abstraction layer for persisting and retrieving structured data. It provides a unified interface for storing records, views, and event logs (PLog/WLog) across different storage backends (Cassandra, ScyllaDB, BoltDB, in-memory, DynamoDB).

## Architecture

### Component hierarchy

```text
Application Layer
    |
    v
IAppStructs (pkg/istructs)
    |
    +-- IRecords (Records storage)
    +-- IViewRecords (View records storage)
    +-- IEvents (PLog/WLog storage)
    |
    v
IAppStorage (pkg/istorage)
    |
    +-- IAppStorageProvider (caching, lifecycle)
    +-- IAppStorageFactory (driver-specific)
    |
    v
Storage Backends
    +-- Cassandra/ScyllaDB (cas)
    +-- BoltDB (bbolt)
    +-- In-Memory (mem)
    +-- DynamoDB (amazondb)
```

---

## System views: PL/CC layouts

Information in structured storage is organized into **views**. Each view has a specific layout for its **Partition Key (PK)** and **Clustering Columns (CC)**.

```text
pkg\istructsmem\internal\consts\qnames.go

ID    Name                Purpose
16    SysView_Versions    System view versions
17    SysView_QNames      Application QNames mapping
18    SysView_Containers  Application container names
19    SysView_Records     Application Records (CDoc, WDoc, etc.)
20    SysView_PLog        Partition Event Log
21    SysView_WLog        Workspace Event Log
22    SysView_SingletonIDs Application singletons IDs
```

### SysView_Versions (16)

- PK: `[uint16: 16]`
- CC: `[uint16: VersionKey]`

### SysView_QNames (17)

- PK: `[uint16: 17][uint16: Version]`
- CC: `[string: QName]`

### SysView_Containers (18)

- PK: `[uint16: 18][uint16: Version]`
- CC: `[string: ContainerName]`

### SysView_Records (19) - CDoc/WDoc/CRecord/WRecord

- PK: `[uint16: 19][uint64: WSID][uint64: RecordID_Hi]` (18 bytes)
- CC: `[uint16: RecordID_Lo]` (2 bytes)

### SysView_PLog (20)

- PK: `[uint16: 20][uint16: PartitionID][uint64: Offset_Hi]` (12 bytes)
- CC: `[uint16: Offset_Lo]` (2 bytes)

### SysView_WLog (21)

- PK: `[uint16: 21][uint64: WSID][uint64: Offset_Hi]` (18 bytes)
- CC: `[uint16: Offset_Lo]` (2 bytes)

### SysView_SingletonIDs (22)

- PK: `[uint16: 22][uint16: Version]`
- CC: `[string: QName]`

### User Views

- PK: `[uint16: ViewQNameID][uint64: WSID][...user PK fields]`
- CC: `[...user CC fields]`

---

## QNameID

**QName (Qualified Name)** is a two-part identifier: `<package>.<entity>`

- Examples: `"air.Restaurant"`, `"sys.CDoc"`, `"myapp.Order"`

QNames are are mapped to **QNameID** (uint16):

```text
Type:        QNameID = uint16
Size:        2 bytes
Encoding:    BigEndian
Range:       0 - 65535 (0xFFFF)
```

### QNameID ranges

```text
Range           Purpose                      Examples
0               NullQNameID                  (null/empty)
1-5             System QNames (hardcoded)    Error, CUD, etc.
6-255           Reserved (QNameIDSysLast)    System use
256-65535       User-defined QNames          Application types
```

### Hardcoded system QNameIDs

```text
pkg/istructs/consts.go

0    NullQNameID
1    QNameIDForError
2    QNameIDCommandCUD
3    QNameIDForCorruptedData
4    QNameIDWLogOffsetSequence
5    QNameIDRecordIDSequence
...
255  QNameIDSysLast (boundary)
```

### QName mapping storage

The QName <-> QNameID mapping is stored in SysView_QNames (ID=17):

```text
Partition Key:
[0-1]    uint16   SysView_QNames (constant = 17)
[2-3]    uint16   Version (ver01)

Clustering Columns:
[0...]   string   QName as string (e.g., "myapp.Order")

Value:
[0-1]    uint16   QNameID (e.g., 1000)
```

This mapping is:

- Loaded at application startup
- Cached in memory (pkg/istructsmem/internal/qnames)
- Persisted to storage when new QNames are added
- Immutable once assigned (QNameID never changes for a given QName)

---

## System views

### SysView_Versions

SysView_Versions (ID=16) is a system view that tracks the schema version of other system views in Voedger's storage layer. It acts as a version registry for internal data structures, enabling schema evolution and backward compatibility.

#### Data structure

**Partition key:**

```text
[0-1]    uint16   SysView_Versions (constant = 16)
```

**Clustering columns:**

```text
[0-1]    uint16   VersionKey (identifies which system view)
```

**Value:**

```text
[0-1]    uint16   VersionValue (the version number)
```

#### Version keys

The system tracks versions for these internal views:

```text
VersionKey    System view           Purpose
1             SysQNamesVersion      QName to QNameID mapping format
2             SysContainersVersion  Container names format
3             SysSingletonsVersion  Singleton IDs format
4             SysUniquesVersion     Uniques format (deprecated)
```

#### How it works

**On application startup:**

```go
// Load all version information from storage
versions := vers.New()
err := versions.Prepare(storage)

// This reads all entries from SysView_Versions into memory
// Partition Key: [16]
// Reads all clustering columns and values
```

**When loading a system view (e.g., QNames):**

```go
// pkg/istructsmem/internal/qnames/impl.go

// Check what version of QNames view is stored
ver := versions.Get(vers.SysQNamesVersion)

switch ver {
case vers.UnknownVersion:
    // No data exists yet - first time initialization
    return nil
case ver01:
    // Load using version 1 format
    return names.load01(storage)
default:
    // Unknown version - error
    return ErrorInvalidVersion
}
```

**When storing a system view:**

```go
// pkg/istructsmem/internal/qnames/impl.go

// After storing QNames data, update the version
if ver := versions.Get(vers.SysQNamesVersion); ver != latestVersion {
    err = versions.Put(vers.SysQNamesVersion, latestVersion)
}

// This writes to SysView_Versions:
// Partition Key: [16]
// Clustering Columns: [1] (SysQNamesVersion)
// Value: [latestVersion]
```

---

### SysView_QNames

SysView_QNames (ID=17) is a system view that stores the mapping between QNames (qualified names like `"myapp.Order"`) and their corresponding QNameIDs (uint16 numeric identifiers). This mapping enables compact storage by using 2-byte integers instead of variable-length strings in storage keys.

#### QNames data structure

**Partition key (4 bytes total):**

```text
[0-1]    uint16   SysView_QNames (constant = 17)
[2-3]    uint16   Version (ver01 = 1)
```

**Clustering columns (variable length):**

```text
[0...]   string   QName as string (e.g., "myapp.Order")
```

**Value (2 bytes):**

```text
[0-1]    uint16   QNameID (e.g., 1000)
```

#### QNames lifecycle

**On application startup:**

```go
// 1. Load versions
versions := vers.New()
err := versions.Prepare(storage)

// 2. Load existing QName mappings from storage
qnames := qnames.New()
err = qnames.Prepare(storage, versions, appDef)

// This process:
// - Reads all QName -> QNameID mappings from SysView_QNames
// - Collects all QNames from application definition (appDef)
// - Assigns new QNameIDs to any new QNames
// - Writes new mappings back to storage if changes detected
```

**Loading from storage (version 1 format):**

```go
func (names *QNames) load01(storage istorage.IAppStorage) error {
    readQName := func(cCols, value []byte) error {
        // Parse QName from clustering columns
        qName, err := appdef.ParseQName(string(cCols))
        if err != nil {
            return err
        }

        // Read QNameID from value
        id := binary.BigEndian.Uint16(value)

        // Skip deleted QNames
        if id == istructs.NullQNameID {
            return nil
        }

        // Store in memory cache
        names.qNames[qName] = id
        names.ids[id] = qName

        return nil
    }

    // Read all entries from SysView_QNames
    pKey := utils.ToBytes(consts.SysView_QNames, ver01)
    return storage.Read(context.Background(), pKey, nil, nil, readQName)
}
```

**Assigning new QNameIDs:**

```go
func (names *QNames) collect(qName appdef.QName) error {
    if _, ok := names.qNames[qName]; ok {
        return nil // Already known QName
    }

    // Find next available ID after lastID
    for id := names.lastID + 1; id < MaxAvailableQNameID; id++ {
        if _, ok := names.ids[id]; !ok {
            // Found unused ID
            names.qNames[qName] = id
            names.ids[id] = qName
            names.lastID = id
            names.changes++
            return nil
        }
    }

    return ErrQNameIDsExceeds // No more IDs available
}
```

**Storing to storage:**

```go
func (names *QNames) store(storage, versions) error {
    pKey := utils.ToBytes(consts.SysView_QNames, ver01)

    batch := make([]istorage.BatchItem, 0)
    for qName, id := range names.qNames {
        // Only store user-defined QNames (> 255)
        if id > istructs.QNameIDSysLast {
            item := istorage.BatchItem{
                PKey:  pKey,
                CCols: []byte(qName.String()),
                Value: utils.ToBytes(id),
            }
            batch = append(batch, item)
        }
    }

    // Write all mappings in one batch
    err = storage.PutBatch(batch)

    // Update version in SysView_Versions
    if ver := versions.Get(vers.SysQNamesVersion); ver != latestVersion {
        err = versions.Put(vers.SysQNamesVersion, latestVersion)
    }

    return nil
}
```

#### QNames storage example

If an application has these QNames:

```text
"myapp.Order" -> 256
"myapp.Customer" -> 257
"myapp.Product" -> 258
```

SysView_QNames contains:

```text
Partition Key: [0x00][0x11][0x00][0x01]  (SysView_QNames=17, ver01=1)

Entry 1:
  Clustering Columns: "myapp.Customer"
  Value: [0x01][0x01]  (257)

Entry 2:
  Clustering Columns: "myapp.Order"
  Value: [0x01][0x00]  (256)

Entry 3:
  Clustering Columns: "myapp.Product"
  Value: [0x01][0x02]  (258)
```

Note: Entries are sorted by clustering columns (QName strings)

#### QName renaming

Voedger supports renaming QNames while preserving their QNameID:

```go
// Rename "old.Name" to "new.Name"
err := qnames.Rename(storage, oldQName, newQName)

// This:
// 1. Loads existing mappings
// 2. Finds QNameID for oldQName
// 3. Marks oldQName as deleted (sets ID to NullQNameID)
// 4. Assigns the same ID to newQName
// 5. Writes changes to storage
```

This is critical because:

- QNameIDs are used in storage keys throughout the system
- Changing a QNameID would require rewriting all data
- Renaming preserves the ID, so existing data remains valid

#### Lookup operations

**QName to QNameID:**

```go
id, err := qnames.ID(appdef.NewQName("myapp", "Order"))
// Returns: 256, nil
```

**QNameID to QName:**

```go
qname, err := qnames.QName(256)
// Returns: QName{pkg: "myapp", entity: "Order"}, nil
```

#### QNames implementation characteristics

- Bidirectional mapping - Fast lookup in both directions (QName -> ID and ID -> QName)
- In-memory cache - All mappings loaded at startup, no storage access during runtime
- Immutable IDs - Once assigned, a QNameID never changes for a given QName
- Sequential allocation - New IDs assigned sequentially from lastID + 1
- Batch writes - All changes written in a single batch operation
- Version tracking - Uses SysView_Versions to track schema version
- Deletion support - QNames can be marked as deleted (ID set to NullQNameID)
- Rename support - QNames can be renamed while preserving their ID

#### Why it's needed

**Storage efficiency:**

- 2 bytes (QNameID) vs 10-50 bytes (QName string)
- Used in every record, view, and event log entry
- Massive space savings across the entire database

**Performance:**

- Integer comparison (2 bytes) vs string comparison (variable length)
- Faster key generation and lookups
- Reduced memory footprint

**Example impact:**

Without QNameID:
```text
View partition key: [ViewID][WSID]["myapp.OrderStatus"]
Size: 2 + 8 + 18 = 28 bytes
```

With QNameID:
```text
View partition key: [ViewQNameID][WSID]
Size: 2 + 8 = 10 bytes
```

Savings: 18 bytes per key × millions of records = significant storage reduction

---

### SysView_Containers

SysView_Containers (ID=18) is a system view that stores the mapping between container names and their corresponding ContainerIDs (uint16 numeric identifiers). Containers are used to organize records within documents.

#### Containers data structure

**Partition key (4 bytes total):**

```text
[0-1]    uint16   SysView_Containers (constant = 18)
[2-3]    uint16   Version (ver01 = 1)
```

**Clustering columns (variable length):**

```text
[0...]   string   Container name as string (e.g., "items")
```

**Value (2 bytes):**

```text
[0-1]    uint16   ContainerID (e.g., 64)
```

#### Container ID ranges

```text
Range           Purpose
0               NullContainerID
1-63            Reserved (ContainerNameIDSysLast)
64-65535        User-defined containers
```

#### Containers lifecycle

Similar to QNames, containers are:

- Loaded at application startup
- Cached in memory (pkg/istructsmem/internal/containers)
- Persisted to storage when new containers are added
- Immutable once assigned (ContainerID never changes for a given name)

**Loading from storage:**

```go
func (cnt *Containers) load01(storage istorage.IAppStorage) error {
    readName := func(cCols, value []byte) error {
        name := string(cCols)
        id := ContainerID(binary.BigEndian.Uint16(value))

        if id == NullContainerID {
            return nil // deleted Container
        }

        cnt.containers[name] = id
        cnt.ids[id] = name
        return nil
    }

    pKey := utils.ToBytes(consts.SysView_Containers, ver01)
    return storage.Read(context.Background(), pKey, nil, nil, readName)
}
```

**Storing to storage:**

```go
func (cnt *Containers) store(storage, versions) error {
    pKey := utils.ToBytes(consts.SysView_Containers, latestVersion)

    batch := make([]istorage.BatchItem, 0)
    for name, id := range cnt.containers {
        if name == "" {
            continue // skip NullContainerID
        }
        item := istorage.BatchItem{
            PKey:  pKey,
            CCols: []byte(name),
            Value: utils.ToBytes(id),
        }
        batch = append(batch, item)
    }

    err = storage.PutBatch(batch)

    // Update version in SysView_Versions
    if ver := versions.Get(vers.SysContainersVersion); ver != latestVersion {
        err = versions.Put(vers.SysContainersVersion, latestVersion)
    }

    return nil
}
```

---

### SysView_SingletonIDs

SysView_SingletonIDs (ID=22) is a system view that stores the mapping between singleton QNames and their corresponding RecordIDs. Singletons are special records that exist only once per workspace.

#### SingletonIDs data structure

**Partition key (4 bytes total):**

```text
[0-1]    uint16   SysView_SingletonIDs (constant = 22)
[2-3]    uint16   Version (ver01 = 1)
```

**Clustering columns (variable length):**

```text
[0...]   string   QName as string (e.g., "myapp.Settings")
```

**Value (8 bytes):**

```text
[0-7]    uint64   RecordID (e.g., 65536)
```

#### Singleton ID ranges

```text
Range                    Purpose
65536 - 66047           Singleton IDs (512 total)
```

#### SingletonIDs lifecycle

Similar to QNames and Containers, singleton IDs are:

- Loaded at application startup
- Cached in memory (pkg/istructsmem/internal/singletons)
- Persisted to storage when new singletons are added
- Immutable once assigned (RecordID never changes for a given singleton)

**Loading from storage:**

```go
func (st *Singletons) load01(storage istorage.IAppStorage) error {
    readSingleton := func(cCols, value []byte) error {
        qName, err := appdef.ParseQName(string(cCols))
        if err != nil {
            return err
        }
        id := istructs.RecordID(binary.BigEndian.Uint64(value))

        st.qNames[qName] = id
        st.ids[id] = qName
        return nil
    }

    pKey := utils.ToBytes(consts.SysView_SingletonIDs, ver01)
    return storage.Read(context.Background(), pKey, nil, nil, readSingleton)
}
```

**Storing to storage:**

```go
func (st *Singletons) store(storage, versions) error {
    pKey := utils.ToBytes(consts.SysView_SingletonIDs, latestVersion)

    batch := make([]istorage.BatchItem, 0)
    for qName, id := range st.qNames {
        if id >= istructs.FirstSingletonID {
            item := istorage.BatchItem{
                PKey:  pKey,
                CCols: []byte(qName.String()),
                Value: utils.ToBytes(uint64(id)),
            }
            batch = append(batch, item)
        }
    }

    err = storage.PutBatch(batch)

    // Update version in SysView_Versions
    if ver := versions.Get(vers.SysSingletonsVersion); ver != latestVersion {
        err = versions.Put(vers.SysSingletonsVersion, latestVersion)
    }

    return nil
}
```

---

## Field serialization rules

When fields are serialized in partition keys and clustering columns:

**Fixed-size types (BigEndian):**
```text
int8      -> 1 byte
int16     -> 2 bytes (BigEndian)
int32     -> 4 bytes (BigEndian)
int64     -> 8 bytes (BigEndian)
uint16    -> 2 bytes (BigEndian)
uint32    -> 4 bytes (BigEndian)
uint64    -> 8 bytes (BigEndian)
float32   -> 4 bytes (BigEndian, IEEE 754 bits)
float64   -> 8 bytes (BigEndian, IEEE 754 bits)
bool      -> 1 byte (0x00=false, 0x01=true)
QName     -> 2 bytes (BigEndian QNameID)
RecordID  -> 8 bytes (BigEndian)
```

**Variable-size types:**
```text
string    -> Written as-is (UTF-8 bytes)
bytes     -> Written as-is (raw bytes)
```

**Note:** Variable-size fields (string, bytes) should be placed last in clustering columns for efficient range queries.

---

## RecordID structure

RecordID is a 64-bit unsigned integer with specific ranges:

```text
Range                    Purpose
0                        NullRecordID
1 - 65535               Raw IDs (temporary, client-generated)
65536 - 200000          Reserved range
  65536 - 66047           - Singleton IDs (512 total)
  66048                   - NonExistingRecordID (testing)
  66049 - 200000          - Other reserved IDs
200001+                 User record IDs
```

**Key constants:**

```go
const NullRecordID = RecordID(0)
const MinRawRecordID = RecordID(1)
const MaxRawRecordID = RecordID(0xffff)  // 65535

const MinReservedRecordID = MaxRawRecordID + 1  // 65536
const MaxReservedRecordID = RecordID(200000)

const FirstSingletonID = MinReservedRecordID  // 65536
const MaxSingletonID = FirstSingletonID + 0x1ff  // 66047 (512 singletons)

const NonExistingRecordID = MaxSingletonID + 1  // 66048

const FirstUserRecordID = MaxReservedRecordID + 1  // 200001
```

**Splitting for Storage:**

- Partition bits: 12 (lower bits)
- Creates partitions of 4,096 records each
- Formula: `partitionBits = 12`
- Hi = upper 52 bits, Lo = lower 12 bits

## WSID (Workspace ID) structure

WSID is a 63-bit value (highest bit always 0):

```text
Bit:  63  62-47        46-0
      0   ClusterID    BaseWSID
```

- Bit 63: Always 0 (allows safe int64 casting)
- Bits 62-47: ClusterID (16 bits)
- Bits 46-0: BaseWSID (47 bits)

**Formula:**
```go
WSID = (ClusterID << 47) + BaseWSID
```

## Data flow

### Writing a View Record

```text
1. Application calls IViewRecords.Put(wsid, keyBuilder, valueBuilder)
   |
2. keyBuilder.ToBytes(wsid) generates:
   - Partition Key: [ViewQNameID][WSID][PartitionKeyFields]
   - Clustering Columns: [ClusteringColumnFields]
   |
3. valueBuilder.ToBytes() serializes value data
   |
4. IAppStorage.Put(pKey, cCols, value)
   |
5. Storage backend writes to underlying database
```

### Reading a View Record

```text
1. Application calls IViewRecords.Get(wsid, keyBuilder)
   |
2. keyBuilder.ToBytes(wsid) generates pKey and cCols
   |
3. IAppStorage.Get(pKey, cCols, &data)
   |
4. Storage backend retrieves from database
   |
5. Value deserialized from bytes to IValue
```

### Range Reading (Scan)

```text
1. Application calls IViewRecords.Read(ctx, wsid, keyBuilder, callback)
   |
2. keyBuilder.ToBytes(wsid) generates:
   - Partition Key: [ViewQNameID][WSID][PartitionKeyFields]
   - Start Clustering Columns: [SpecifiedFields]
   - Finish Clustering Columns: [SpecifiedFields + 0xFF...]
   |
3. IAppStorage.Read(ctx, pKey, startCCols, finishCCols, callback)
   |
4. Storage backend iterates over range
   |
5. For each record: callback(cCols, value)
   |
6. Deserialize and process each record
```

## Implementation details

### Translation layer (pkg/istructsmem)

The `istructsmem` package implements high-level `istructs` interfaces by translating them into low-level `IAppStorage` calls:

```text
High Level (istructs)          Translation          Low Level (istorage)
IRecords.Get(wsid, id)    ->   recordKey()     ->   IAppStorage.Get(pKey, cCols)
IViewRecords.Put(...)     ->   storeToBytes()  ->   IAppStorage.Put(pKey, cCols, value)
IEvents.PutPlog(...)      ->   plogKey()       ->   IAppStorage.Put(pKey, cCols, data)
IEvents.ReadWLog(...)     ->   wlogKey()       ->   IAppStorage.Read(pKey, startCCols, finishCCols)
```

**Key Generation Functions (pkg/istructsmem/utils.go):**

- `recordKey(ws WSID, id RecordID) (pkey, ccols []byte)` - Generates keys for records
- `plogKey(partition PartitionID, offset Offset) (pkey, ccols []byte)` - Generates keys for PLog
- `wlogKey(ws WSID, offset Offset) (pkey, ccols []byte)` - Generates keys for WLog

**View Key Generation (pkg/istructsmem/viewrecords-dynobuf.go):**

- `storeViewPartKey(ws WSID) []byte` - Generates partition key for views
- `storeViewClustKey() []byte` - Generates clustering columns for views

### Caching layer

**istoragecache** (pkg/istoragecache) provides transparent caching:

```text
IAppStorage (cached)
    |
    +-- Cache (LRU)
    |
    +-- IAppStorage (underlying)
```

## Usage examples

### Example: Writing a Record

```go
// High-level API
err := appStructs.Records().PutJSON(wsid, map[string]interface{}{
    "sys.ID": 200001,
    "sys.QName": "myapp.Order",
    "CustomerName": "John Doe",
    "Amount": 100.50,
})

// Under the hood:
// 1. Build record from JSON
// 2. Generate keys: pk, cc := recordKey(wsid, 200001)
//    pk = [0x00][0x13][WSID bytes][RecordID_Hi bytes]
//    cc = [RecordID_Lo bytes]
// 3. Serialize record data
// 4. Call storage.Put(pk, cc, data)
```

### Example: Writing a View Record

```go
// Get view records interface
viewRecords := appStructs.ViewRecords()

// Build key
kb := viewRecords.KeyBuilder(viewQName)
kb.PartitionKey().PutInt64("partitionKey1", 1)
kb.ClusteringColumns().PutInt32("clusteringColumn1", 100)
kb.ClusteringColumns().PutString("clusteringColumn2", "test")

// Build value
vb := viewRecords.NewValueBuilder(viewQName)
vb.PutInt64("valueField1", 123)
vb.PutString("valueField2", "data")

// Store
err := viewRecords.Put(wsid, kb, vb)

// Under the hood:
// 1. kb.ToBytes(wsid) generates:
//    pk = [ViewQNameID][WSID][partitionKey1 bytes]
//    cc = [clusteringColumn1 bytes][clusteringColumn2 bytes]
// 2. vb.ToBytes() serializes value
// 3. Call storage.Put(pk, cc, value)
```

### Example: Reading a View Record

```go
// Build key
kb := viewRecords.KeyBuilder(viewQName)
kb.PartitionKey().PutInt64("partitionKey1", 1)
kb.ClusteringColumns().PutInt32("clusteringColumn1", 100)
kb.ClusteringColumns().PutString("clusteringColumn2", "test")

// Read
value, err := viewRecords.Get(wsid, kb)
if err != nil {
    // Handle error
}

// Access fields
field1 := value.AsInt64("valueField1")
field2 := value.AsString("valueField2")
```

### Example: Range Reading (Scan)

```go
// Build key with partial clustering columns
kb := viewRecords.KeyBuilder(viewQName)
kb.PartitionKey().PutInt64("partitionKey1", 1)
kb.ClusteringColumns().PutInt32("clusteringColumn1", 100)
// Note: clusteringColumn2 not specified - will scan all values

// Read range
err := viewRecords.Read(ctx, wsid, kb, func(key IKey, value IValue) error {
    // Process each record
    col1 := key.AsInt32("clusteringColumn1")
    col2 := key.AsString("clusteringColumn2")
    val1 := value.AsInt64("valueField1")
    return nil
})
```
