## General-Purpose Data Types
| Data Type (sql-2016)    | Aliases                      | Description                                                    |
| ----------------------- | ---------------------------- | -------------------------------------------------------------- |
| character varying [(n)] | varchar [(n)], text [(n)]    | variable-length character string. n is 1..65535, default 255   |
| binary varying [(n)]    | varbinary [(n)], bytes [(n)] | variable-length binary data. n is 1..65535, default 255        |
| bigint                  | int64                        | signed eight-byte integer                                      |
| integer                 | int, int32                   | signed four-byte integer                                       |
| real                    | float, float32               | single precision floating-point number (4 bytes)               |
| double precision        | float64                      | double precision floating-point number (8 bytes)               |
| timestamp               |                              | date and time (no time zone)                                   |
| decimal                 | money, currency              | currency amount, accurate to a ten-thousandth of the units     |
| boolean                 | bool                         | logical Boolean (true/false)                                   |
| binary large object     | blob                         | reference to the binary data in the blobber app (RecordID)     |

## Voedger-Specific Data Types
| Data Type (voedger)     | Aliases                      | Description                                                    |
| ----------------------- | ---------------------------- | -------------------------------------------------------------- |
| qualified name          | qname                        | package and entity                                             |
| record                  |                              | record inherited from crecord/orecord/wrecord                  |


## References
- [SQL:2016 or ISO/IEC 9075-2:2016](http://www.sai.msu.su/~megera/postgres/files/sql-2016-json.txt) 
- [PostgreSQL data types](https://www.postgresql.org/docs/current/datatype.html)
- [SQL Server: binary and varbinary](https://learn.microsoft.com/en-us/sql/t-sql/data-types/binary-and-varbinary-transact-sql?view=sql-server-ver16)
