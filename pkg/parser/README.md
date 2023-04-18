# TODO: Semantic analysis
- DO not mix named and unnamed params in functions, commands and queries
- param of functions specified in projectors
- projector ON in (COMMAND, COMMANDARGUMENT, ...)
- command and query arguments are the same with function
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

# References
- https://cgsql.dev/cql-guide/int02/

