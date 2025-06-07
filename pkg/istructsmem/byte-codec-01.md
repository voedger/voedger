# Codec used to represent the structures in byte form

## Introduction

The application structures (records, events, representations) in the database are stored in the form of an array of bytes.
Codec — a set of algorithms for the presentation of stored structures in the form of bytes arrays used in a particular version of the application.

## Versions

Codec has its own version. Codec version number - a whole unsigned byte value (*byte*). The later version of Codec corresponds to a larger version number. The codec of one version can be used by the application of different versions. The application of any version should be able to record the data in the last (at the time of the application) of the Codec version, and read the data encoded by this or any earlier version of the Codec.

### Current codec version

The current version of the codec: `1` .

### Codec version changes summary

The storage of the values of the system fields has been changed. In the previous version of the Codec, whether the system field will be recorded by the defiant code on the basis of the type of definition.I n the current version, the values of the system fields precedes the bit, which indicates which system fields are preserved.

## Record

Record *record* contains the Codec version *codecVer* and a data *rowData*.

`record = codecVer rowData .`

### Codec Version

Codec version *codecVer* stored as an unsigned single byte number.

`codecVer = byte . // (0x00)`

### Row Data

Row data *rowData* contains the type name identifier *QNameID*, the values of the system fields *sysFields* and field values with user data *userFields*.

`rowData = QNameID [sysFields] [userFields] .`

If type name `QName` is not specified (i.e. equal to `appdef.NullQName`), then other members (*sysFields* и *userFields*) are omitted.

#### QName ID

Type name identifier *QNameID* is stored as unsigned two-bytes number. Here and in the following, unless specifically stated, numerical values are recorded using a byte order from most significant to least significant. (`BigEndian`).

`QNameID = uint16 . // BigEndian`

#### System fields

❗ *Changed* compared to the previous version

System fields include the bit mask of system fields *sysFieldsMask*, the record identifier *sys.ID*, the owning record identifier *sys.ParentID*, the container *sys.Container*, and the activity indicator *sys.IsActive*.

`systemFields = sysFieldsMask [sys.ID] [sys.ParentID] [sys.Container] [sys.IsActive] .`

#### System fields mask

❗ *New* in version

The system fields bit mask is an unsigned two-byte number, each set bit of which corresponds to the stored value of a system field.

| Bit | Mask | System field | Comment |
| --: | :---: | :------------- | ----------- |
|   0 | x0001 | sys.ID         | Record identifier is provided for documents (`CDoc`, `GDoc`, `ODoc`, `WDoc`) and for records (`CRecord`, `GRecord`, `ORecord`, `WRecord`)
|   1 | x0002 | sys.ParentID   | The identifier of the owner entry (document) is provided for by the definitions of records (`CRecord`, `GRecord`, `ORecord`, `WRecord`). Some documents (`ODoc`) They also allow an indication of the identifier of the owner document if they are invested in the document
|   2 | x0004 | sys.Container  | The container identifier is provided for records (`CRecord`, `GRecord`, `ORecord`, `WRecord`) and for objects (`Object`). Some documents (`ODoc`) also allow the container if they are nested into document
|   3 | x0008 | sys.IsActive   | A sign of activity is provided for documents (`CDoc`, `GDoc`, `WDoc`) and records (`CRecord`, `GRecord`, `WRecord`).

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

Буфер события *event* содержит версию кодека *codecVer* и данные события *eventData*.

`event = codecVersion eventData .`

### Codec Version

Версия кодека *codecVersion* записана как беззнаковое однобайтовое целое число.

`codecVersion = byte . // (0x00)`

### Event Data

Данные события *eventData* содержат идентификатор имени определения *QNameID*, конструкционные данные события *createParameters*, сведения об ошибке сборки события *buildError*, аргументы команды *commandArguments* и план изменяемых записей *commandCUDs*.

`eventData = QNameID [ createParameters buildError [ commandArguments commandCUDs ] ].`

Если имя определения (QName) у события не указано (т.е. равно `appdef.NullQName`), то больше ничего для события не указывается.

#### QName ID

Идентификатор имени *QNameID* записывается как беззнаковое двух-байтовое число.

`QNameID = uint16 .`

#### Create Parameters

Конструкционные данные события *createParameters* содержат информацию о статических полях события, заданных при его конструировании.

`createParameters = ( partition pLogOffs ) ( workSpace wLogOffs ) registerTime ( sync [device syncTime] ) .`

##### Partition

Раздел обработки события *partition* записан как беззнаковое двух-байтовое число.

`partition = uint16 .`

##### PLog Offset

Смещение события в журнале обработки *pLogOffs* записано как беззнаковое  восьми-байтовое число.

`partition = uint64 .`

##### Work Space

Рабочая область события *workSpace* записана как беззнаковое восьми-байтовое число.

`workSpace = uint64 .`

##### WLog Offset

Смещение события в журнале рабочей области *wLogOffs* записано как беззнаковое восьми-байтовое число.

`workSpace = uint64 .`

##### Register Time

Время регистрации события *registerTime* записано как восьми-байтовое число.

`registerTime = int64 .`

##### Sync Flag

Признак состоявшегося события *sync* записан как булевское значение (1 байт).

`sync = bool .`

Если указано `false`, то следующие два поля (*device* и *syncTime*) отсутствуют.

##### Device ID

Идентификатор устройства, на котором состоялось событие, *device* записан как беззнаковое двух-байтовое число.

`device = uint16 . // заполняется, если sync == true`

##### Sync Time

Время, когда состоялось событие, *syncTime* записано как восьми-байтовое число.

`syncTime = int64 . // заполняется, если sync == true`

#### Build Error

Данные об ошибке сборки события *buildError* содержат информацию об ошибке, произошедшей во время сборки события, либо сведения о факте повреждения данных, если событие отмечено, как событие с поврежденными данными (QName = "sys.Corrupted")

`buildError = valid [ ( messageLength [ messageBytes ] ) ( qNameLength [ qNameChars ] ) ( [ rawDataLength [ rawDataBytes ] ] ) ] .`

##### Valid Flag

Флаг успешной сборки *valid* записан как булевское значение (1 байт).

`valid = bool .`

Флаг успешной сборки *valid* содержит информацию, прошло ли сообщение сборку успешно (`true`) или же при сборке возникла ошибка (`false`). Если записана истина (`true`), то остальные члены, описывающие ошибку сборки, отсутствуют.
Если записана ложь (`false`), то члены после описания ошибки сборки (аргументы команды и изменяемые записи) отсутствуют.

##### Error Message Length

Длина сообщения об ошибке при сборке события *messageLength* записана как беззнаковое двух-байтовое число.

`messageLength = uint16 .`

Если длина сообщения об ошибке `0` (ноль), то следующий член (сообщение об ошибке *messageBytes*) отсутствует.

##### Error Message Text

Сообщение об ошибке при сборке события *messageBytes* записано как последовательность байт.

`messageBytes = byte { byte } . // [messageLength]byte`

##### Source QName length

Длина исходного имени команды события *qNameLength* записана как беззнаковое двух-байтовое число.

`qNameLength = uint16 .`

Если длина исходного имени команды `0` (ноль), то следующий член (строка исходного имени команды *qNameChars*) отсутствует.

##### Source QName string

Строка исходного имени команды события *qNameChars* записана как последовательность байт.

`qNameChars = byte { byte } . // [qNameLength]byte`

##### Raw Event Data Length

Длина исходных данных события *rawDataLength* записана как беззнаковое четырех-байтовое число.

`rawDataLength = uint32 .`

Если длина данных `0` (ноль), то следующий член (исходные данные *rawDataBytes*) отсутствует.

##### Raw Event Data Bytes

Исходные данные события *rawDataBytes* записаны как последовательность байт.

`rawDataBytes = byte { byte } . // [rawDataLength]byte`

#### Command Argument

Аргументы команды *commandArguments* содержат объект *object* и секретный объект *unloggedObject*

`commandArguments = ( object unloggedObject ) .`

##### Argument Object

Объект *object* содержит данные объекта в формате [строка данных](#row-data) *rowData*, количество дочерних элементов *childCount* и список дочерних элементов *children*.

`object = rowData [ childCount { children } ] .`

Если имя определения `QName` у объекта не указано (т.е. равно `appdef.NullQName`), то остальные члены объекта отсутствуют.

###### Argument Children

Элемент *children* содержит данные дочерних объектов в формате [строка данных](#row-data) *rowData*, количество дочерних документов *childCount* и список дочерних элементов *children*.

`children = rowData [ childCount { children } ] .`

Если имя определения `QName` у элемента не указано (т.е. равно `appdef.NullQName`), то остальные члены объекта отсутствуют.

##### Unlogged Argument Object

Секретный объект *unloggedObject* содержит маскированные данные объекта в формате строки данных *rowData*, количество дочерних объектов *childCount* и список дочерних объектов *unloggedChildren* так же с маскированными данными.

`unloggedObject = rowData [ childCount { unloggedChildren } ] .`

От обычного объекта с обычными элементами секретный объект c его элементами отличаются тем, что в последних данные пользовательских полей маскированы:

- Значения всех числовых полей установлены в `0` (ноль),
- Значения всех строковых полей установлены в `*` (звездочка),
- Значения всех булевских полей установлены в `false` (ложь),
- Длина всех байтовых полей установлена в `0` (ноль),
- Значения всех полей с квалифицированным именем установлена в `appdef.NullQName`.

#### Command CUDs

Изменяемые командой записи *commandCUDs* содержат сведения о записях, запланированных к изменению в аргументах команды (для системной команды `sys.CUD`), либо для записей, изменяемых в результате выполнения (для всех остальных команд). Эти сведения содержат количество новых записей *createCount*, список новых записей *creates*, количество измененных записей *updateCount*, список изменений в записях *updates*.

`commandCUD = ( createCount { creates } ) ( updateCount { updates } ) .`

##### Create Records Count

Количество создаваемых (новых) записей *createCount* записано как беззнаковое двух-байтовое число.

`createCount = unit16 .`

##### Created Records

Список создаваемых (новых) записей *creates* содержит массив новых [строк данных](#row-data) *rowData*.

`creates = { rowData }`

##### Update Record Count

Количество измененных записей *updateCount* записано как беззнаковое двух-байтовое число.

`updateCount = unit16 .`

##### Updated Records

Список измененных записей *updates* содержит массив измененных [строк данных](#row-data) *rowData*.

`updates = { rowData }`

В измененных данных строки сохранены:

- значения всех предусмотренных определением системных полей и
- значения только измененных пользовательских полей.

сведения о значении остальных (неизмененных) пользовательских полей в списке отсутствуют.

## View Record Value

Структура записи представления совпадает со структурой [записи](#record). Однако, кроме полей с простейшими данными (числа, строки, байты и т.д.), записи представления могут содержать пользовательские поля комплексных типов `appdef.DataKind_Record` и `appdef.DataKind_Event`. Значения полей этих типов сохранены внутри *userFields* в виде массивов байтов, внутренняя структура которых определена выше в разделах [Record](#record) и [Event](#event).
