## letarette.sql - SQL-based Letarette Document Manager

This is an all-SQL Document Manager for the Letarette full-text search thingy.
To connect a SQL-based primary document storage to Letarette, only two queries need to be supplied.
Check the [example](./example) to see how basic they can be, given that the primary storage has a similar structure.

The **letarette.sql** service is a single binary configured by environment variables.

The following SQL drivers are supported:

* [PostgreSQL](https://github.com/lib/pq)
* [MySQL](https://github.com/go-sql-driver/mysql/)
* [SQLite](github.com/mattn/go-sqlite3)
* [MS SQL Server](https://github.com/denisenkom/go-mssqldb)
* [Athena](github.com/segmentio/go-athena)

### Getting started

The **letarette.sql** service, `lrsql`, needs to know how to connect to the SQL database, and where to find the queries that provides documents to the **Letarette** indexer.

To make the service connect to a PostgreSQL source, and use default values for the rest of the settings:
```sh
export LRSQL_DB_DRIVER="postgres"
export LRSQL_DB_CONNECTION="postgres://user:password@localhost/testdb?sslmode=verify-full"
./lrsql
```

Running `lrsql` with any command-line argument will print out available settings and their default values.

### Service configuration

|*Variable* |*Type* |*Description* |
|---|---|---|
|LRSQL_NATS_URL|String|URL for the NATS service to connect to, defaults to `nats://localhost:4222`.|
|LRSQL_NATS_TOPIC|String|NATS topic prefix for all messages, defaults to `leta`.|
|LRSQL_INDEX_SPACE|String|The index space to serve, defaults to `docs`.|
|LRSQL_DB_DRIVER|String|Database driver name|
|LRSQL_DB_CONNECTION|String|Database connection string|
|LRSQL_SQL_INDEXSQLFILE|String|SQL source file for handling index requests. Default: `indexrequest.sql`.|
|LRSQL_SQL_DOCUMENTSQLFILE|String|SQL source filed for handling document requests. Default: `documentrequest.sql`.|

### The two queries

The **Letarette** indexer update cycle has two separate steps, first it fetches an "interest list" of documents that have updated since the current index position (index request) then it fetches the documents on that list (document request).

The current index position is a combination of a document ID and when that document was last updated.
To handle the situation where the index position document has updated since it was last fetched, *index request* queries need to use strict document ordering. This is best handled by sorting primarily on *update timestamp* and secondarily on *document ID* and handling the case where the *update timestamp* is unchanged separately. See [indexrequest.sql](example/indexrequest.sql) from the example project.

Implementing the *document request* is even easier, since this only needs to retrieve the documents in a list of document IDs: [documentrequest.sql](example/documentrequest.sql).

### Building

**letarette.sql** uses build tags to control which drivers are built into the service binary. The build tags have the same names as the drivers they are enabling.

The current list of driver build tags is:
* postgres
* mysql
* sqlite3
* mssql
* athena

For example, to build a binary with "postgres" and "mysql" support:
```
go build -tags "postgres,mysql"
```


