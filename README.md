[![GitHub release](https://img.shields.io/github/release/erkkah/letarette.sql.svg)](https://github.com/erkkah/letarette.sql/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/erkkah/letarette.sql)](https://goreportcard.com/report/github.com/erkkah/letarette.sql)

## letarette.sql - SQL-based Letarette Document Manager

This is an all-SQL Document Manager for the Letarette full-text search thingy.
To connect a SQL-based primary document storage to Letarette, only two queries need to be supplied.
Check the [example](./example) to see how basic they can be, given that the primary storage has a similar structure.

The following SQL drivers are supported:

* [PostgreSQL](https://github.com/lib/pq)
* [MySQL](https://github.com/go-sql-driver/mysql/)
* [SQLite](github.com/mattn/go-sqlite3)
* [MS SQL Server](https://github.com/denisenkom/go-mssqldb)

### Getting started

The **letarette.sql** service, `lrsql`, needs to know how to connect to the SQL database, and where to find the queries that provide documents to the **Letarette** indexer.

To make the service connect to a PostgreSQL source, and use default values for the rest of the settings:
```sh
export LRSQL_DB_DRIVER="postgres"
export LRSQL_DB_CONNECTION="postgres://user:password@localhost/testdb?sslmode=verify-full"
./lrsql
```

Running `lrsql` with any command-line argument will print out available settings and their default values.

### Service configuration
The **letarette.sql** service is configured by environment variables.

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

The **Letarette** indexer update cycle has two separate steps, first it fetches an "interest list" of documents that are newer than the current index position (index request), and then it fetches the documents on that list (document request).

Timestamps are UTC - referenced UNIX epoch nanoseconds.

#### Index requests

The current index position is a combination of a document ID and the *updated timestamp* of that document.
To handle the situation where the index position document has been updated since it was last fetched, *index request* queries need to follow a strict document ordering. This is best handled by sorting primarily on *update timestamp* and secondarily on *document ID* and handling the case where the *update timestamp* is unchanged separately. See [indexrequest.sql](example/indexrequest.sql) from the example project.

The index request query gets three bound parameters: `afterDocument` (string), `fromTimeNanos` (int64) and `documentLimit` (int) and should return rows of two columns: `id` (string) and `updatedNanos` (int64). The order of the bound parameters can optionally be changed by adding a special `@bind` comment:

```sql
---
--- My index request query.
---
--- Bind order:
--- @bind[documentLimit, afterDocument, fromTimeNanos]
---
```

> Note that the parameters are not bound by name. The `@bind` directive just specifies the order of the parameters for drivers that have generic `?` - type parameter binding.

#### Document requests

Implementing the *document request* is even easier, since this only needs to retrieve all documents for a list of document IDs: [documentrequest.sql](example/documentrequest.sql). The single bound parameter will be replaced with a list of document IDs (string).

The document request query should return rows of `id` (string), `updatedNanos` (int64), `title` (string), `txt` (string) and `alive` (bool).

### Building

**letarette.sql** uses build tags to control which drivers are built into the service binary. The build tags have the same names as the drivers they are enabling.

The current list of driver build tags is:
* postgres
* mysql
* sqlite3
* sqlserver

For example, to build a binary with "postgres" and "mysql" support:
```
go build -tags "postgres,mysql"
```
