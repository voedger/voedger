## TODO: Semantic analysis
- table OF: 
    - type is correct
    - type is not specified twice
- types in params, tables
- type is correct if:
    - if package is empty: type is declared in the same package or embedded type
- Nested tables cannot override table kind
- CDOC table can only reference to CDOC tables
- field default value corresponds field type
- NEXTVAL = only for integer
- REFERENCES = only for fields of type ...
- UNIQUE:
    - all fields present in table
    - no fields listed twice
- VIEW:
    - PrimaryKey is required
    - Comment is valid
- SEQUENCE:
    - ...

## Detailed Design

- [Compatibility](compat/README.md)



## References
- https://cgsql.dev/cql-guide/int02/

