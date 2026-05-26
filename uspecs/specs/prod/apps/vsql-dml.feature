Feature: VSQL DML

  VADeveloper reads records and views via SELECT, and VAOperator changes
  state via UPDATE / INSERT against tables, views and singletons, across
  workspaces and applications, with full ACL enforcement.

  Background:
    Given VADeveloper is authenticated in workspace "test_ws" of app "test1/app1"
    And VAOperator holds the system principal of app "sys/cluster"

  Rule: SELECT from a table

    Scenario: Read a single record by ID returns all readable fields
      Given record "app1pkg.payments.<id>" exists with "name=EFT" and "guid=guidEFT"
      When VADeveloper executes "select * from app1pkg.payments where id = <id>"
      Then response status is "200 OK"
      And result contains "sys.QName=app1pkg.payments", "sys.ID=<id>", "name=EFT", "guid=guidEFT", "sys.IsActive=true"

    Scenario: Read selected fields from records by ID range
      Given records "app1pkg.payments.<id1>" and "app1pkg.payments.<id2>" exist
      When VADeveloper executes "select name, sys.IsActive from app1pkg.payments where id in (<id1>, <id2>)"
      Then response status is "200 OK"
      And result contains a row per matching ID with only the requested fields

    Scenario: A non-singleton table requires at least one record ID
      When VADeveloper executes "select * from app1pkg.payments"
      Then response status is "400 Bad Request"
      And error message is "'app1pkg.payments' is not a singleton. At least one record ID must be provided"

    Scenario Outline: WHERE clause validation errors
      When VADeveloper executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                       | error                                  |
        | select * from app1pkg.payments where something = 1                          | unsupported column name: something     |
        | select * from app1pkg.payments where id = 2.3                               | parsing "2.3": invalid syntax          |
        | select * from app1pkg.payments where id >= 2                                | unsupported operation: >=              |
        | select * from app1pkg.payments where id = 2 and something = 2              | unsupported expression                 |
        | select * from app1pkg.payments.2 where id = 2                               | record ID and 'where id ...' clause can not be used in one query |
        | select name, name from app1pkg.payments where id = 1                        | field "name" is selected more than once |
        | select abracadabra from app1pkg.pos_emails where id = 1                     | not found: field «abracadabra»         |

    Scenario: Record-ID type mismatch
      Given "app1pkg.pos_emails.<id>" exists
      When VADeveloper executes "select * from app1pkg.payments where id = <id>"
      Then response status is "400 Bad Request"
      And error message is "record with ID '<id>' has mismatching QName 'app1pkg.pos_emails'"

    Scenario: Non-existing record ID
      When VADeveloper executes "select * from app1pkg.payments where id = <missingID>"
      Then response status is "400 Bad Request"
      And error message is "record with ID '<missingID>' not found"

  Rule: SELECT from an ODoc or ORecord

    Scenario Outline: Read an ODoc or ORecord by single ID
      Given command "c.app1pkg.CmdODocOne" emitted ODoc "<odocID>" and child ORecord "<orecordID>"
      When VADeveloper executes "<query>"
      Then response status is "200 OK"
      And result contains "sys.ID=<id>", "<field>=<value>"

      Examples:
        | query                                          | id          | field            | value |
        | select * from app1pkg.odoc1.<odocID>           | <odocID>    | odocIntFld       | 42    |
        | select * from app1pkg.orecord1.<orecordID>     | <orecordID> | orecord1IntFld   | 43    |

    Scenario Outline: Read multiple ODocs or ORecords by ID range
      When VADeveloper executes "select * from <doc> where id in (<id1>, <id2>)"
      Then response status is "200 OK"
      And result contains two rows, one per requested id

      Examples:
        | doc                |
        | app1pkg.odoc1      |
        | app1pkg.orecord1   |

  Rule: SELECT from a view

    Scenario: Read a view filtered by full partition key
      Given view record "sys.CollectionView" with "PartKey=1" and "DocQName=app1pkg.payments" exists
      When VADeveloper executes "select * from sys.CollectionView where PartKey = 1 and DocQName = 'app1pkg.payments'"
      Then response status is "200 OK"
      And result contains "DocQName=app1pkg.payments", "PartKey=1"

    Scenario Outline: WHERE supports any integer key kind
      Given view "<view>" with key field "<field>" of kind "<kind>"
      When VADeveloper executes "select * from <view> where <where>"
      Then response status is "200 OK"
      And matching view records are returned

      Examples:
        | view                    | field | kind  | where                                  |
        | app1pkg.DailyIdxSmall   | Year  | int16 | Year = <year>                          |
        | app1pkg.DailyIdxSmall   | Month | int8  | Year = <year> and Month = 3 and Day = 7 |

    Scenario: Lower-cased table and field names are recovered
      When VADeveloper executes "select docqname from sys.collectionview where PartKey = 1 and DocQName = 'app1pkg.payments'"
      Then response status is "200 OK"

    Scenario Outline: Unsupported WHERE on a view
      When VADeveloper executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                          | error                                  |
        | select * from sys.CollectionView where partKey > 1                              | unsupported operator: >                |
        | select * from sys.CollectionView where partKey = 1 or docQname = 'app1pkg.payments' | unsupported expression: *sqlparser.OrExpr |
        | select abracadabra from sys.CollectionView where PartKey = 1                    | not found: field «abracadabra»         |

    Scenario: Filter a view by a QName-typed key field
      Given view "app1pkg.ViewWithQName" carries a key field "QName" of QName kind
      When VADeveloper executes "select * from app1pkg.ViewWithQName where IntFld = 42 and QName = 'app1pkg.category'"
      Then response status is "200 OK"

  Rule: SELECT from a singleton

    Scenario: Read a singleton without conditions
      When VADeveloper executes "select sys.QName from app1pkg.test_ws"
      Then response status is "200 OK"
      And result is "{\"sys.QName\":\"app1pkg.test_ws\"}"

    Scenario Outline: Conditions on a singleton are rejected
      When VADeveloper executes "<query>"
      Then response status is "400 Bad Request"
      And error message is "conditions are not allowed to query a singleton"

      Examples:
        | query                                              |
        | select sys.QName from app1pkg.test_ws.1            |
        | select sys.QName from app1pkg.test_ws where id = 1 |

  Rule: SELECT from system logs

    Scenario: Default offset and limit on sys.plog
      When VADeveloper executes "select * from sys.plog"
      Then response status is "200 OK"
      And result contains at most "DefaultLimit" events starting at the first offset

    Scenario: Read all events with "limit -1"
      When VADeveloper executes "select * from sys.plog limit -1"
      Then response status is "200 OK"
      And every plog event is returned

    Scenario Outline: Offset filters on sys.plog and sys.wlog
      When VADeveloper executes "<query>"
      Then response status is "200 OK"
      And result contains "<count>" event(s)

      Examples:
        | query                                              | count |
        | select * from sys.plog limit 1                     | 1     |
        | select * from sys.plog where Offset > <lastOff-1>  | 1     |
        | select * from sys.plog where Offset >= <lastOff-1> | 2     |
        | select * from sys.wlog limit 1                     | 1     |
        | select * from sys.wlog where Offset > <lastOff-1>  | 1     |

    Scenario: Only SELECT is allowed on a log
      When VADeveloper executes "update sys.plog set a = 1"
      Then response status is "400 Bad Request"
      And error message is "'select' operation is expected"

    Scenario: Read CUDs payload from sys.plog after a c.sys.CUD post
      Given a c.sys.CUD posted "app1pkg.payments" with "guid=<guid>" at plog offset "<plogOffset>"
      When VADeveloper executes "select CUDs from sys.plog where Offset >= <plogOffset>"
      Then response status is "200 OK"
      And result contains the CUD payload with "<guid>"

    Scenario: Read sys.RawArg body from sys.wlog
      Given command "c.app1pkg.TestCmdRawArg" with raw body "hello world" produced wlog event at "<wlogOffset>"
      When VADeveloper executes "select * from sys.wlog where Offset > <wlogOffset-1>"
      Then response status is "200 OK"
      And "ArgumentObject.Body" equals "hello world"

    Scenario Outline: Log query parameter validation
      When VADeveloper executes "<query>"
      Then response status is "500 Internal Server Error"
      And error message contains "<error>"

      Examples:
        | query                                              | error                                       |
        | select * from sys.plog limit 7.1                   | strconv.ParseInt: parsing "7.1"             |
        | select * from sys.plog limit -3                    | limit must be greater than -2               |
        | select * from sys.plog where Offset >= 2.1         | strconv.ParseUint: parsing "2.1"            |
        | select * from sys.plog where Offset >= 0           | offset must be greater than zero            |
        | select * from sys.plog where Offset < 2            | unsupported operation: <                    |
        | select * from sys.plog where something >= 1        | unsupported column name: something          |
        | select * from sys.wlog where Offset >= 1 and something >= 5 | unsupported expression: *sqlparser.AndExpr |

  Rule: SELECT across workspaces and applications

    Scenario: Read sys.wlog of another workspace by WSID
      Given workspace "<otherWSID>" of the same app holds wlog events
      When VADeveloper executes "select * from <otherWSID>.sys.wlog"
      Then response status is "200 OK"
      And result contains events belonging to "<otherWSID>"

    Scenario: Read a record from another app by fully-qualified location
      Given workspace "<wsidApp1>" of app "test1/app1" holds record "app1pkg.category.<categoryID>"
      And VADeveloper is authenticated in workspace of app "test1/app2"
      When VADeveloper executes "select * from test1.app1.<wsidApp1>.app1pkg.category.<categoryID>"
      Then response status is "200 OK"
      And the foreign-app record is returned

    Scenario: Read from a registry app workspace by number
      Given VAOperator holds the system principal of app "test1/app1"
      When VAOperator executes "select * from sys.registry.a<appWSNumber>.registry.Login where id = <loginID>"
      Then response status is "200 OK"
      And result contains the LoginHash of "<login>"

    Scenario: Read from a registry app workspace by login hash
      Given VAOperator holds the system principal of app "test1/app1"
      When VAOperator executes 'select * from sys.registry."<login>".registry.Login where id = <loginID>'
      Then response status is "200 OK"
      And result contains the LoginHash of "<login>"

    Scenario: Foreign-app read without the system token is forbidden
      When VADeveloper executes "select * from sys.registry.a<appWSNumber>.registry.Login where id = <loginID>"
      Then response status is "403 Forbidden"

    Scenario: Read from a non-initialized workspace is forbidden
      When VADeveloper executes "select * from <nonExistingWSID>.sys.wlog"
      Then response status is "403 Forbidden"
      And error message is "workspace is not initialized"

    Scenario: Unsupported data source
      When VADeveloper executes "select * from git.hub"
      Then response status is "400 Bad Request"
      And error message is "do not know how to read from the requested git.hub"

  Rule: SELECT ACL on documents, fields, and system fields

    Background:
      Given the workspace schema declares
        """
        GRANT SELECT, INSERT, UPDATE, ACTIVATE, DEACTIVATE ON ALL TABLES WITH TAG WorkspaceOwnerTableTag TO sys.WorkspaceOwner;
        GRANT SELECT ON ALL VIEWS WITH TAG WorkspaceOwnerTableTag TO sys.WorkspaceOwner;

        -- TestDeniedCDoc has no SELECT grant to any role
        GRANT SELECT(Fld1) ON TABLE app1pkg.TestCDocWithDeniedFields TO sys.WorkspaceOwner;

        GRANT SELECT ON TABLE app1pkg.TestCDocSysIDRevoked TO sys.WorkspaceOwner;
        REVOKE SELECT(sys.ID) ON TABLE app1pkg.TestCDocSysIDRevoked FROM sys.WorkspaceOwner;

        GRANT SELECT(Year, Month, AllowedFld) ON VIEW app1pkg.TestViewWithDeniedFields TO sys.WorkspaceOwner;

        PUBLISHED ROLE app1pkg.ApiRole;
        GRANT EXECUTE ON QUERY sys.SqlQuery TO app1pkg.ApiRole;
        GRANT SELECT ON TABLE app1pkg.category TO app1pkg.ApiRole;
        -- ApiRole has no SELECT on odoc1, orecord1, TestCDocWithDeniedFields
        """
      And VADeveloper is authenticated as workspace owner with role "sys.WorkspaceOwner"

    Scenario: Document-level denial when no SELECT grant exists
      When VADeveloper executes "select * from app1pkg.TestDeniedCDoc.<id>"
      Then response status is "403 Forbidden"

    Scenario Outline: A denied field in WHERE forces 403 regardless of projection
      When VADeveloper executes "<query>"
      Then response status is "403 Forbidden"

      Examples:
        | query                                                                                              |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where DeniedFld2 = 1                         |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where Fld1 = 1 and DeniedFld2 = 2            |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where Fld1 = 1 or DeniedFld2 = 2             |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where (Fld1 = 1 and (DeniedFld2 = 2 or Fld1 = 3)) |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where DeniedFld2 in (1, 2)                   |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where Fld1 in (DeniedFld2, 2)                |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where DeniedFld2 not in (1, 2)               |
        | select Fld1 from app1pkg.TestCDocWithDeniedFields.123 where TestCDocWithDeniedFields.DeniedFld2 = 1 |
        | select DeniedFld2 from app1pkg.TestCDocWithDeniedFields.123                                        |
        | select * from app1pkg.TestCDocWithDeniedFields.123                                                 |
        | select * from app1pkg.TestViewWithDeniedFields where Year = 2025                                   |

    Scenario: An allowed projection on a denied-field doc reaches business logic
      When VADeveloper executes "select Fld1 from app1pkg.TestCDocWithDeniedFields.123"
      Then response status is "400 Bad Request"
      And error message is "record with ID '123' not found"

    Scenario: Star projection allowed when every field is granted via tag
      Given the workspace schema declares
        """
        TABLE app1pkg.payments INHERITS sys.CDoc (...) WITH Tags=(WorkspaceOwnerTableTag);
        """
      When VADeveloper executes "select * from app1pkg.payments.<id>"
      Then response status is "200 OK"

    Scenario Outline: REVOKE of a system field denies that field even on a full grant
      When VADeveloper executes "<query>"
      Then response status is "403 Forbidden"

      Examples:
        | query                                                       |
        | select sys.ID from app1pkg.TestCDocSysIDRevoked.<id>        |
        | select * from app1pkg.TestCDocSysIDRevoked.<id>             |

    Scenario: A user field stays readable after only a system field is revoked
      When VADeveloper executes "select Fld1 from app1pkg.TestCDocSysIDRevoked.<id>"
      Then response status is "200 OK"

    Scenario: A partial field grant still allows reading system fields
      When VADeveloper executes "select sys.ID, sys.QName, Fld1 from app1pkg.TestCDocWithDeniedFields.<id>"
      Then response status is "200 OK"

    Scenario: ODoc and ORecord are 403 under an API token role that has no SELECT
      Given VADeveloper is authenticated via an API token issued for role "app1pkg.ApiRole"
      When VADeveloper executes "select * from app1pkg.odoc1.<id>"
      Then response status is "403 Forbidden"
      When VADeveloper executes "select * from app1pkg.orecord1.<id>"
      Then response status is "403 Forbidden"

    Scenario: Foreign-app SELECT under an API token is 403
      Given VADeveloper is authenticated via an API token issued for role "app1pkg.ApiRole"
      When VADeveloper executes "select * from sys.registry.a<appWSNumber>.registry.Login where id = <loginID>"
      Then response status is "403 Forbidden"

  Rule: UPDATE a record in a table

    Scenario: Logged update of a record by ID
      Given record "app1pkg.category.<categoryID>" exists in workspace "<wsid>" of app "test1/app1"
      When VAOperator executes "update test1.app1.<wsid>.app1pkg.category.<categoryID> set name = 'NewName'"
      Then response status is "200 OK"
      And subsequent "select * from app1pkg.category where id = <categoryID>" returns "name=NewName"
      And the routing log records the rerouting from "c.cluster.VSqlUpdate" to "q.cluster.VSqlUpdate2"

    Scenario: Logged update returns positive WLog and CUD offsets
      When VAOperator executes "update test1.app1.<wsid>.app1pkg.category.<categoryID> set name = 'X'"
      Then response status is "200 OK"
      And "LogWLogOffset" is positive
      And "CUDWLogOffset" is positive

    Scenario: Unlogged update of a record by ID writes new field values
      Given record "app1pkg.category.<categoryID>" exists with "name=N0", "int_fld1=42", "int_fld2=43", "hq_id=hq"
      When VAOperator executes "unlogged update test1.app1.<wsid>.app1pkg.category.<categoryID> set name = 'N1', cat_external_id = 'cv', int_fld1 = 44"
      Then response status is "200 OK"
      And subsequent select returns "name=N1, cat_external_id=cv, int_fld1=44, int_fld2=43, hq_id=hq"

    Scenario: Logged update round-trips every scalar field kind
      Given record "app1pkg.DocManyTypes.<id>" exists with "Int=1, Int64=2, Float32=3.4, Float64=5.6, Str=str, Bool=true, Bytes=0x<oldHex>"
      When VAOperator executes "update test1.app1.<wsid>.app1pkg.DocManyTypes.<id> set Int = 7, Int64 = 8, Float32 = 9.1, Float64 = 10.2, Str = 'str1', Bool = false, Bytes = 0x<newHex>"
      Then response status is "200 OK"
      And subsequent "select * from app1pkg.DocManyTypes where id = <id>" returns "Int=7, Int64=8, Float32=9.1, Float64=10.2, Str=str1, Bool=false, Bytes=base64(<newHex>)"

    Scenario Outline: Unlogged update on a record - error cases
      When VAOperator executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                                                       | error                                  |
        | unlogged update test1.app1.<wsid>.app1pkg.category.<nonExistingID> set int_fld1 = 44                          | record ID <nonExistingID> does not exist |
        | unlogged update test1.app1.<wsid>.app1pkg.category.<categoryID> set unknownField = 44                         | unknownField                           |

  Rule: UPDATE a view

    Scenario: Unlogged update of a view by full key
      Given view record "app1pkg.CategoryIdx" exists with key "IntFld=43, Dummy=1" and "Name=N0, Val=42"
      When VAOperator executes "unlogged update test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'N1' where IntFld = 43 and Dummy = 1"
      Then response status is "200 OK"
      And subsequent "select * from app1pkg.CategoryIdx where IntFld = 43 and Dummy = 1" returns "Name=N1, Val=42"

    Scenario Outline: Unlogged update of a view - error cases
      When VAOperator executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                                                       | error                       |
        | unlogged update test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'X' where IntFld = 43                       | field is empty: Dummy       |
        | unlogged update test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'X' where IntFld = 1 and Dummy = 1          | record not found            |
        | unlogged update test1.app1.<wsid>.app1pkg.CategoryIdx set unexistingField = 'X' where IntFld = 43 and Dummy = 1 | unexistingField            |

  Rule: INSERT into a table or a view

    Scenario: Insert a record into a table
      When VAOperator executes "insert test1.app1.<wsid>.app1pkg.category set name = 'NewCat'"
      Then response status is "200 OK"
      And "NewID" is positive
      And subsequent "select * from app1pkg.category where id = <NewID>" returns "name=NewCat"

    Scenario: Unlogged insert into a view
      Given no view record "app1pkg.CategoryIdx" exists with "IntFld=<intFld>, Dummy=1"
      When VAOperator executes "unlogged insert test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'N1', Val = 123, IntFld = <intFld>, Dummy = 1"
      Then response status is "200 OK"
      And subsequent select by the same key returns "Name=N1, Val=123, IntFld=<intFld>, Dummy=1, offs=0"

    Scenario: Re-inserting an existing view record returns 409 Conflict
      Given view record "app1pkg.CategoryIdx" with "IntFld=<intFld>, Dummy=1" already exists
      When VAOperator executes "unlogged insert test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'X', Val = 1, IntFld = <intFld>, Dummy = 1"
      Then response status is "409 Conflict"
      And error message is "view record already exists"

    Scenario Outline: Unlogged insert into a view - error cases
      When VAOperator executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                                                              | error                                          |
        | unlogged insert test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'N', Val = 1, IntFld = <intFld>                    | is empty: Dummy                                |
        | unlogged insert test1.app1.<wsid>.app1pkg.CategoryIdx set Name = 'N', Val = 1, IntFld = <intFld>, Dummy = 1, Unexisting = 42 | Unexisting                                     |

  Rule: UPDATE a singleton (CDoc / WDoc)

    Scenario: Unlogged update of a singleton by login hash
      Given VAOperator holds the system principal of app "sys/cluster"
      When VAOperator executes 'unlogged update sys.registry."<login>".registry.Login.<loginID> set WSKindInitializationData = ''abc'''
      Then response status is "200 OK"
      And subsequent select returns "WSKindInitializationData=abc"

    Scenario: Unlogged update of a singleton by app workspace number
      When VAOperator executes "unlogged update sys.registry.a<appWSNumber>.registry.Login.<loginID> set WSKindInitializationData = 'def'"
      Then response status is "200 OK"
      And subsequent select returns "WSKindInitializationData=def"

  Rule: UPDATE CORRUPTED on system logs

    Scenario: Mark a wlog event as corrupted
      Given wlog event at offset "<wlogOffset>" exists in workspace "<wsid>" of app "test1/app1"
      When VAOperator executes "update corrupted test1.app1.<wsid>.sys.WLog.<wlogOffset>"
      Then response status is "200 OK"
      And subsequent "select * from sys.wlog where Offset = <wlogOffset>" returns a record with empty ArgumentObject, empty CUDs and Error.ErrStr="corrupted data", Error.QNameFromParams="sys.Corrupted", Error.ValidEvent=false
      And the matching plog event is not modified

    Scenario: Mark a plog event as corrupted
      Given plog event at offset "<plogOffset>" exists in partition "<partitionID>" of app "test1/app1"
      When VAOperator executes "update corrupted test1.app1.<partitionID>.sys.PLog.<plogOffset>"
      Then response status is "200 OK"
      And subsequent "select * from sys.plog where Offset = <plogOffset>" returns a record with empty ArgumentObject, empty CUDs and Error.ErrStr="corrupted data", Error.QNameFromParams="sys.Corrupted", Error.ValidEvent=false
      And the matching wlog event is not modified

  Rule: UPDATE query syntax and semantic validation

    Scenario Outline: Common query format errors
      When VAOperator executes "<query>"
      Then response status is "400 Bad Request"
      And error message contains "<error>"

      Examples:
        | query                                                                              | error                                                |
        |                                                                                    | field is empty                                       |
        | update                                                                             | invalid query format                                 |
        | select * from sys.plog                                                             | 'update' or 'insert' clause expected                 |
        | update test1.app1.42.app1pkg.category.1                                            | no fields to update                                  |
        | update test1.app1.app1pkg.category.1 set name = 's'                                | location must be specified                           |
        | update test1.app1.42.app1pkg.category set name = 42                                | record ID is not provided                            |
        | update test1.app1.42.app1pkg.category.1 set name = 42 where sys.ID = 1             | conditions are not allowed on update                 |
        | update test1.app1.42.app1pkg.category.1 set sys.ID = 1                             | field sys.ID can not be updated                      |
        | update test1.app1.42.app1pkg.category.1 set sys.QName = 'a.b'                      | field sys.QName can not be updated                   |
        | update test1.app1.42.app1pkg.category.1 set x = 1, x = 2                           | field x specified twice                              |
        | update test1.app1.42.unknown.table.1 set x = 1, x = 2                              | qname unknown.table is not found                     |
        | update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = 0x1                        | hex: odd length hex string                           |
        | update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = null                       | null value is not supported                          |
        | update test1.app1.42.app1pkg.DocManyTypes.1 set Bytes = sin(42)                    | unsupported value type                               |
        | update test1.app1.42.app1pkg.MockCmd set Bytes = 0x00                              | CDoc or WDoc only expected                           |
        | insert test1.app1.42.app1pkg.CategoryIdx set Val = 42                              | CDoc or WDoc only expected                           |
        | insert test1.app1.42.app1pkg.category                                              | no fields to set                                     |
        | insert test1.app1.42.app1pkg.category set a = 1 where x = 1                        | conditions are not allowed on insert table           |
        | insert test1.app1.42.app1pkg.category.1 set a = 1                                  | record ID must not be provided on insert table       |
        | insert test1.app1.42.app1pkg.category set a = null                                 | null value is not supported                          |
        | update corrupted                                                                   | invalid query format                                 |
        | update corrupted test1.app1.1.sys.PLog                                             | offset must be provided                              |
        | update corrupted test1.app1.0.sys.WLog.44                                          | wsid must be provided                                |
        | update corrupted test1.app1.1000.sys.PLog.44                                       | provided partno 1000 is out of 5 declared by app test1/app1 |
        | update corrupted test1.app1.1.sys.PLog.0                                           | provided offset or ID must not be 0                  |
        | update corrupted test1.app1.1.sys.PLog.1 set name = 42                             | any params of update corrupted are not allowed       |
        | update corrupted test1.app1.1.app1pkg.category.44                                  | sys.plog or sys.wlog are only allowed                |
        | update corrupted unknown.app.1.sys.PLog.44                                         | application not found: unknown/app                   |
        | unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = 44, Name = 'x'          | full key must be provided on view unlogged update    |
        | unlogged update test1.app1.1.app1pkg.CategoryIdx.42 set a = 2 where x = 1          | record ID must not be provided on view unlogged update |
        | unlogged update test1.app1.1.app1pkg.CategoryIdx set Val = null where IntFld = 43 and Dummy = 1 | null value is not supported                          |
        | unlogged update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x = 1 and x = 1   | key field x is specified twice                       |
        | unlogged update test1.app1.1.app1pkg.CategoryIdx set a = 2 where x > 1             | 'where viewField1 = val1 [and viewField2 = val2 ...]' condition is only supported |
        | unlogged update test1.app1.1.app1pkg.MockCmd set Val = 44, Name = 'x'              | view, CDoc or WDoc only expected                     |
        | unlogged update test1.app1.1.app1pkg.category set a = 2                            | record ID must be provided on record unlogged update |
        | unlogged update test1.app1.1.app1pkg.category.1 set a = 2 where b = 3              | 'where' clause is not allowed on record unlogged update |
        | unlogged insert test1.app1.1.app1pkg.category set Val = 44, Name = 'x'             | unlogged insert is not allowed for records           |
        | unlogged insert test1.app1.1.app1pkg.MockCmd set Val = 44, Name = 'x'              | view, CDoc or WDoc only expected                     |
        | unlogged insert test1.app1.1.app1pkg.CategoryIdx set Val = 44, Name = 'x' where a = 1 | 'where' clause is not allowed on view unlogged insert |
        | unlogged insert test1.app1.1.app1pkg.CategoryIdx set Val = null                    | null value is not supported                          |

