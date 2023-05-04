# AppSchemas

## Contents
- [Common](#common)
- [Workspaces](#workspaces)
- [Data Types](#data-types)
- [DDL Statements](#ddl-statements)
    - [Sequences](#sequences)
    - [Types](#types)
    - [Tables](#tables)
    - [Singletones](#singletones)
    - [Triggers](#triggers)
    - [Commands](#commands)
    - [Queries](#queries)
    - [Views](#views)
    - [ACL](#acl)
- [Notes](#notes)
- [Documentation](#documentation)
- [See Also](#see-also)

## Common
### SchemaFile Syntax:
```sql
package_statement

[import_statement [...]]

[{root_ddl_statement | normal_ddl_statement} [...]]
```
where:
```
package_statement = SCHEMA name
import_statement = IMPORT SCHEMA path [AS alias]
root_ddl_statement = 
    template_statement
normal_ddl_statement = 
    role_statement | workspace_statement | type_statement | 
    function_statement | sequence_statement | table_statement | 
    comment_statement | tag_statement
    
```

## Workspaces
### Syntax
```sql
WORKSPACE name (
    [ { normal_ddl_statement | workspace_ddl_statement} [...] ]
);
```

where:
```
workspace_ddl_statement = 
    grant_statement | command_statement | query_statement | view_statement |
    projector_statement | use_table_statement  | rate_statement |
    alter_table_statement | alter_query_statement | alter_command_statement

```

## Data Types
```sql
data_type = ID | OFFSET | INT | INT32 | INT64 | FLOAT | FLOAT32 | FLOAT64 | QNAME | TEXT | BOOLEAN
-- INT = INT32
-- FLOAT = FLOAT32
```

## DDL Statements

### Sequences
Syntax:
```sql
SEQUENCE sequence_name AS data_type [WITH { optname[=optvalue] [AND ...] }];
```

Example:
```sql
SEQUENCE aricle_numbers as int;
```

### Types
#### Syntax
```sql
-- Create composite type
TYPE name AS ( attribute [, ... ] ) 
    [WITH { optname[=optvalue] [AND ...] }];

attribute = attribute_name { data_type | type_name }
    [NOT NULL] 
    [DEFAULT const_value ] 
    [REFERENCES table_name] 
    [CHECK(condition | 'regexp')]

-- Create enumeration
TYPE name AS ENUM
    ( [ 'label' [, ... ] ] ) 
    [WITH { optname[=optvalue] [AND ...] }];
```

Example:
```sql
-- Create composite type
TYPE HasNameAndNumber AS (number int, name text);

-- Create enumeration
TYPE Reaction AS ENUM ('accept', 'reject', 'smile', 'sorrow');


#### Arrays
???

-- TYPE Weight AS (brutto int, netto int) CHECK (brutto <= netto);

```


### Tables

#### Principles
- table can inherit from one or more composite types. Table kind (CDOC, WDOC, ODOC) is a mandatory composite type providing 'sys.ID' field and 'sys.IsActive' for CDOC. Table kind is specified for root table only.
- nested tables supported

#### Syntax
```sql
TABLE table_name [OF composite_type_name [, ...]] (
    { field_expr | inner_table_expr | table_constraint [, ...] }   
) [WITH { optname[=optvalue] [AND ...] }];

where:    
field_expr = field_name { data_type | type_name }
            [NOT NULL] 
            [VERIFIABLE]
            [DEFAULT const_value | NEXTVAL('sequence_name')] 
            [REFERENCES table_name] 
            [CHECK(condition | 'regexp')]

table_constraint = CHECK(condition) | UNIQUE (field_name (',' field_name)*) 
```
optname supported:
- Description
- Tags

Example:
```sql
TABLE articles OF CDOC, HasNameAndNumber (
    article_number  int NOT NULL DEFAULT NEXTVAL('article_numbers') CHECK(article_number>0),
    barcode         text NOT NULL,
    ean13barcode    text CHECK('^[0-9]{13}$')
    
    UNIQUE(article_number),

    TABLE article_prices OF IdName (
        id_articles     int64 REFERENCES articles,
        id_prices       int64 REFERENCES prices,
        price           float32 DEFAULT 1.00,
        UNIQUE(id_articles, id_prices)
    )
) WITH Description='Information about article';
```

#### Limitations
- References 
    - CDOC table can only reference to CDOC tables
- Nested tables cannot override table kind

### Singletones
#### Principles:
- Singletone is always CDOC table under the hood
- Singletone fields may only refer to other CDoc tables

```sql
SINGLETONE table_name [OF composite_type_name [, ...]] (
    { field_expr | table_constraint [, ...] }   
) [WITH { optname[=optvalue] [AND ...] }];

where:    
field_expr = field_name { data_type | type_name }
            [NOT NULL] 
            [VERIFIABLE]
            [DEFAULT const_value | NEXTVAL('sequence_name')] 
            [REFERENCES table_name] 
            [CHECK(condition | 'regexp')]

table_constraint = CHECK(condition) | UNIQUE (field_name (',' field_name)*) 
```

### Triggers
#### Principles

- BEFORE = validator
- AFTER = projectors

#### Syntax
```sql
TRIGGER name { BEFORE|AFTER } 
    { event [ OR ...] } 
    ON { table_name | command_name }
    EXECUTE PROCEDURE function_name 
    ENGINE { WASM | BUILTIN }
    [WITH { optname[=optvalue] [AND ...] }];

where:
event = INSERT | UPDATE
```

#### Example Validator:
```sql
TRIGGER ValidateArticle BEFORE INSERT OR UPDATE ON articles EXECUTE PROCEDURE air.ValidateArticle ENGINE WASM;
```

####  Example Projector:
```sql
TRIGGER AirDashboardProjector AFTER INSERT ON air.PBill 
EXECUTE PROCEDURE air.DashboardProjector ENGINE WASM
WITH HandleErrors=true AND Description='';
```

#### Notes
- `WITH HandleErrors=true` - only for `AFTER`, indicates that events with errors must be handled

### Commands

#### Syntax
```sql
COMMAND name ([[argname] argtype [, ...]]) 
    [RETURNS argtype]
    ENGINE { WASM | BUILTIN }
    [WITH { optname[=optvalue] [AND ...] }];

argtype = "sys.Json" | data_type | type_name | table_name
```

#### Examples
```sql
COMMAND mycommand(untill.pbill) RETURNS sys.Json ENGINE WASM 
WITH Description='This is my first command' AND Rate='1/HOUR';
```

### QUERIES
#### Syntax
```sql
QUERY name ([[argname] argtype [, ...]]) 
    RETURNS argtype
    ENGINE { WASM | BUILTIN }
    [WITH { optname[=optvalue] [AND ...] }];

argtype = "sys.Json" | data_type | type_name | table_name
```
#### Examples
```sql
QUERY myquery(untill.pbill) RETURNS sys.Json ENGINE WASM 
WITH Description='This is my first query' AND Rate='100/MINUTE';
```


### Views
#### Principles
- Views are always "materialized"
- `AS SELECT...` means that a Projector will be created by core which "meterializes" it.

#### Syntax
```sql
VIEW name(column_name[, ...]) 
    [AS {SELECT ... | RESULT OF projector_name}  ]
    [WITH { optname[=optvalue] [AND ...] }];
```

#### Examples
```sql
VIEW channel_messages(is_channel_message, reactions, firstReactors, replies, lastRepliers) 
AS SELECT is_channel_message
   ,(SELECT kind, COUNT() FROM reactions GROUP BY kind)
   ,(SELECT FIRST(10) author DISTINCT(author) FROM reactions)
   ,(SELECT COUNT() FROM threads)
   ,(SELECT LAST(3) author DISTINCT(author) FROM messages)
FROM messages 
WHERE messages.is_channel_message = true
ORDER BY messages.id;


VIEW XZReports(
    Year int32, 
    Month int32, 
    Day int32, 
    Kind int32, 
    Number int32, 
    XZReportWDocID id,
    PRIMARY KEY((Year), Month, Day, Kind, Number)
) AS RESULT OF UpdateXZReportsView;
```

### ACL
#### Syntax
```sql
GRANT {{ SELECT | INSERT | UPDATE } | ALL } ON { TABLE [WITH TAG tagname] } TO role_name
GRANT {{ EXECUTE } | ALL } ON { COMMAND | QUERY [WITH TAG tagname] } TO role_name
```

## Notes
- ??? Support for multiline strings

## Documentation
- PostgreSQL syntax
    - https://www.postgresql.org/docs/current/ddl-basics.html    
    - https://www.postgresql.org/docs/current/sql-createtable.html
    - https://www.postgresql.org/docs/current/sql-creatematerializedview.html
    - https://www.postgresql.org/docs/15/sql-createfunction.html
- [Oracle: Types of SQL Statements](https://docs.oracle.com/cd/B14117_01/server.101/b10759/statements_1001.htm)
- [Cassandra: create table WITH table_options](https://docs.datastax.com/en/cql-oss/3.3/cql/cql_reference/cqlCreateTable.html#table_options)
## See Also
- https://github.com/heeus/inv-go/blob/master/20220221-parsing/participle/schema.sql
- https://github.com/heeus/heeus-design/blob/main/20220414-slack-wdocs/slack-sql-syntax.md
- https://github.com/heeus/core/blob/ea31af585e5519673be8ff6e489d1afabe6364d8/istructsmem/schema-utils.go#L90
- [API v2](https://dev.heeus.io/launchpad/#!23905)