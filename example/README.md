## letarette.sql example project

This is probably the smallest example possible of using **letarette.sql** to tie a document database to a Letarette service.

The document database in this case is a small SQLite database of articles with the following schema:

```sql
create table docs (
    id integer primary key,
    title text,
    txt text,
    updated integer
);
create index docs_updated on docs(updated);
```

Look at [indexrequest.sql](indexrequest.sql) and [documentrequest.sql](documentrequest.sql) for the corresponding *index* and *document* request queries to pass to **letarette.sql** for the full integration implementation.

### Creating the test database

A test database can created by the `loader.go` program, which loads the articles from an optionally gzip-compressed JSON object stream file. The JSON file should contain one JSON object per line, with objects of the following shape:

```json
{"title": "A nice title", "text": "This is the article text", "date": "2019-01-01T09:35:48.000Z"}
```

Each article will be given an auto-incremented ID.

For example, to load at most 1000 articles from `myarticles.json.gz` to the database `articles.db`, run:
```sh
go run loader.go -l 1000 articles.db myarticles.json.gz
```

### Running the SQL Document Manager

Now that we have a document database in place, `lrsql` can be started with the SQLite settings like this:

```sh
LRSQL_DB_DRIVER=sqlite3 LRSQL_DB_CONNECTION=articles.db lrsql
```

If there is a **Letarette** service connected to the default NATS server running on the local machine, it should start indexing the "docs" space with articles from our little database.

Set `LOG_LEVEL=DEBUG` to trace the indexing process.
