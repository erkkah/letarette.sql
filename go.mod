module github.com/erkkah/letarette.sql

require (
	github.com/denisenkom/go-mssqldb v0.0.0-20190515213511-eb9f6a1743f3
	github.com/erkkah/letarette v1.0.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.0.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/segmentio/go-athena v0.0.0-20181208004937-dfa5f1818930
)

replace github.com/erkkah/letarette v1.0.0 => ../letarette

go 1.13
