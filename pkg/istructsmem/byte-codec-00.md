Описание кодека, используемого для представления сущностей core-istructs в байтовом виде
========================================================================================

- [Описание кодека, используемого для представления сущностей core-istructs в байтовом виде](#описание-кодека-используемого-для-представления-сущностей-core-istructs-в-байтовом-виде)
  - [Introduction](#introduction)
  - [Versions](#versions)
    - [Current codec version](#current-codec-version)
    - [Codec version changes summary](#codec-version-changes-summary)
  - [Record](#record)
    - [Codec Version](#codec-version)
    - [Row Data](#row-data)
      - [QName ID](#qname-id)
      - [System fields](#system-fields)
        - [Record ID](#record-id)
        - [Parent ID](#parent-id)
        - [Container](#container)
        - [Is active](#is-active)
      - [User fields](#user-fields)
        - [Buffer Length](#buffer-length)
        - [Buffer Bytes](#buffer-bytes)
  - [Event](#event)
    - [Codec Version](#codec-version-1)
    - [Event Data](#event-data)
      - [QName ID](#qname-id-1)
      - [Create Parameters](#create-parameters)
        - [Partition](#partition)
        - [PLog Offset](#plog-offset)
        - [Work Space](#work-space)
        - [WLog Offset](#wlog-offset)
        - [Register Time](#register-time)
        - [Sync Flag](#sync-flag)
        - [Device ID](#device-id)
        - [Sync Time](#sync-time)
      - [Build Error](#build-error)
        - [Valid Flag](#valid-flag)
        - [Error Message Length](#error-message-length)
        - [Error Message Text](#error-message-text)
        - [Source QName length](#source-qname-length)
        - [Source QName string](#source-qname-string)
        - [Raw Event Data Length](#raw-event-data-length)
        - [Raw Event Data Bytes](#raw-event-data-bytes)
      - [Command Argument](#command-argument)
        - [Argument Object](#argument-object)
          - [Argument Element](#argument-element)
        - [Unlogged Argument Object](#unlogged-argument-object)
      - [Command CUDs](#command-cuds)
        - [Create Records Count](#create-records-count)
        - [Created Records](#created-records)
        - [Update Record Count](#update-record-count)
        - [Updated Records](#updated-records)
  - [View Record Value](#view-record-value)

## Introduction
Сущности core-istructs (Записи, События, Представления) в БД хранятся в виде массива байтов.
Кодек — набор алгоритмов представления хранимых сущностей в виде массивов байтов, используемый в конкретной версии приложения.

## Versions
Кодек имеет свою версию. Номер версии кодека — целое беззнаковое однобайтовое значение (*byte*). Более поздней версии кодека соответствует больший номер версии. Кодек одной версии может использоваться приложением разных версий. Приложение любой версии должно уметь записывать данные в последней (на момент выпуска приложения) версии кодека, и читать данные, закодированные этой или любой более ранней версией кодека.

### Current codec version
Текущая версия кодека: `0` .

### Codec version changes summary
Начальная версия кодека.

## Record

Запись *record* содержит версию кодека *codecVer* и строку данных *rowData*.

`record = codecVer rowData .`

### Codec Version
Версия кодека *codecVer* записана как беззнаковое однобайтовое целое число.

`codecVer = byte . // (0x00)`

### Row Data
Строка данных *rowData* содержит идентификатор имени определения *qNameID*, значения системных полей *sysFields* и значения полей с пользовательскими данными *userFields*.

`rowData = qNameID [sysFields] [userFields] .`

Если имя определения `QName` у строки данных не указано (т.е. равно `appdef.NullQName`), то остальные сущности (*sysFields* и *userFields*) отсутствуют.

#### QName ID
Идентификатор имени *qNameID* записывается как беззнаковое двух-байтовое число. Здесь и далее, если не оговорено специально, для записи числовых значений используется запись с порядком байтов от старшего к младшему (`BigEndian`).

`qNameID = uint16 . // BigEndian`

#### System fields
Системные поля включают идентификатор записи *sys.ID*, идентификатор владеющей записи *sys.ParentID*, контейнер *sys.Container* и признак активности *sys.IsActive*.

`systemFields = [sys.ID] [sys.ParentID] [sys.Container] [sys.IsActive].`

##### Record ID
Идентификатор записи предусмотрен определениями документов (`CDoc`, `GDoc`, `ODoc`, `WDoc`) и определениями записей (`CRecord`, `GRecord`, `ORecord`, `WRecord`).

Идентификатор записи *sys.ID*, если он предусмотрен определениям, записывается как беззнаковое четырех-байтовое число.

`sys.ID = uint64 . // BigEndian`

##### Parent ID
Идентификатор владеющей записи (документа) предусмотрен определениями записей (`CRecord`, `GRecord`, `ORecord`, `WRecord`).

Идентификатор владеющей записи *sys.ParentID*, если он предусмотрен определениями, записывается как беззнаковое четырех-байтовое число.

`sys.ParentID = uint64 . // BigEndian`

##### Container
Идентификатор контейнера записи (элемента) предусмотрен определениями записей (`CRecord`, `GRecord`, `ORecord`, `WRecord`) и определением элемента (`Element`).

Идентификатор контейнера *sys.Container*, если он предусмотрен определением, записывается как беззнаковое двух-байтовое число.

`sys.Container = unit16 . // BigEndian`

##### Is active
Признак активности предусмотрен определениями документов (`CDoc`, `GDoc`) и записей (`CRecord`, `GRecord`).

Признак активности *sys.IsActive*, если он предусмотрен определением, записывается как булевское значение (1 байт).

`sys.IsActive = bool .`


#### User fields
Значения пользовательских полей *userFields* сохраняются внутри [dynoBuffer](https://github.com/untillpro/dynobuffers)-а и содержат длину буфера *bufferLength* и данные буфера *buffer*.

`userFields = bufferLength [ buffer ] . `

Если длина буфера `0` (ноль), то данные после длины отсутствуют.

##### Buffer Length
Длина буфера *bufferLength* записывается как беззнаковое четырех-байтовое число.

`bufferLength = uint32 .`

##### Buffer Bytes
Данные буфера *buffer* представляют собой последовательность из одного или более байтов.

`buffer = byte { byte } . // [bufferLength]byte`

Внутренняя структура буфера определяется схемой [dynoBuffer](https://github.com/untillpro/dynobuffers).
В буфере содержатся данные всех заполненных пользовательских полей. Сведения о значениях предусмотренных определеним системных полей (`sys.QName`, `sys.ID`, `sys.ParentID`, `sys.Container`) в буфере отсутствуют.

## Event

Буфер события *event* содержит версию кодека *codecVer* и данные события *eventData*.

`event = codecVersion eventData .`

### Codec Version
Версия кодека *codecVersion* записана как беззнаковое однобайтовое целое число.

`codecVersion = byte . // (0x00)`

### Event Data
Данные события *eventData* содержат идентификатор имени определения *qNameID*, конструкционные данные события *createParameters*, сведения об ошибке сборки события *buildError*, аргументы команды *commandArguments* и план изменяемых записей *commandCUDs*.

`eventData = qNameID [createParameters buildError commandArguments commandCUDs] .`

Если имя определения (QName) у события не указано (т.е. равно `appdef.NullQName`), то больше ничего для события не указывается.

#### QName ID
Идентификатор имени *qNameID* записывается как беззнаковое двух-байтовое число.

`qNameID = uint16 .`

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
Данные об ошибке сборки события *buildError* содержат информацию об ошибке, произошедшей во время сборки события.

`buildError = valid [ ( messageLength [ messageBytes ] ) ( qNameLength [ qNameChars ] ) ( [ rawDataLength [ rawDataBytes ] ] ) ] .`

##### Valid Flag
Флаг успешной сборки *valid* записан как булевское значение (1 байт).

`valid = bool .`

Флаг успешной сборки *valid* содержит информацию, прошло ли сообщение сборку успешно (`true`) или же при сборке возникла ошибка (`false`). Если записана истина (`true`), то остальные члены, описывающие ошибку сборки, отсутствуют.

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
Объект *object* содержит данные объекта в формате [строка данных](#row-data) *rowData*, количество дочерних элементов *childCount* и список дочерних элементов *element*.

`object = rowData [ childCount { element } ] .`

Если имя определения `QName` у объекта не указано (т.е. равно `appdef.NullQName`), то остальные члены объекта отсутствуют.

###### Argument Element
Элемент *element* содержит данные элемента в формате [строки данных](#row-data) *rowData*, количество дочерних элементов *childCount* и список дочерних элементов *element*.

`element = rowData [ childCount { element } ] .`

Если имя определения `QName` у элемента не указано (т.е. равно `appdef.NullQName`), то остальные члены элемента отсутствуют.

##### Unlogged Argument Object
Секретный объект *unloggedObject* содержит маскированные данные объекта в формате строки данных *rowData*, количество дочерних элементов *childCount* и список дочерних элементов *unloggedElement* так же с маскированными данными.

`unloggedObject = rowData [ childCount { unloggedElement } ] .`

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
