//+build sqlserver

package adapter

import _ "github.com/denisenkom/go-mssqldb"

// sqlserver://username:password@host/instance?param1=value&param2=value
// Bound parameters are referenced by @p1, @p2, et.c.

func init() {
	addDriver("sqlserver")
}
