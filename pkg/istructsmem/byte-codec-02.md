# Codec used to represent the structures in byte form

## Introduction

The application structures (records, events, representations) in the database are stored in the form of an array of bytes.
Codec — a set of algorithms for the presentation of stored structures in the form of bytes arrays used in a particular version of the application.

## Versions

Codec has its own version. Codec version number - a whole unsigned byte value (*byte*). The later version of Codec corresponds to a larger version number. The codec of one version can be used by the application of different versions. The application of any version should be able to record the data in the last (at the time of the application) of the Codec version, and read the data encoded by this or any earlier version of the Codec.

### Current codec version

The current version of the codec: `2` .

### Codec version changes summary

The storage of the plan of modified records *commandCUDs* has been changed.
In addition to changes in records, the plan of modified records contains information about emptied (cleared) fields.

## Record

Record *record* contains the Codec version *codecVer* and a data *rowData*.

`record = codecVer rowData .`

### Codec Version (record)

Codec version *codecVer* stored as an unsigned single byte number.

`codecVer = byte . // (0x00)`

### Row Data

Row data *rowData* contains the type name identifier *qNameID*, the values of the system fields *sysFields* and field values with user data *userFields*.

`rowData = qNameID [sysFields] [userFields] .`

If type name `QName` is not specified (i.e. equal to `appdef.NullQName`), then other members (*sysFields* и *userFields*) are omitted.

#### QName ID (record)

Type name identifier *qNameID* is stored as unsigned two-bytes number. Here and in the following, unless specifically stated, numerical values are recorded using a byte order from most significant to least significant. (`BigEndian`).

`qNameID = uint16 . // BigEndian`

#### System fields

❗ *Changed* compared to the previous version

System fields include the bit mask of system fields *sysFieldsMask*, the record identifier *sys.ID*, the owning record identifier *sys.ParentID*, the container *sys.Container*, and the activity indicator *sys.IsActive*.

`systemFields = sysFieldsMask [sys.ID] [sys.ParentID] [sys.Container] [sys.IsActive] .`

#### System fields mask

The system fields bit mask is an unsigned two-byte number, each set bit of which corresponds to the stored value of a system field.

| Bit | Mask | System field | Comment |
| --: | :---: | :------------- | ----------- |
|   0 | x0001 | sys.ID         | Record identifier is provided for documents (`CDoc`, `GDoc`, `ODoc`, `WDoc`) and for records (`CRecord`, `GRecord`, `ORecord`, `WRecord`) |
|   1 | x0002 | sys.ParentID   | The identifier of the owner entry (document) is provided for by the definitions of records (`CRecord`, `GRecord`, `ORecord`, `WRecord`). Some documents (`ODoc`) They also allow an indication of the identifier of the owner document if they are invested in the document |
|   2 | x0004 | sys.Container  | The container identifier is provided for records (`CRecord`, `GRecord`, `ORecord`, `WRecord`) and for objects (`Object`). Some documents (`ODoc`) also allow the container if they are nested into document |
|   3 | x0008 | sys.IsActive   | A sign of activity is provided for documents (`CDoc`, `GDoc`, `WDoc`) and records (`CRecord`, `GRecord`, `WRecord`) |

##### Record ID

The record identifier *sys.ID* is recorded as an unsigned four-byte number.

`sys.ID = uint64 . // BigEndian`

##### Parent ID

The owning record identifier *sys.ParentID* is recorded as an unsigned four-byte number.

`sys.ParentID = uint64 . // BigEndian`

##### Container

The container identifier *sys.Container* is recorded as an unsigned two-byte number.

`sys.Container = unit16 . // BigEndian`

##### Is active

The activity indicator *sys.IsActive* is recorded as a Boolean value (1 byte).

`sys.IsActive = bool .`

#### User fields

The values of user fields *userFields* are stored inside [dynoBuffer](https://github.com/untillpro/dynobuffers) and contain the buffer length *bufferLength* and buffer data *buffer*.

`userFields = bufferLength [buffer] .`

If the buffer length is 0 (zero), then the data following the length is absent.

##### Buffer Length

The buffer length *bufferLength* is recorded as an unsigned four-byte number.

`bufferLength = uint32 .`

##### Buffer Bytes

The buffer data *buffer* consists of a sequence of one or more bytes.

`buffer = byte { byte } . // [bufferLength]byte`

The internal structure of the buffer is determined by the [dynoBuffer](https://github.com/untillpro/dynobuffers) scheme. The buffer contains data from all filled user fields. Information about the values of system fields(`sys.QName`, `sys.ID`, `sys.ParentID`, `sys.Container`) is absent in the buffer.

## Event

The event buffer *event* contains the codec version *codecVer* and event data *eventData*.

`event = codecVersion eventData .`

### Codec Version (event)

The codec version *codecVersion* is recorded as an unsigned one-byte integer.

`codecVersion = byte . // (0x00)`

### Event Data

The event data *eventData* contains the definition name identifier *qNameID*, event construction data *createParameters*, event build error information *buildError*, command arguments *commandArguments*, and the plan of modified records *commandCUDs*.

`eventData = qNameID [ createParameters buildError [ commandArguments commandCUDs ] ].`

If the definition name (QName) is not specified for the event (i.e., it is equal to `appdef.NullQName`), then nothing else is specified for the event.

#### QName ID (event)

The name identifier *qNameID* is recorded as an unsigned two-byte number.

`qNameID = uint16 .`

#### Create Parameters

The event construction data *createParameters* contains information about the static fields of the event, specified during its construction.

`createParameters = ( partition pLogOffs ) ( workSpace wLogOffs ) registerTime ( sync [device syncTime] ) .`

##### Partition

The event partition *partition* is recorded as an unsigned two-byte number.

`partition = uint16 .`

##### PLog Offset

The event offset in the processing log *pLogOffs* is recorded as an unsigned eight-byte number.

`partition = uint64 .`

##### Work Space

The event workspace *workSpace* is recorded as an unsigned eight-byte number.

`workSpace = uint64 .`

##### WLog Offset

The event offset in the workspace log *wLogOffs* is recorded as an unsigned eight-byte number.

`workSpace = uint64 .`

##### Register Time

The event registration time *registerTime* is recorded as an eight-byte number.

`registerTime = int64 .`

##### Sync Flag

The event completion flag *sync* is recorded as a boolean value (1 byte).

`sync = bool .`

Если указано `false`, то следующие два поля (*device* и *syncTime*) отсутствуют.

##### Device ID

The identifier of the device on which the event occurred *device* is recorded as an unsigned two-byte number.

`device = uint16 . // filled if sync == true

##### Sync Time

The time when the event occurred *syncTime* is recorded as an eight-byte number.

`syncTime = int64 . // filled if sync == true

#### Build Error

The event build error information *buildError* contains information about an error that occurred during the event construction, or information about data corruption if the event is marked as an event with corrupted data (QName = "sys.Corrupted").

`buildError = valid [ ( messageLength [ messageBytes ] ) ( qNameLength [ qNameChars ] ) ( [ rawDataLength [ rawDataBytes ] ] ) ] .`

##### Valid Flag

The build success flag *valid* is recorded as a boolean value (1 byte).

`valid = bool .`

The build success flag *valid* contains information on whether the message was successfully built (`true`) or if an error occurred during the build (`false`).
If `true` is recorded, the other members describing the build error are absent.
If `false` is recorded, the members after the build error description (command arguments and modified records) are absent.

##### Error Message Length

The length of the event build error message *messageLength* is recorded as an unsigned two-byte number.

`messageLength = uint16 .`

If the length of the error message is `0` (zero), then the next member (error message *messageBytes*) is absent.

##### Error Message Text

The event build error message *messageBytes* is recorded as a sequence of bytes.

`messageBytes = byte { byte } . // [messageLength]byte`

##### Source QName length

The length of the original event command name *qNameLength* is recorded as an unsigned two-byte number.

`qNameLength = uint16 .`

If the length of the original command name is `0` (zero), then the next member (original command name string *qNameChars*) is absent.

##### Source QName string

The original event command name string *qNameChars* is recorded as a sequence of bytes.

`qNameChars = byte { byte } . // [qNameLength]byte`

##### Raw Event Data Length

The length of the original event data *rawDataLength* is recorded as an unsigned four-byte number.

`rawDataLength = uint32 .`

If the length of the data is `0` (zero), then the next member (original data *rawDataBytes*) is absent.

##### Raw Event Data Bytes

The original event data *rawDataBytes* is recorded as a sequence of bytes.

`rawDataBytes = byte { byte } . // [rawDataLength]byte`

#### Command Argument

Command arguments *commandArguments* contain an object *object* and a secret object *unloggedObject*.

`commandArguments = ( object unloggedObject ) .`

##### Argument Object

The object *object* contains the object data in the format of [row data](#row-data) *rowData*, the number of child elements *childCount*, and the list of child elements *children*.

`object = rowData [ childCount { children } ] .`

If the definition name `QName` of the object is not specified (i.e., it is equal to `appdef.NullQName`), then the other members of the object are absent.

###### Argument Children

The element *children* contains the data of child objects in the format of [row data](#row-data) *rowData*, the number of child documents *childCount*, and the list of child elements *children*.

`children = rowData [ childCount { children } ] .`

If the definition name `QName` of the element is not specified (i.e., it is equal to `appdef.NullQName`), then the other members of the object are absent.

##### Unlogged Argument Object

The secret object *unloggedObject* contains masked object data in the format of row data *rowData*, the number of child objects *childCount*, and the list of child objects *unloggedChildren* also with masked data.

`unloggedObject = rowData [ childCount { unloggedChildren } ] .`

The secret object with its elements differs from the regular object with regular elements in that the data of user fields in the former are masked:

- Values of all numeric fields are set to `0` (zero),
- Values of all string fields are set to `*` (asterisk),
- Values of all boolean fields are set to `false`,
- Length of all byte fields is set to `0` (zero),
- Values of all fields with a qualified name are set to `appdef.NullQName`.

#### Command CUDs

The records modified by the command *commandCUDs* contain information about the records scheduled for modification.

This information includes the number of new records *createCount*, the list of new records *creates*, the number of modified records *updateCount*, and the list of changes in the records *updates*.

`commandCUDs = ( createCount [ creates ] ) ( updateCount [ updates ] ) .`

##### Create Records Count

The number of created (new) records *createCount* is recorded as an unsigned two-byte number.

`createCount = unit16 .`

##### Created Records

The list of created (new) records *creates* contains an array of new [CUD records](#cud-data) *cudData*.

If the number of created records is `0` (zero), then the list of created records is absent.

`creates = cudData { cudData } .`

##### Update Record Count

The number of modified records *updateCount* is recorded as an unsigned two-byte number.

If the number of modified records is `0` (zero), then the list of modified records is absent.

`updateCount = unit16 .`

##### Updated Records

The list of modified records *updates* contains an array of modified [CUD records](#cud-data) *cudData*.

`updates = cudData { cudData } .`

In the modified data, the rows contain:

- values of all system fields provided by the type and
- values of only the modified user fields.

Information about the values of the remaining (unchanged) user fields is not included in the list.

###### CUD Data

The CUD record data *cudData* contains [row data](#row-data) *rowData* and [list of emptied fields](#emptied-fields) *emptiedFields*.

`cudData = rowData emptiedFields .`

###### Emptied Fields

❗ *New* in version

The list of emptied fields *emptiedFields* contains the [count of emptied fields](#emptied-fields-count) *emptiedCount* and [list of indexes of emptied fields](#emptied-fields-indexes) *emptiedFieldsIndexes*.

`emptiedFields = emptiedCount emptiedFieldsIndexes .`

###### Emptied Fields Count

❗ *New* in version

The count of emptied fields *emptiedCount* is recorded as an unsigned two-byte number.

If count of emptied fields is zero, then the [list of indexes of emptied fields](#emptied-fields-indexes) is absent.

`emptiedCount = unit16 .`

###### Emptied Fields Indexes

❗ *New* in version

The list of indexes of emptied fields *emptiedFieldsIndexes* contains an array of unsigned two-byte numbers.

`emptiedFieldsIndexes = uint16 { uint16 } .`

The index of the field in the list of emptied fields corresponds to the index of the field in the list of *user* fields of the record. The first user field has an index of `0` (zero), the second - `1` (one), and so on.

## View Record Value

The structure of the representation record matches the structure of the [record](#record). However, in addition to fields with simple data (numbers, strings, bytes, etc.), representation records can contain user fields of complex types `appdef.DataKind_Record` and `appdef.DataKind_Event`. The values of fields of these types are stored inside *userFields* as byte arrays, the internal structure of which is defined above in the sections [Record](#record) and [Event](#event).
