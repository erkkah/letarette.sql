//+build mysql

package adapter

import _ "github.com/go-sql-driver/mysql"

// user:password@/dbname
// Use ? for accessing bound parameters.

func init() {
	addDriver("mysql")
}
