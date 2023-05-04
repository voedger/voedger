### Types

```sql

// ENUM: https://dev.mysql.com/doc/refman/8.0/en/enum.html
ENUM reaction ('accept', 'reject', 'smile', 'sorrow')

// FIELDSET: https://www.w3schools.com/tags/tag_fieldset.asp
FIELDSET common_fields (
    author sys.auto.creator /* `auto` means that field value is set by system */
    created_at sys.auto.created_timestamp_ms
    updated_at sys.auto.updated_timestamp_ms
)
```

### Tables

```sql

/* WITH: https://cassandra.apache.org/doc/latest/cassandra/cql/ddl.html#create-table-statement */

TABLE messages (
    
    /* common_fields */

    author sys.auto.creator /* `auto` means that field value is set by system */
    created_at sys.auto.created_timestamp_ms
    updated_at sys.auto.updated_timestamp_ms

    /* message fields */

    is_channel_message boolean
    text nvarchar(170)
    attachments [10]blob

    /* constraits */

    CONSTRAINT presence (
            is_channel_message                                          /* Visible at least in channel */
            OR (sys.parent.id != 0 AND sys.parent.sys.parent.id = 0)    /* Visible in thread  */
    )

) WITH parent = messages


TABLE reactions (
    /* common_fields */

    author sys.auto.creator /* `auto` means that field value is set by system */
    created_at sys.auto.created_timestamp_ms
    updated_at sys.auto.updated_timestamp_ms

    /* table fields */

    kind reaction
    
    /* constraits */

    UNIQUE (parent, author, kind)
    
) WITH parent = messages

/*  TODO  ??? sys.parent.sys.parent_id = 0 */
/*  TODO  ??? [10]blob https://www.ibm.com/docs/en/db2-for-zos/11?topic=type-arrays-in-sql-statements */
```

### Views

[Subqueries (SQL Server)](https://docs.microsoft.com/en-us/sql/relational-databases/performance/subqueries?view=sql-server-ver15)

```sql
MATERIALIZED VIEW channel_messages AS SELECT is_channel_message
   ,(SELECT kind, COUNT() FROM reactions GROUP BY kind) AS reactions
   ,(SELECT FIRST(10) author DISTINCT(author) FROM reactions) AS firstReactors
   ,(SELECT COUNT() FROM threads) AS replies
   ,(SELECT LAST(3) author DISTINCT(author) FROM messages) AS lastRepliers
FROM messages 
WHERE messages.is_channel_message = true
ORDER BY messages.id


MATERIALIZED VIEW thread_messages AS SELECT messages.id 
   ,(SELECT kind, COUNT() FROM reactions GROUP BY kind) AS reactions
   ,(SELECT FIRST(10) author DISTINCT(author) FROM reactions) AS firstReactors
FROM messages
WHERE messages.parent.id != 0
ORDER BY messages.parent.id, messages.id
```

### Use View

```sql

/*  Read TOP 10 records from channel_messages */

SELECT TOP 10 
    ROWID, 
    is_channel_message,     /* by value, materialized */
    text                    /* by reference, not included into view */
FROM channel_messages

/*  Read TOP 10 records using FRAME */

SELECT FRAME(10, TOP, 0)
    ROWID, 
    is_channel_message,     /* by value, materialized */
    text                    /* by reference, not included into view */
FROM channel_messages


SELECT PREV 10 TOP, NEXT 10 TOP,
    ROWID, 
    is_channel_message,     /* by value, materialized */
    text                    /* by reference, not included into view */
FROM channel_messages

/* Read frame (кадр) around known ROWID-value */

SELECT FRAME(10, <ROWID-value>,  10)
    ROWID, 
    is_channel_message,     /* by value, materialized */
    text                    /* by reference, not included into view */
FROM channel_messages

/* Dereference author  */

SELECT TOP 10 ROWID, is_channel_message, text, firstReactors.author.Name FROM channel_messages


/* Top 10 messages and all reactors where exists a reactor with name John  */

SELECT TOP 10 
    ROWID,
    is_channel_message,
    text, 
    SELECT author.name, author.created_at FROM firstReactors
FROM channel_messages
WHERE EXISTS SELECT FROM firstReactors WHERE author.Name = 'John'


/* Top 10 messages and John reactor where exists a reactor with name John  */

SELECT TOP 10 
    ROWID,
    is_channel_message,
    text, 
    SELECT author.name, author.created_at FROM firstReactors WHERE author.Name = 'John'
FROM channel_messages
WHERE EXISTS SELECT FROM firstReactors WHERE author.Name = 'John'

/* smm  does not like: Top 10 messages and John reactor where exists a reactor with name John  */

SELECT TOP 10 
    ROWID,
    is_channel_message,
    text, 
    SELECT author.name, author.created_at FROM firstReactors
FROM channel_messages
WHERE firstReactors.author.Name = 'John'

/* Top 10 messages, only John reactions are shown*/

SELECT TOP 10 
    ROWID,
    is_channel_message,
    text, 
    SELECT author.name, author.created_at FROM firstReactors WHERE author.Name = 'John'
FROM channel_messages

```

#### graphql

```graphql
channel_messages{
    id
    is_channel_message
    text
    reactions{
        kind
        count
    }
    firstReactors{
        author(name: "John"){
            id
            name
            created_at
        }
    }
    replies
    lastRepliers{
        author{
            id
            name
            created_at
        }
    }
}

thread_messages{
    id
    is_channel_message
    text
    reactions{
        kind
        count
    }
    firstReactors{
        author{
            id
            name
            created_at
        }
    }
}

```

gojq -M .rows.[].firstReactors.[].author.Name
```json
{
  "rows": [
   {
        "ROWID":"asjk:123",
        "is_channel_message": true,
        "text": "my message",
        "firstReactors": [
            {
                "author": {
                    "Name": "John"
                }
            },
            {
                "author": {
                    "Name": "Jack"
                }
            }
        ]
    }
  ]
}
```

