
LDFLAGS=$(./stamp.sh github.com/erkkah/letarette.sql/adapter)
go build -ldflags="${LDFLAGS}" -o lrsql -tags "postgres,mysql,sqlite3,sqlserver"
